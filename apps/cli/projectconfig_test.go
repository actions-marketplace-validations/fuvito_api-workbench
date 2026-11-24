package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// project makes a folder that looks like a repository (so the search for a project file stops in
// it), writes the files and makes it the current directory for the test.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	return root
}

func writeServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writes.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &writes
}

func TestProjectConfig_ConformNeedsNoFlags(t *testing.T) {
	srv, writes := writeServer(t)
	project(t, map[string]string{
		"specs/ro.yaml": readOnlySpec,
		"clrsig.yaml":   "schemaVersion: \"1\"\nspec: specs/ro.yaml\nbaseUrl: " + srv.URL + "\nconform:\n  readOnly: true\n",
	})
	out, err := exec(t, "conform")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if writes.Load() != 0 || !strings.Contains(out, "SKIP  POST    /things") {
		t.Errorf("readOnly from the file must apply (writes=%d):\n%s", writes.Load(), out)
	}
	if !strings.Contains(out, "Using clrsig.yaml (spec, baseUrl, conform.readOnly from the file)") {
		t.Errorf("the run must say what it took from the file:\n%s", out)
	}
}

func TestProjectConfig_FlagsWinOverTheFile(t *testing.T) {
	srv, writes := writeServer(t)
	project(t, map[string]string{
		"specs/ro.yaml": readOnlySpec,
		"clrsig.yaml":   "schemaVersion: \"1\"\nspec: specs/ro.yaml\nbaseUrl: http://127.0.0.1:1\nconform:\n  readOnly: true\n",
	})
	out, err := exec(t, "conform", "--base-url", srv.URL, "--read-only=false")
	if exitCodeOf(err) != 1 { // the server answers 200 to everything, so the writes fail the spec check; they were still sent
		t.Fatalf("%v\n%s", err, out)
	}
	if writes.Load() == 0 {
		t.Errorf("--read-only=false must override readOnly: true:\n%s", out)
	}
	if strings.Contains(out, "baseUrl") || strings.Contains(out, "conform.readOnly") {
		t.Errorf("a value given as a flag is not reported as taken from the file:\n%s", out)
	}
	// a spec argument wins over spec:
	other := writeTempFile(t, "other.yaml", readOnlySpec)
	if out, err := exec(t, "conform", other, "--base-url", srv.URL); err != nil || strings.Contains(out, "spec,") {
		t.Errorf("an argument wins over spec: %v\n%s", err, out)
	}
}

func TestProjectConfig_PathsAreRelativeToTheFileNotTheCurrentFolder(t *testing.T) {
	srv, _ := writeServer(t)
	root := project(t, map[string]string{
		"specs/ro.yaml": readOnlySpec,
		"svc/.keep":     "",
		"clrsig.yaml":   "schemaVersion: \"1\"\nspec: specs/ro.yaml\nbaseUrl: " + srv.URL + "\n",
	})
	t.Chdir(filepath.Join(root, "svc"))
	out, err := exec(t, "conform", "--read-only")
	if err != nil {
		t.Fatalf("from a subfolder: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Using ") || !strings.Contains(out, "clrsig.yaml") {
		t.Errorf("%s", out)
	}
}

func TestProjectConfig_NoConfigAndExplicitConfig(t *testing.T) {
	srv, _ := writeServer(t)
	root := project(t, map[string]string{
		"specs/ro.yaml": readOnlySpec,
		"clrsig.yaml":   "schemaVersion: \"1\"\nspec: specs/ro.yaml\nbaseUrl: " + srv.URL + "\n",
		"other.json":    `{"schemaVersion":"1","spec":"specs/ro.yaml","baseUrl":"` + srv.URL + `"}`,
	})
	if _, err := exec(t, "conform", "--no-config"); err == nil || !strings.Contains(err.Error(), "conform needs a spec") {
		t.Errorf("--no-config ignores the file: %v", err)
	}
	if out, err := exec(t, "conform", "--config", filepath.Join(root, "other.json"), "--read-only"); err != nil || !strings.Contains(out, "other.json") {
		t.Errorf("--config chooses the file: %v\n%s", err, out)
	}
	if _, err := exec(t, "conform", "--config", "x.yaml", "--no-config"); err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("%v", err)
	}
	if _, err := exec(t, "conform", "--config", filepath.Join(root, "missing.yaml")); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("%v", err)
	}
}

func TestProjectConfig_BadFileIsAConfigurationErrorBeforeAnythingIsSent(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	defer srv.Close()
	project(t, map[string]string{
		"specs/ro.yaml": readOnlySpec,
		"clrsig.yaml":   "schemaVersion: \"1\"\nspec: specs/ro.yaml\nbaseUrl: " + srv.URL + "\nconform:\n  failOn: note\n",
	})
	_, err := exec(t, "conform")
	if err == nil || !strings.Contains(err.Error(), "clrsig.yaml") || !strings.Contains(err.Error(), "conform.failOn") {
		t.Fatalf("the error names the file and the key: %v", err)
	}
	if hits.Load() != 0 {
		t.Error("nothing is sent when the project file is wrong")
	}
}

func TestProjectConfig_RunAndValidateUseScenarios(t *testing.T) {
	srv := pingServer(t)
	root := project(t, map[string]string{
		"scenarios/ping.yaml":  scenarioYAML(srv.URL),
		"environments/ci.yaml": "schemaVersion: \"1\"\nname: ci\nvariables:\n  - {name: unused, value: x}\n",
		"clrsig.yaml":          "schemaVersion: \"1\"\nenvironment: environments/ci.yaml\nrun:\n  scenarios: [scenarios]\n  failFast: true\nreports:\n  - json=reports/run.json\n",
	})
	out, err := exec(t, "validate")
	if err != nil || !strings.Contains(out, "run.scenarios") {
		t.Fatalf("validate takes run.scenarios: %v\n%s", err, out)
	}
	out, err = exec(t, "run")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{"environment", "run.scenarios", "run.failFast", "reports"} {
		if !strings.Contains(out, want) {
			t.Errorf("the note should name %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "reports", "run.json")); err != nil {
		t.Errorf("the report from the file is written: %v", err)
	}
	// a scenario given on the command line replaces run.scenarios
	if out, err := exec(t, "run", filepath.Join("scenarios", "ping.yaml")); err != nil || strings.Contains(out, "run.scenarios") {
		t.Errorf("an argument wins over run.scenarios: %v\n%s", err, out)
	}
}

func TestProjectConfig_NothingHappensWithoutAFile(t *testing.T) {
	project(t, nil)
	if _, err := exec(t, "run"); err == nil || !strings.Contains(err.Error(), "run needs a scenario file or folder") {
		t.Errorf("%v", err)
	}
	if _, err := exec(t, "validate"); err == nil || !strings.Contains(err.Error(), "validate needs a scenario") {
		t.Errorf("%v", err)
	}
	if _, err := exec(t, "conform"); err == nil || !strings.Contains(err.Error(), "conform needs a spec") {
		t.Errorf("%v", err)
	}
}

func TestProjectConfig_AFileThatNamesNoSpecSaysWhere(t *testing.T) {
	project(t, map[string]string{"clrsig.yaml": "schemaVersion: \"1\"\n"})
	if _, err := exec(t, "conform"); err == nil || !strings.Contains(err.Error(), "set spec: in clrsig.yaml") {
		t.Errorf("%v", err)
	}
}
