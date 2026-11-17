package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// A --spec path is relative to the current directory, like every other CLI path.
func TestRunCmd_SpecFlagIsRelativeToCwd(t *testing.T) {
	srv := pingServer(t)
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"api.yaml":         minimalSpec,
		"scenarios/s.yaml": scenarioYAML(srv.URL),
	})
	t.Chdir(dir)

	out, err := exec(t, "run", "--spec", "api.yaml", "--coverage", filepath.Join("scenarios", "s.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(out, "could not load spec") {
		t.Errorf("--spec api.yaml should resolve from the current directory:\n%s", out)
	}
	if !strings.Contains(out, "Spec coverage:") {
		t.Errorf("want a coverage section, got:\n%s", out)
	}
}

// A specFile: written inside a scenario stays relative to the scenario file.
func TestRunCmd_ScenarioSpecFileIsRelativeToScenario(t *testing.T) {
	srv := pingServer(t)
	dir := t.TempDir()
	sc := strings.Replace(scenarioYAML(srv.URL), "title: \"ping\"", "title: \"ping\"\nspecFile: api.yaml", 1)
	writeFiles(t, dir, map[string]string{"api.yaml": minimalSpec, "s.yaml": sc})
	t.Chdir(t.TempDir())

	out, err := exec(t, "run", "--coverage", filepath.Join(dir, "s.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(out, "could not load spec") || !strings.Contains(out, "Spec coverage:") {
		t.Errorf("specFile: inside a scenario must resolve next to the scenario:\n%s", out)
	}
}

func TestBundleRun_SpecFlagIsRelativeToCwd(t *testing.T) {
	srv := pingServer(t)
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"api.yaml":       minimalSpec,
		"bundles/b.yaml": bundleFor(srv.URL),
	})
	t.Chdir(dir)

	out, _ := execBundle(t, "bundle", "run", "--spec", "api.yaml", filepath.Join("bundles", "b.yaml"))
	if strings.Contains(out, "could not load spec") {
		t.Errorf("--spec api.yaml should resolve from the current directory:\n%s", out)
	}
}

func TestCwdSpecPath(t *testing.T) {
	if got := cwdSpecPath(""); got != "" {
		t.Errorf("empty stays empty, got %q", got)
	}
	for _, keep := range []string{"https://example.com/openapi.yaml", "http://localhost/o.json", filepath.Join(t.TempDir(), "x.yaml")} {
		if got := cwdSpecPath(keep); got != keep {
			t.Errorf("%q should be unchanged, got %q", keep, got)
		}
	}
	got := cwdSpecPath("api.yaml")
	if !filepath.IsAbs(got) || filepath.Base(got) != "api.yaml" {
		t.Errorf("relative path should become absolute, got %q", got)
	}
}
