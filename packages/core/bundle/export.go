package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// SecretPlaceholder replaces secret variable values in an export.
const SecretPlaceholder = "CHANGE_ME"

// DropPlaceholders returns env without the secret variables that still hold SecretPlaceholder, and
// their names. The placeholder stands for a value the export left out, so it must not be sent as one.
func DropPlaceholders(env *domain.Environment) (*domain.Environment, []string) {
	if env == nil {
		return nil, nil
	}
	out := *env
	out.Variables = nil
	var dropped []string
	for _, v := range env.Variables {
		if v.Secret && v.Value == SecretPlaceholder {
			dropped = append(dropped, v.Name)
			continue
		}
		out.Variables = append(out.Variables, v)
	}
	if len(dropped) == 0 {
		return env, nil
	}
	return &out, dropped
}

// ExportOptions control what an export contains.
type ExportOptions struct {
	// KeepSecrets writes secret variable values as they are. By default they are replaced
	// with SecretPlaceholder, because a bundle is usually pasted into a chat or shared.
	KeepSecrets bool
}

// SkippedFile is a workspace file that could not be exported.
type SkippedFile struct {
	Path   string // workspace-relative, "/"-separated
	Reason string
}

// ExportResult is a bundle built from a workspace, plus what was left out or changed.
type ExportResult struct {
	Bundle          *serialization.Bundle
	Skipped         []SkippedFile
	Warnings        []string
	RedactedSecrets int
}

// Marshal encodes the exported bundle as YAML.
func (r *ExportResult) Marshal() ([]byte, error) {
	return serialization.WriteBundle(r.Bundle, "yaml")
}

// Export gathers the environments, requests and scenarios of a workspace into one bundle.
// Files that cannot be parsed are skipped and reported rather than failing the export.
// Scenario requestRefs that point at an exported request are rewritten to that request's
// bundle name, so the result imports cleanly into another workspace.
func Export(workspaceDir string, opts ExportOptions) (*ExportResult, error) {
	res := &ExportResult{Bundle: &serialization.Bundle{
		SchemaVersion: "1",
		Title:         filepath.Base(workspaceDir),
	}}
	b := res.Bundle

	// Environments live directly in environments/.
	envDir := filepath.Join(workspaceDir, "environments")
	if entries, err := os.ReadDir(envDir); err == nil {
		usedKeys := map[string]bool{}
		for _, e := range entries {
			if e.IsDir() || isHidden(e.Name()) || !isDataFile(e.Name()) {
				continue
			}
			rel := "environments/" + e.Name()
			data, err := os.ReadFile(filepath.Join(envDir, e.Name()))
			if err != nil {
				res.Skipped = append(res.Skipped, SkippedFile{rel, err.Error()})
				continue
			}
			env, err := serialization.ParseEnvironmentFile(data, e.Name())
			if err != nil {
				res.Skipped = append(res.Skipped, SkippedFile{rel, err.Error()})
				continue
			}
			key := uniqueKey(stripDataExt(e.Name()), usedKeys)
			if env.Name == "" {
				env.Name = key
			}
			if !opts.KeepSecrets {
				for i := range env.Variables {
					if env.Variables[i].Secret && env.Variables[i].Value != "" {
						env.Variables[i].Value = SecretPlaceholder
						res.RedactedSecrets++
					}
				}
			}
			b.Environments = append(b.Environments, serialization.BundleEnvironment{File: key, Environment: env})
		}
	}

	// Requests, recursively. exportedRequests maps absolute path -> bundle key.
	reqDir := filepath.Join(workspaceDir, "requests")
	exportedRequests := map[string]string{}
	usedReq := map[string]bool{}
	walkDataFiles(reqDir, func(abs, rel string) {
		display := "requests/" + rel
		data, err := os.ReadFile(abs)
		if err != nil {
			res.Skipped = append(res.Skipped, SkippedFile{display, err.Error()})
			return
		}
		req, err := serialization.ParseRequestFile(data, abs)
		if err != nil {
			res.Skipped = append(res.Skipped, SkippedFile{display, err.Error()})
			return
		}
		key := uniqueKey(stripDataExt(rel), usedReq)
		exportedRequests[filepath.Clean(abs)] = key
		b.Requests = append(b.Requests, serialization.BundleRequest{File: key, Request: req})
	})

	// Scenarios, recursively.
	scnDir := filepath.Join(workspaceDir, "scenarios")
	usedScn := map[string]bool{}
	walkDataFiles(scnDir, func(abs, rel string) {
		display := "scenarios/" + rel
		data, err := os.ReadFile(abs)
		if err != nil {
			res.Skipped = append(res.Skipped, SkippedFile{display, err.Error()})
			return
		}
		sc, err := serialization.ParseScenarioFile(data, abs)
		if err != nil {
			res.Skipped = append(res.Skipped, SkippedFile{display, err.Error()})
			return
		}
		for i := range sc.Steps {
			ref := sc.Steps[i].RequestRef
			if ref == "" {
				continue
			}
			if key, ok := resolveRequestRef(ref, filepath.Dir(abs), reqDir, exportedRequests); ok {
				sc.Steps[i].RequestRef = key
			} else {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: requestRef %q does not point at an exported request, so it was left unchanged", display, ref))
			}
		}
		b.Scenarios = append(b.Scenarios, serialization.BundleScenario{File: uniqueKey(stripDataExt(rel), usedScn), Scenario: sc})
	})

	if len(b.Environments)+len(b.Requests)+len(b.Scenarios) == 0 {
		return res, fmt.Errorf("the workspace has no environments, requests or scenarios to export")
	}

	sort.Slice(b.Environments, func(i, j int) bool { return b.Environments[i].File < b.Environments[j].File })
	sort.Slice(b.Requests, func(i, j int) bool { return b.Requests[i].File < b.Requests[j].File })
	sort.Slice(b.Scenarios, func(i, j int) bool { return b.Scenarios[i].File < b.Scenarios[j].File })

	hoistCommonSpec(b)
	return res, nil
}

