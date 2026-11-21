package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSpecWS(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const specScenario = "schemaVersion: \"1\"\ntitle: S\nspecFile: %s\nsteps:\n  - label: One\n    request: {method: GET, url: \"http://x/a\"}\n"

func TestLoadWorkspaceSpec_RelativeToTheScenarioThatNamesIt(t *testing.T) {
	root := t.TempDir()
	writeSpecWS(t, root, "specs/api.yaml", "openapi: 3.0.0")
	writeSpecWS(t, root, "scenarios/a/deep/s.yaml", strings.Replace(specScenario, "%s", "../../../specs/api.yaml", 1))
	writeSpecWS(t, root, "scenarios/top.yaml", strings.Replace(specScenario, "%s", "../specs/api.yaml", 1))

	for _, ref := range []string{"../../../specs/api.yaml", "../specs/api.yaml"} {
		data, format, err := loadWorkspaceSpec(root, ref)
		if err != nil || !strings.Contains(string(data), "openapi") || format != "yaml" {
			t.Errorf("%s: %q %q %v", ref, data, format, err)
		}
	}
}

func TestLoadWorkspaceSpec_FallsBackToTheWorkspaceRootAndAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	writeSpecWS(t, root, "specs/api.yaml", "openapi: 3.0.0")
	if _, _, err := loadWorkspaceSpec(root, "specs/api.yaml"); err != nil {
		t.Errorf("a path relative to the workspace root: %v", err)
	}
	if _, _, err := loadWorkspaceSpec(root, filepath.Join(root, "specs", "api.yaml")); err != nil {
		t.Errorf("an absolute path: %v", err)
	}
}

func TestLoadWorkspaceSpec_SaysWhatItTriedWhenTheFileIsMissing(t *testing.T) {
	root := t.TempDir()
	_, _, err := loadWorkspaceSpec(root, "../specs/gone.yaml")
	if err == nil || !strings.Contains(err.Error(), `"../specs/gone.yaml"`) || !strings.Contains(err.Error(), "relative to the scenarios that use it") {
		t.Fatalf("error should name the spec and where it was looked for: %v", err)
	}
}
