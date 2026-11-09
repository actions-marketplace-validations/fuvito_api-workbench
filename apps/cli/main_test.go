package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRoot builds a fresh root command with errors silenced so cobra does
// not write to os.Stderr and does not call os.Exit on RunE errors.
func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "clrsig", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(runCmd())
	root.AddCommand(requestCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(generateCmd())
	root.AddCommand(conformCmd())
	root.AddCommand(diffCmd())
	return root
}

// exec runs a subcommand and returns combined stdout+stderr.
func exec(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	root := newTestRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// writeTempFile creates a named file inside a fresh temp directory.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// pingServer starts a test HTTP server that responds 200 JSON to all requests.
func pingServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func scenarioYAML(url string) string {
	return fmt.Sprintf(`schemaVersion: "1"
title: "ping"
steps:
  - title: "get root"
    request:
      method: GET
      url: "%s"
    assertions:
      - type: status
        operator: eq
        expected: 200
`, url)
}

func requestFileYAML(url string) string {
	return fmt.Sprintf(`schemaVersion: "1"
title: "ping"
method: GET
url: "%s"
`, url)
}

const minimalSpec = `openapi: "3.0.3"
info:
  title: "test-api"
  version: "1.0.0"
paths:
  /ping:
    get:
      operationId: "getPing"
      responses:
        "200":
          description: "OK"
          content:
            application/json:
              schema:
                type: object
`

// ---- run ----

func TestRunCmd_TextOutput(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	out, err := exec(t, "run", sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "PASSED") {
		t.Errorf("want 'PASSED' in output, got:\n%s", out)
	}
}

func TestRunCmd_JSONOutput(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	out, err := exec(t, "run", "--format", "json", sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &m); jsonErr != nil {
		t.Fatalf("want valid JSON, got: %v\noutput: %s", jsonErr, out)
	}
	if passed, _ := m["passed"].(bool); !passed {
		t.Errorf("want passed=true, got: %v", m["passed"])
	}
}

func TestRunCmd_JUnitOutput(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	out, err := exec(t, "run", "--format", "junit", sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "<testsuite") {
		t.Errorf("want JUnit XML in output, got:\n%s", out)
	}
}

func TestRunCmd_OutputFile(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	outFile := filepath.Join(t.TempDir(), "results", "report.json")
	_, err := exec(t, "run", "--format", "json", "--output", outFile, sc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("output file not created: %v", readErr)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal(data, &m); jsonErr != nil {
		t.Errorf("output file is not valid JSON: %v\ncontent: %s", jsonErr, data)
	}
}

func TestRunCmd_WithEnv(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", `schemaVersion: "1"
title: "env test"
steps:
  - title: "get"
    request:
      method: GET
      url: "{{baseUrl}}"
    assertions:
      - type: status
        operator: eq
        expected: 200
`)
	env := writeTempFile(t, "env.yaml", fmt.Sprintf(`schemaVersion: "1"
variables:
  - name: baseUrl
    value: "%s"
`, srv.URL))
	out, err := exec(t, "run", "--env", env, sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "PASSED") {
		t.Errorf("want 'PASSED' in output, got:\n%s", out)
	}
}

func TestRunCmd_MissingFile(t *testing.T) {
	_, err := exec(t, "run", "/no/such/scenario.yaml")
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
}

func TestRunCmd_InvalidYAML(t *testing.T) {
	sc := writeTempFile(t, "bad.yaml", "not: [valid yaml")
	_, err := exec(t, "run", sc)
	if err == nil {
		t.Fatal("want error for invalid YAML, got nil")
	}
}

// ---- validate ----

func TestValidateCmd_Valid(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	out, err := exec(t, "validate", sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("want 'OK' in output, got:\n%s", out)
	}
}

func TestValidateCmd_MultipleValid(t *testing.T) {
	srv := pingServer(t)
	sc1 := writeTempFile(t, "s1.yaml", scenarioYAML(srv.URL))
	sc2 := writeTempFile(t, "s2.yaml", scenarioYAML(srv.URL))
	out, err := exec(t, "validate", sc1, sc2)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "2 file(s) checked") {
		t.Errorf("want summary line, got:\n%s", out)
	}
}

func TestValidateCmd_JSONFormat(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	out, err := exec(t, "validate", "--format", "json", sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var results []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &results); jsonErr != nil {
		t.Fatalf("want valid JSON array, got: %v\noutput: %s", jsonErr, out)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if valid, _ := results[0]["valid"].(bool); !valid {
		t.Errorf("want valid=true, got: %v", results[0])
	}
}

// ---- generate ----

func TestGenerateCmd_DryRun(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "generate", "--dry-run", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "GET") {
		t.Errorf("want operation info in output, got:\n%s", out)
	}
	if !strings.Contains(out, "dry run") {
		t.Errorf("want '(dry run)' marker in output, got:\n%s", out)
	}
}

func TestGenerateCmd_JSONFormat(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "generate", "--dry-run", "--format", "json", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &m); jsonErr != nil {
		t.Fatalf("want valid JSON, got: %v\noutput: %s", jsonErr, out)
	}
	gen, ok := m["generated"].([]any)
	if !ok || len(gen) == 0 {
		t.Errorf("want non-empty generated array, got: %v", m["generated"])
	}
}

func TestGenerateCmd_CreatesFiles(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	dir := t.TempDir()
	_, err := exec(t, "generate", "--output", dir, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("reading output dir: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("want at least one generated file, got none")
	}
}

func TestGenerateCmd_BaseURL(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "generate", "--dry-run", "--format", "json", "--base-url", "https://api.example.com/v1", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "api.example.com") {
		t.Errorf("want base URL in output, got:\n%s", out)
	}
}

func TestGenerateCmd_MissingSpec(t *testing.T) {
	_, err := exec(t, "generate", "--dry-run", "/no/such/spec.yaml")
	if err == nil {
		t.Fatal("want error for missing spec, got nil")
	}
}

// ---- request ----

func TestRequestCmd_TextOutput(t *testing.T) {
	srv := pingServer(t)
	req := writeTempFile(t, "request.yaml", requestFileYAML(srv.URL))
	out, err := exec(t, "request", req)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "200") {
		t.Errorf("want '200' in output, got:\n%s", out)
	}
}

func TestRequestCmd_JSONOutput(t *testing.T) {
	srv := pingServer(t)
	req := writeTempFile(t, "request.yaml", requestFileYAML(srv.URL))
	out, err := exec(t, "request", "--format", "json", req)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &m); jsonErr != nil {
		t.Errorf("want valid JSON, got: %v\noutput: %s", jsonErr, out)
	}
}

func TestRequestCmd_MissingFile(t *testing.T) {
	_, err := exec(t, "request", "/no/such/request.yaml")
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
}

// ---- conform ----

func TestConformCmd_TextOutput(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "conform", "--base-url", srv.URL, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if len(strings.TrimSpace(out)) == 0 {
		t.Error("want non-empty output")
	}
}

func TestConformCmd_JSONOutput(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "conform", "--base-url", srv.URL, "--format", "json", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &m); jsonErr != nil {
		t.Errorf("want valid JSON, got: %v\noutput: %s", jsonErr, out)
	}
}

func TestConformCmd_MissingSpec(t *testing.T) {
	_, err := exec(t, "conform", "--base-url", "http://localhost:9999", "/no/such/spec.yaml")
	if err == nil {
		t.Fatal("want error for missing spec, got nil")
	}
}

// ---- diff ----

func TestDiffCmd_NoChanges(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "diff", spec, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No changes") {
		t.Errorf("want 'No changes' in output, got:\n%s", out)
	}
}

func TestDiffCmd_JSONFormat(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "diff", "--format", "json", spec, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &m); jsonErr != nil {
		t.Fatalf("want valid JSON, got: %v\noutput: %s", jsonErr, out)
	}
	if breaking, _ := m["hasBreaking"].(bool); breaking {
		t.Errorf("want hasBreaking=false, got true")
	}
}

func TestDiffCmd_MissingFile(t *testing.T) {
	_, err := exec(t, "diff", "/no/base.yaml", "/no/rev.yaml")
	if err == nil {
		t.Fatal("want error for missing files, got nil")
	}
}

// ---- fetchSpec ----

func TestFetchSpec_LocalYAML(t *testing.T) {
	f := writeTempFile(t, "spec.yaml", minimalSpec)
	data, format, err := fetchSpec(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "yaml" {
		t.Errorf("want 'yaml', got %q", format)
	}
	if len(data) == 0 {
		t.Error("want non-empty data")
	}
}

func TestFetchSpec_LocalJSON(t *testing.T) {
	f := writeTempFile(t, "spec.json", `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{}}`)
	data, format, err := fetchSpec(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "json" {
		t.Errorf("want 'json', got %q", format)
	}
	if len(data) == 0 {
		t.Error("want non-empty data")
	}
}

func TestFetchSpec_LocalYML(t *testing.T) {
	f := writeTempFile(t, "spec.yml", minimalSpec)
	_, format, err := fetchSpec(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "yaml" {
		t.Errorf(".yml extension: want 'yaml', got %q", format)
	}
}

func TestFetchSpec_NonExistent(t *testing.T) {
	_, _, err := fetchSpec("/no/such/spec.yaml")
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
}

// ---- pure helpers ----

func TestGenSanitizeDirName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My API", "my-api"},
		{"test_service", "test-service"},
		{"Hello World 123", "hello-world-123"},
		{"ALL-CAPS", "all-caps"},
		{"v2.0 API!", "v20-api"},
		{"", ""},
	}
	for _, c := range cases {
		if got := genSanitizeDirName(c.in); got != c.want {
			t.Errorf("genSanitizeDirName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGenSanitizeFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"getUsers", "getUsers"},
		{"get-users", "get-users"},
		{"get users", "get-users"},
		{"create_resource", "create_resource"},
		{"", ""},
	}
	for _, c := range cases {
		if got := genSanitizeFilename(c.in); got != c.want {
			t.Errorf("genSanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
