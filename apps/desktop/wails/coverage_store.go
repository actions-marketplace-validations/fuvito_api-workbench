package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// AccumulatedCoverageIO is the accumulated spec coverage across all runs in a workspace.
type AccumulatedCoverageIO struct {
	SpecFile   string   `json:"specFile"`
	Covered    int      `json:"covered"`
	Total      int      `json:"total"`
	Percent    float64  `json:"percent"`
	TotalRuns  int      `json:"totalRuns"`
	PassedRuns int      `json:"passedRuns"`
	FailedRuns int      `json:"failedRuns"`
	LastRunAt  string   `json:"lastRunAt,omitempty"`
	Exercised  []string `json:"exercised"`
	Missed     []string `json:"missed"`
}

// storedSpecCoverage is the on-disk format for a single spec's accumulated data.
type storedSpecCoverage struct {
	SpecFile   string    `json:"specFile"`
	Exercised  []string  `json:"exercised"`
	TotalRuns  int       `json:"totalRuns"`
	PassedRuns int       `json:"passedRuns"`
	FailedRuns int       `json:"failedRuns"`
	LastRunAt  time.Time `json:"lastRunAt"`
}

// coverageKey turns a specFile value (URL or path) into a safe filename.
func coverageKey(specFile string) string {
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

func coverageDirPath(workspacePath string) string {
	return filepath.Join(workspacePath, ".clrsig-coverage")
}

func loadStoredCoverage(workspacePath, specFile string) storedSpecCoverage {
	path := filepath.Join(coverageDirPath(workspacePath), coverageKey(specFile))
	data, err := os.ReadFile(path)
	if err != nil {
		return storedSpecCoverage{SpecFile: specFile}
	}
	var cov storedSpecCoverage
	if err := json.Unmarshal(data, &cov); err != nil {
		return storedSpecCoverage{SpecFile: specFile}
	}
	return cov
}

func saveStoredCoverage(workspacePath string, cov storedSpecCoverage) error {
	if workspacePath == "" {
		return nil
	}
	dir := coverageDirPath(workspacePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cov, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, coverageKey(cov.SpecFile)), data)
}

// mergeExercised merges two slices of "METHOD /path" strings into a sorted, deduplicated result.
func mergeExercised(existing, newOps []string) []string {
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

// buildAccumulatedCoverageIO computes the full accumulated picture by comparing stored
// exercised operations against the spec's declared operations.
func buildAccumulatedCoverageIO(stored storedSpecCoverage, spec *openapi.Spec) AccumulatedCoverageIO {
	exercisedSet := make(map[string]bool, len(stored.Exercised))
	for _, op := range stored.Exercised {
		exercisedSet[op] = true
	}

	all := spec.AllOperations()
	var missed []string
	for _, op := range all {
		if !exercisedSet[op.String()] {
			missed = append(missed, op.String())
		}
	}

	total := len(all)
	covered := total - len(missed)
	var pct float64
	if total > 0 {
		pct = float64(covered) / float64(total) * 100
	}

	lastRun := ""
	if !stored.LastRunAt.IsZero() {
		lastRun = stored.LastRunAt.Format(time.RFC3339)
	}

	return AccumulatedCoverageIO{
		SpecFile:   stored.SpecFile,
		Covered:    covered,
		Total:      total,
		Percent:    pct,
		TotalRuns:  stored.TotalRuns,
		PassedRuns: stored.PassedRuns,
		FailedRuns: stored.FailedRuns,
		LastRunAt:  lastRun,
		Exercised:  stored.Exercised,
		Missed:     missed,
	}
}
