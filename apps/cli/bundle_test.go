package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

// execBundle runs a `clrsig bundle ...` command and returns stdout and stderr together.
func execBundle(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "clrsig", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(bundleCmd(), runCmd())
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// bundleFor returns a bundle with two scenarios against baseURL: "Healthy" passes and
// "Broken" expects a status the server never returns.
func bundleFor(baseURL string) string {
	return fmt.Sprintf(`schemaVersion: "1"
title: CLI suite
environments:
  - name: local
    variables:
      - {name: baseUrl, value: "%s"}
      - {name: apiKey, value: real-key, secret: true}
requests:
  - file: ping
    method: GET
    url: "{{baseUrl}}/ping"
scenarios:
  - title: Healthy
    file: healthy
    steps:
      - label: Ping
        requestRef: ping
        assertions:
          - {type: status, operator: eq, expected: 200}
          - {type: jsonpath, path: "$.status", operator: eq, expected: ok}
      - label: Again
        request: {method: GET, url: "{{baseUrl}}/again"}
        assertions:
          - {type: status, operator: eq, expected: 200}
  - title: Broken
    file: broken
    steps:
      - label: Wrong expectation
        request: {method: GET, url: "{{baseUrl}}/x"}
        assertions:
          - {type: status, operator: eq, expected: 404}
`, baseURL)
}

func TestBundleRun_AllPass(t *testing.T) {
	srv := pingServer(t)
	path := writeTempFile(t, "b.yaml", strings.Replace(bundleFor(srv.URL), "expected: 404", "expected: 200", 1))
	out, err := execBundle(t, "bundle", "run", path)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{"Bundle: CLI suite  (2 scenarios)", "Healthy", "Broken", "Bundle PASSED  2/2 scenarios passed  (3 steps, 4/4 assertions passed)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestBundleRun_FailureExitsWithOne(t *testing.T) {
	srv := pingServer(t)
	out, err := execBundle(t, "bundle", "run", writeTempFile(t, "b.yaml", bundleFor(srv.URL)))
	if exitCode(err) != 1 {
		t.Fatalf("a failing scenario should exit 1, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Bundle FAILED  1/2 scenarios passed  (3 steps, 3/4 assertions passed)") {
		t.Errorf("summary:\n%s", out)
	}
	if !strings.Contains(out, "Healthy") || !strings.Contains(out, "Broken") {
		t.Error("every scenario runs even when an earlier one passes or fails")
	}
}

func TestBundleRun_JSONAndJUnitReports(t *testing.T) {
	srv := pingServer(t)
	path := writeTempFile(t, "b.yaml", bundleFor(srv.URL))

	out, err := execBundle(t, "bundle", "run", path, "-f", "json")
	if exitCode(err) != 1 {
		t.Fatal(err)
	}
	var js struct {
		Bundle    string            `json:"bundle"`
		Passed    bool              `json:"passed"`
		Scenarios []json.RawMessage `json:"scenarios"`
	}
	if err := json.Unmarshal([]byte(out), &js); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if js.Bundle != "CLI suite" || js.Passed || len(js.Scenarios) != 2 {
		t.Errorf("json: %+v", js)
	}

	out, err = execBundle(t, "bundle", "run", path, "-f", "junit")
	if exitCode(err) != 1 {
		t.Fatal(err)
	}
	var ju struct {
		Tests    int `xml:"tests,attr"`
		Failures int `xml:"failures,attr"`
		Suites   []struct {
			Name string `xml:"name,attr"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal([]byte(out), &ju); err != nil {
		t.Fatalf("not XML: %v\n%s", err, out)
	}
	if ju.Tests != 3 || ju.Failures != 1 || len(ju.Suites) != 2 || ju.Suites[0].Name != "Healthy" {
		t.Errorf("junit: %+v", ju)
	}
}

func TestBundleRun_WritesReportToFile(t *testing.T) {
	srv := pingServer(t)
	path := writeTempFile(t, "b.yaml", bundleFor(srv.URL))
	dest := filepath.Join(t.TempDir(), "reports", "out.json")
	out, _ := execBundle(t, "bundle", "run", path, "-f", "json", "-o", dest)
	if strings.Contains(out, "scenarios") {
		t.Errorf("the report should go to the file, not the terminal:\n%s", out)
	}
	data, err := os.ReadFile(dest)
	if err != nil || !json.Valid(data) {
		t.Errorf("report file: %v %q", err, data)
	}
}

func TestBundleRun_ScenarioFilter(t *testing.T) {
	srv := pingServer(t)
	path := writeTempFile(t, "b.yaml", bundleFor(srv.URL))

	out, err := execBundle(t, "bundle", "run", path, "--scenario", "healthy")
	if err != nil || !strings.Contains(out, "1/1 scenarios passed") || strings.Contains(out, "Wrong expectation") {
		t.Errorf("filter by file name should run only that scenario: %v\n%s", err, out)
	}
	out, err = execBundle(t, "bundle", "run", path, "--scenario", "Broken", "--scenario", "Healthy")
	if exitCode(err) != 1 || !strings.Contains(out, "0/2") && !strings.Contains(out, "1/2") {
		t.Errorf("filter by title, repeated: %v\n%s", err, out)
	}
	if _, err := execBundle(t, "bundle", "run", path, "--scenario", "nope"); err == nil || !strings.Contains(err.Error(), `"nope"`) && !strings.Contains(err.Error(), "nope") {
		t.Errorf("an unknown scenario must be an error: %v", err)
	}
}

func TestBundleRun_EnvironmentSelection(t *testing.T) {
	srv := pingServer(t)
	two := strings.Replace(bundleFor(srv.URL), "environments:\n", "environments:\n  - name: other\n    variables:\n      - {name: baseUrl, value: \"http://127.0.0.1:1\"}\n", 1)
	two = strings.Replace(two, "expected: 404", "expected: 200", 1)
	path := writeTempFile(t, "b.yaml", two)

	if _, err := execBundle(t, "bundle", "run", path); err == nil || !strings.Contains(err.Error(), "several environments") {
		t.Errorf("a choice is required: %v", err)
	}
	if out, err := execBundle(t, "bundle", "run", path, "--env", "LOCAL"); err != nil {
		t.Errorf("selecting by name (any case): %v\n%s", err, out)
	}
	if _, err := execBundle(t, "bundle", "run", path, "--env", "nope"); err == nil || !strings.Contains(err.Error(), "available") {
		t.Errorf("unknown environment: %v", err)
	}

	envPath := writeTempFile(t, "env.yaml", fmt.Sprintf("schemaVersion: \"1\"\nname: ext\nvariables:\n  - {name: baseUrl, value: \"%s\"}\n", srv.URL))
	if out, err := execBundle(t, "bundle", "run", path, "--env-file", envPath); err != nil {
		t.Errorf("an environment file replaces the bundle's: %v\n%s", err, out)
	}
	if _, err := execBundle(t, "bundle", "run", path, "--env-file", "/no/such/env.yaml"); err == nil {
		t.Error("missing env file")
	}
}

func TestBundleRun_InvalidBundleIsNotRun(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()

	bad := strings.Replace(bundleFor(srv.URL), "requestRef: ping", "requestRef: missing", 1)
	out, err := execBundle(t, "bundle", "run", writeTempFile(t, "b.yaml", bad))
	if err == nil || exitCode(err) == 1 || !strings.Contains(err.Error(), "error(s)") {
		t.Errorf("validation errors are a usage error (exit 2), not a test failure: %v", err)
	}
	if !strings.Contains(out, `error: scenarios[0] "Healthy" › step 1 "Ping"`) {
		t.Errorf("errors should say where:\n%s", out)
	}
	if hits.Load() != 0 {
		t.Error("nothing may be sent when the bundle is invalid")
	}

	for name, path := range map[string]string{
		"missing file": filepath.Join(t.TempDir(), "none.yaml"),
		"bad yaml":     writeTempFile(t, "bad.yaml", "a: ["),
		"unknown key":  writeTempFile(t, "key.yaml", "schemaVersion: \"1\"\nscenarioz: []\n"),
	} {
		if _, err := execBundle(t, "bundle", "run", path); err == nil || exitCode(err) == 1 {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestBundleRun_ResolvesExternalRequestRefsNextToTheBundle(t *testing.T) {
	srv := pingServer(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "external.yaml"), []byte(fmt.Sprintf("method: GET\nurl: %s/external\n", srv.URL)), 0o644)
	doc := `schemaVersion: "1"
scenarios:
  - title: Uses a file
    steps:
      - {label: Ext, requestRef: external.yaml, assertions: [{type: status, operator: eq, expected: 200}]}
`
	path := filepath.Join(dir, "b.yaml")
	os.WriteFile(path, []byte(doc), 0o644)
	out, err := execBundle(t, "bundle", "run", path)
	if err != nil || !strings.Contains(out, "1/1 scenarios passed") {
		t.Errorf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "warning") {
		t.Log("(the ref is outside the bundle, which validation reports as a warning)")
	}

	missing := strings.Replace(doc, "external.yaml", "gone.yaml", 1)
	os.WriteFile(path, []byte(missing), 0o644)
	out, err = execBundle(t, "bundle", "run", path)
	if exitCode(err) != 1 || !strings.Contains(out, `requestRef "gone.yaml" could not be loaded`) {
		t.Errorf("an unloadable file is reported and the scenario fails: %v\n%s", err, out)
	}
}

func TestBundleRun_LoadsASharedSpecOnce(t *testing.T) {
	var specHits atomic.Int32
	spec := `openapi: "3.0.3"
info: {title: t, version: "1"}
paths:
  /ping:
    get:
      operationId: getPing
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [status]
                properties:
                  status: {type: string}
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.yaml" {
			specHits.Add(1)
			fmt.Fprint(w, spec)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	doc := fmt.Sprintf(`schemaVersion: "1"
specFile: %s/openapi.yaml
scenarios:
  - title: One
    steps:
      - {label: a, request: {method: GET, url: "%s/ping"}, assertions: [{type: schema, schema: getPing}]}
  - title: Two
    steps:
      - {label: b, request: {method: GET, url: "%s/ping"}}
`, srv.URL, srv.URL, srv.URL)
	out, err := execBundle(t, "bundle", "run", writeTempFile(t, "b.yaml", doc))
	if err != nil || !strings.Contains(out, "2/2 scenarios passed") {
		t.Fatalf("%v\n%s", err, out)
	}
	if specHits.Load() != 1 {
		t.Errorf("a spec shared by scenarios should be fetched once, got %d", specHits.Load())
	}

	// --spec replaces the bundle's spec for every scenario.
	specHits.Store(0)
	out, err = execBundle(t, "bundle", "run", writeTempFile(t, "b2.yaml", strings.Replace(doc, srv.URL+"/openapi.yaml", "http://127.0.0.1:1/nope.yaml", 1)), "--spec", srv.URL+"/openapi.yaml")
	if err != nil || specHits.Load() != 1 {
		t.Errorf("--spec override: %v hits=%d\n%s", err, specHits.Load(), out)
	}
}

func TestBundleRun_UnusableSpecIsAWarningNotAFailure(t *testing.T) {
	srv := pingServer(t)
	doc := fmt.Sprintf("schemaVersion: \"1\"\nspecFile: http://127.0.0.1:1/openapi.yaml\nscenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GET, url: \"%s/x\"}}\n", srv.URL)
	out, err := execBundle(t, "bundle", "run", writeTempFile(t, "b.yaml", doc))
	if err != nil || !strings.Contains(out, "warning: could not load spec") || !strings.Contains(out, "1/1 scenarios passed") {
		t.Errorf("%v\n%s", err, out)
	}
}

func TestBundleValidate(t *testing.T) {
	out, err := execBundle(t, "bundle", "validate", writeTempFile(t, "ok.yaml", bundleFor("http://x")))
	if err != nil || !strings.Contains(out, "Bundle is valid: 1 environment(s), 1 request(s), 2 scenario(s), 0 warning(s)") {
		t.Errorf("%v\n%s", err, out)
	}

	warn := `schemaVersion: "1"
environments:
  - {name: E, variables: [{name: baseUrl, value: "http://x"}]}
scenarios:
  - title: S
    steps:
      - {label: a, request: {method: GET, url: "{{baseUrl}}/{{ghost}}"}}
`
	out, err = execBundle(t, "bundle", "validate", writeTempFile(t, "w.yaml", warn))
	if err != nil || !strings.Contains(out, "warning: ") || !strings.Contains(out, "{{ghost}}") || !strings.Contains(out, "1 warning(s)") {
		t.Errorf("warnings are printed but do not fail: %v\n%s", err, out)
	}

	bad := strings.Replace(bundleFor("http://x"), "requestRef: ping", "requestRef: gone", 1)
	out, err = execBundle(t, "bundle", "validate", writeTempFile(t, "bad.yaml", bad))
	if exitCode(err) != 1 || !strings.Contains(out, "Bundle is invalid: 1 error(s)") {
		t.Errorf("errors exit 1: %v\n%s", err, out)
	}
	if _, err := execBundle(t, "bundle", "validate", filepath.Join(t.TempDir(), "none.yaml")); err == nil || exitCode(err) == 1 {
		t.Errorf("an unreadable file is a usage error: %v", err)
	}
}

func TestBundleImport(t *testing.T) {
	ws := t.TempDir()
	path := writeTempFile(t, "b.yaml", bundleFor("http://x"))

	out, err := execBundle(t, "bundle", "import", path, "-w", ws, "--dry-run")
	if err != nil || !strings.Contains(out, "Dry run: 4 file(s) would be written") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(ws, "scenarios")); !os.IsNotExist(err) {
		t.Fatal("a dry run must not write")
	}

	out, err = execBundle(t, "bundle", "import", path, "-w", ws)
	if err != nil || !strings.Contains(out, "Imported: 4 created, 0 renamed, 0 overwritten, 0 skipped") {
		t.Fatalf("%v\n%s", err, out)
	}
	scn, _ := os.ReadFile(filepath.Join(ws, "scenarios", "healthy.yaml"))
	if !strings.Contains(string(scn), "requestRef: ../requests/ping.yaml") {
		t.Errorf("requestRef should point at the saved request:\n%s", scn)
	}

	out, _ = execBundle(t, "bundle", "import", path, "-w", ws)
	if !strings.Contains(out, "4 renamed") || !strings.Contains(out, "(wanted scenarios/healthy.yaml)") {
		t.Errorf("a second import renames:\n%s", out)
	}
	out, _ = execBundle(t, "bundle", "import", path, "-w", ws, "--on-conflict", "skip")
	if !strings.Contains(out, "4 skipped") {
		t.Errorf("skip:\n%s", out)
	}
	out, _ = execBundle(t, "bundle", "import", path, "-w", ws, "--on-conflict", "overwrite")
	if !strings.Contains(out, "4 overwritten") {
		t.Errorf("overwrite:\n%s", out)
	}

	if _, err := execBundle(t, "bundle", "import", path); err == nil || !strings.Contains(err.Error(), "--workspace") {
		t.Errorf("workspace is required: %v", err)
	}
	if _, err := execBundle(t, "bundle", "import", path, "-w", ws, "--on-conflict", "merge"); err == nil {
		t.Error("unknown strategy")
	}
	bad := writeTempFile(t, "bad.yaml", strings.Replace(bundleFor("http://x"), "requestRef: ping", "requestRef: gone", 1))
	fresh := t.TempDir()
	if _, err := execBundle(t, "bundle", "import", bad, "-w", fresh); err == nil || !strings.Contains(err.Error(), "not imported") {
		t.Errorf("an invalid bundle must not import: %v", err)
	}
	if entries, _ := os.ReadDir(fresh); len(entries) != 0 {
		t.Error("nothing may be written for an invalid bundle")
	}
}

func TestBundleImport_SpecFileOverride(t *testing.T) {
	ws := t.TempDir()
	path := writeTempFile(t, "b.yaml", bundleFor("http://x"))
	if _, err := execBundle(t, "bundle", "import", path, "-w", ws, "-s", "C:/specs/api.yaml"); err != nil {
		t.Fatal(err)
	}
	scn, _ := os.ReadFile(filepath.Join(ws, "scenarios", "healthy.yaml"))
	if !strings.Contains(string(scn), "specFile: C:/specs/api.yaml") {
		t.Errorf("override:\n%s", scn)
	}
}

func TestBundleExport_ThenRunTheExport(t *testing.T) {
	srv := pingServer(t)
	ws := t.TempDir()
	path := writeTempFile(t, "b.yaml", strings.Replace(bundleFor(srv.URL), "expected: 404", "expected: 200", 1))
	if _, err := execBundle(t, "bundle", "import", path, "-w", ws); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "exported.yaml")
	msg, err := execBundle(t, "bundle", "export", "-w", ws, "-o", out)
	if err != nil {
		t.Fatalf("%v\n%s", err, msg)
	}
	if !strings.Contains(msg, "1 secret value(s) replaced with CHANGE_ME") || !strings.Contains(msg, "Exported 1 environment(s), 1 request(s), 2 scenario(s)") {
		t.Errorf("messages:\n%s", msg)
	}
	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "real-key") || !strings.Contains(string(data), "CHANGE_ME") {
		t.Fatalf("secrets must be redacted:\n%s", data)
	}

	if run, err := execBundle(t, "bundle", "run", out); err != nil || !strings.Contains(run, "2/2 scenarios passed") {
		t.Errorf("the exported bundle runs as it is: %v\n%s", err, run)
	}

	stdout, err := execBundle(t, "bundle", "export", "-w", ws, "--keep-secrets")
	if err != nil || !strings.Contains(stdout, "real-key") || strings.Contains(stdout, "secret value(s) replaced") {
		t.Errorf("--keep-secrets to stdout: %v\n%s", err, stdout)
	}
}

func TestBundleExport_Problems(t *testing.T) {
	if _, err := execBundle(t, "bundle", "export"); err == nil || !strings.Contains(err.Error(), "--workspace") {
		t.Errorf("workspace is required: %v", err)
	}
	if _, err := execBundle(t, "bundle", "export", "-w", t.TempDir()); err == nil || !strings.Contains(err.Error(), "no environments, requests or scenarios") {
		t.Errorf("an empty workspace: %v", err)
	}
	ws := t.TempDir()
	os.MkdirAll(filepath.Join(ws, "requests"), 0o755)
	os.WriteFile(filepath.Join(ws, "requests", "ok.yaml"), []byte("method: GET\nurl: http://x\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "requests", "bad.yaml"), []byte("method: ["), 0o644)
	out, err := execBundle(t, "bundle", "export", "-w", ws)
	if err != nil || !strings.Contains(out, "warning: skipped requests/bad.yaml") {
		t.Errorf("unreadable files are reported: %v\n%s", err, out)
	}
}

// `clrsig run` used to fail on scenarios that use requestRef; imported bundles produce them.
func TestRun_ResolvesRequestRefsFromTheWorkspace(t *testing.T) {
	srv := pingServer(t)
	ws := t.TempDir()
	path := writeTempFile(t, "b.yaml", strings.Replace(bundleFor(srv.URL), "expected: 404", "expected: 200", 1))
	if _, err := execBundle(t, "bundle", "import", path, "-w", ws); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(ws, "environments", "local.yaml")

	out, err := execBundle(t, "run", filepath.Join(ws, "scenarios", "healthy.yaml"), "--env", envPath)
	if err != nil || !strings.Contains(out, "PASSED") {
		t.Errorf("a scenario beside its requests folder should run: %v\n%s", err, out)
	}

	// Moved away from its requests folder it cannot resolve; --workspace points it back.
	moved := filepath.Join(t.TempDir(), "healthy.yaml")
	data, _ := os.ReadFile(filepath.Join(ws, "scenarios", "healthy.yaml"))
	os.WriteFile(moved, []byte(strings.Replace(string(data), "../requests/ping.yaml", "ping.yaml", 1)), 0o644)
	out, err = execBundle(t, "run", moved, "--env", envPath)
	if err == nil || !strings.Contains(out, `requestRef "ping.yaml" could not be loaded`) {
		t.Errorf("an unresolvable ref is reported: %v\n%s", err, out)
	}
	out, err = execBundle(t, "run", moved, "--env", envPath, "--workspace", ws)
	if err != nil || !strings.Contains(out, "PASSED") {
		t.Errorf("--workspace adds its requests folder to the lookup: %v\n%s", err, out)
	}
}
