// Package coverage provides workspace-scoped accumulative spec coverage tracking.
// Coverage data is persisted in {workspace}/.clrsig-coverage/ as JSON files,
// one per spec file or URL. Both the desktop app and CLI write to the same store.
package coverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// StoredCoverage is the on-disk format for a single spec's accumulated data.
type StoredCoverage struct {
	SpecFile   string    `json:"specFile"`
	Exercised  []string  `json:"exercised"`
	TotalOps   int       `json:"totalOps"` // cached so list views don't reload the spec
	TotalRuns  int       `json:"totalRuns"`
	PassedRuns int       `json:"passedRuns"`
	FailedRuns int       `json:"failedRuns"`
	LastRunAt  time.Time `json:"lastRunAt"`
}

// Report is the computed accumulated coverage view for a single spec.
type Report struct {
	SpecFile   string
	Covered    int
	Total      int
	Percent    float64
	TotalRuns  int
	PassedRuns int
	FailedRuns int
	LastRunAt  time.Time
	Exercised  []string
	Missed     []string // nil when built from Summary(); populated by BuildReport
}

// Key converts a specFile value (URL or path) into a safe, unique filename.
func Key(specFile string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(specFile) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := b.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if len(s) > 80 {
		s = s[len(s)-80:]
	}
	return s + ".json"
}

// DirPath returns the coverage directory for the given workspace.
func DirPath(workspacePath string) string {
	return filepath.Join(workspacePath, ".clrsig-coverage")
}

// Load reads the stored coverage for specFile from workspacePath.
// Returns a zero-value StoredCoverage (with SpecFile set) if no data exists.
func Load(workspacePath, specFile string) StoredCoverage {
	path := filepath.Join(DirPath(workspacePath), Key(specFile))
	data, err := os.ReadFile(path)
	if err != nil {
		return StoredCoverage{SpecFile: specFile}
	}
	var cov StoredCoverage
	if err := json.Unmarshal(data, &cov); err != nil {
		return StoredCoverage{SpecFile: specFile}
	}
	return cov
}

// Save atomically writes cov to the workspace coverage store.
func Save(workspacePath string, cov StoredCoverage) error {
	if workspacePath == "" {
		return nil
	}
	dir := DirPath(workspacePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cov, "", "  ")
	if err != nil {
		return err
	}
	dst := filepath.Join(dir, Key(cov.SpecFile))
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// Reset deletes the stored coverage data for specFile in workspacePath.
func Reset(workspacePath, specFile string) error {
	path := filepath.Join(DirPath(workspacePath), Key(specFile))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// MergeExercised merges two slices of "METHOD /path" strings into a sorted, deduplicated result.
func MergeExercised(existing, newOps []string) []string {
	seen := make(map[string]bool, len(existing)+len(newOps))
	for _, op := range existing {
		seen[op] = true
	}
	for _, op := range newOps {
		seen[op] = true
	}
	merged := make([]string, 0, len(seen))
	for op := range seen {
		merged = append(merged, op)
	}
	sort.Strings(merged)
	return merged
}

// Summary builds a Report from cached data only — no spec file is loaded.
// The Missed field is nil; use BuildReport when you have the full op list.
func (sc StoredCoverage) Summary() Report {
	covered := len(sc.Exercised)
	total := sc.TotalOps
	if covered > total {
		covered = total
	}
	var pct float64
	if total > 0 {
		pct = float64(covered) / float64(total) * 100
	}
	ex := sc.Exercised
	if ex == nil {
		ex = []string{}
	}
	return Report{
		SpecFile:   sc.SpecFile,
		Covered:    covered,
		Total:      total,
		Percent:    pct,
		TotalRuns:  sc.TotalRuns,
		PassedRuns: sc.PassedRuns,
		FailedRuns: sc.FailedRuns,
		LastRunAt:  sc.LastRunAt,
		Exercised:  ex,
	}
}

// BuildReport computes the full report including Missed operations.
// allOps is every operation declared in the spec, expressed as "METHOD /path" strings
// (e.g. from spec.AllOperations() converted via op.String()).
func BuildReport(stored StoredCoverage, allOps []string) Report {
	exercisedSet := make(map[string]bool, len(stored.Exercised))
	for _, op := range stored.Exercised {
		exercisedSet[op] = true
	}
	var missed []string
	for _, op := range allOps {
		if !exercisedSet[op] {
			missed = append(missed, op)
		}
	}
	total := len(allOps)
	covered := total - len(missed)
	var pct float64
	if total > 0 {
		pct = float64(covered) / float64(total) * 100
	}
	ex := stored.Exercised
	if ex == nil {
		ex = []string{}
	}
	return Report{
		SpecFile:   stored.SpecFile,
		Covered:    covered,
		Total:      total,
		Percent:    pct,
		TotalRuns:  stored.TotalRuns,
		PassedRuns: stored.PassedRuns,
		FailedRuns: stored.FailedRuns,
		LastRunAt:  stored.LastRunAt,
		Exercised:  ex,
		Missed:     missed,
	}
}

// ListAll scans the coverage directory in workspacePath and returns all entries
// as summary Reports, sorted by most recently run first.
func ListAll(workspacePath string) []Report {
	entries, err := os.ReadDir(DirPath(workspacePath))
	if err != nil {
		return nil
	}
	var result []Report
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(DirPath(workspacePath), e.Name()))
		if err != nil {
			continue
		}
		var sc StoredCoverage
		if err := json.Unmarshal(data, &sc); err != nil {
			continue
		}
		if sc.TotalRuns == 0 {
			continue
		}
		result = append(result, sc.Summary())
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastRunAt.After(result[j].LastRunAt)
	})
	return result
}