// hoistCommonSpec moves a specFile shared by every scenario up to the bundle.
func hoistCommonSpec(b *serialization.Bundle) {
	if len(b.Scenarios) == 0 {
		return
	}
	common := b.Scenarios[0].Scenario.SpecFile
	if common == "" {
		return
	}
	for _, s := range b.Scenarios[1:] {
		if s.Scenario.SpecFile != common {
			return
		}
	}
	b.SpecFile = common
	for _, s := range b.Scenarios {
		s.Scenario.SpecFile = ""
	}
}

// resolveRequestRef finds the exported request a scenario's requestRef points at, using
// the same lookup order as the scenario runner: relative to the scenario's folder first,
// then relative to the workspace's requests folder.
func resolveRequestRef(ref, scenarioDir, requestsDir string, exported map[string]string) (string, bool) {
	candidates := []string{filepath.Join(scenarioDir, ref), filepath.Join(requestsDir, ref)}
	if filepath.IsAbs(ref) {
		candidates = []string{ref}
	}
	for _, c := range candidates {
		if key, ok := exported[filepath.Clean(c)]; ok {
			return key, true
		}
	}
	return "", false
}

func walkDataFiles(root string, visit func(abs, rel string)) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if p != root && isHidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !isDataFile(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		visit(p, filepath.ToSlash(rel))
		return nil
	})
}

func isHidden(name string) bool { return strings.HasPrefix(name, ".") }

func isDataFile(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

func stripDataExt(name string) string {
	ext := path.Ext(name)
	if isDataFile(name) {
		return strings.TrimSuffix(name, ext)
	}
	return name
}

// uniqueKey keeps bundle file names unique when two workspace files share a stem
// (for example a.yaml and a.json).
func uniqueKey(key string, used map[string]bool) string {
	candidate := key
	for n := 2; used[candidate]; n++ {
		candidate = fmt.Sprintf("%s-%d", key, n)
	}
	used[candidate] = true
	return candidate
}
