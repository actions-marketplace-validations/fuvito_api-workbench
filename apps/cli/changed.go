package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// changedOperationScope compares the spec with its version at a git ref and returns the operations
// conform should check, with the description reports show. The spec must be a local file.
func changedOperationScope(src, ref string, spec *openapi.Spec) (map[string]bool, *conformance.Scope, error) {
	data, format, err := gitSpecAt("--changed-against", src, ref)
	if err != nil {
		return nil, nil, err
	}
	base, err := openapi.ParseBytes(data, format)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing the spec at %s: %w", ref, err)
	}
	changes, err := openapi.ChangedOperations(base, spec)
	if err != nil {
		return nil, nil, fmt.Errorf("comparing the spec with %s: %w", ref, err)
	}
	scope := &conformance.Scope{
		ChangedAgainst: ref,
		Checked:        []string{},
		Added:          changes.Added,
		Modified:       changes.Modified,
		Unchanged:      changes.Unchanged,
		Unattributed:   changes.Unattributed,
	}
	for _, op := range changes.Changed {
		scope.Checked = append(scope.Checked, op.String())
	}
	for _, op := range changes.Removed {
		scope.Removed = append(scope.Removed, op.String())
	}
	only := changes.Keys()
	if only == nil {
		only = map[string]bool{}
	}
	return only, scope, nil
}

// githubLocator points an annotation at an operation in the spec file, when the spec is a local file
// inside the current folder; for a URL or a file elsewhere the annotations stay on the run.
func githubLocator(src string, data []byte) reporting.Locator {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return nil
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	file := filepath.ToSlash(rel)
	lines := openapi.OperationLines(data)
	return func(method, specPath string) (string, int) {
		return file, lines[openapi.OperationKey(method, specPath)]
	}
}

// githubRuns prepares scenario runs for annotation: the file relative to the current folder (when it
// is inside it) and the line of each step.
func githubRuns(runs []*scenarioRun) []reporting.GitHubRun {
	cwd, _ := os.Getwd()
	out := make([]reporting.GitHubRun, 0, len(runs))
	for _, r := range runs {
		gr := reporting.GitHubRun{Result: r.result}
		if abs, err := filepath.Abs(r.path); err == nil && cwd != "" {
			if rel, err := filepath.Rel(cwd, abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				gr.File = filepath.ToSlash(rel)
				if data, err := os.ReadFile(abs); err == nil {
					gr.StepLines = scenario.StepLines(data)
				}
			}
		}
		out = append(out, gr)
	}
	return out
}
