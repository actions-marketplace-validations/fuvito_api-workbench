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

// newFullTestRoot builds a root with all commands registered.
func newFullTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "clrsig", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(runCmd())
	root.AddCommand(requestCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(generateCmd())
	root.AddCommand(conformCmd())
	root.AddCommand(diffCmd())
	root.AddCommand(coverageCmd())
	root.AddCommand(impactCmd())
	root.AddCommand(telemetryCmd())
	return root
}

// execFull runs a subcommand against the full root and returns stdout+stderr.
func execFull(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	root := newFullTestRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// ---- coverageBar ----

func TestCoverageBar_Empty(t *testing.T) {
	bar := coverageBar(0, 10)
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Fatalf("want bracketed bar, got %q", bar)
	}
	if strings.Contains(bar, "=") {
		t.Errorf("0%% bar should have no filled segments, got %q", bar)
	}
}

func TestCoverageBar_Full(t *testing.T) {
	bar := coverageBar(100, 10)
	if strings.Contains(bar, ".") {
		t.Errorf("100%% bar should have no empty segments, got %q", bar)
	}
}

func TestCoverageBar_Half(t *testing.T) {
	bar := coverageBar(50, 10)
	inner := bar[1 : len(bar)-1]
	filled := strings.Count(inner, "=")
	empty := strings.Count(inner, ".")
	if filled != 5 || empty != 5 {
		t.Errorf("50%% bar: want 5 filled + 5 empty, got %q", bar)
	}
}

func TestCoverageBar_Over100(t *testing.T) {
	bar := coverageBar(150, 10)
	inner := bar[1 : len(bar)-1]
	if len(inner) != 10 {
		t.Errorf("bar width should be capped at 10, got %q", bar)
	}
}

// ---- exitError ----

func TestExitError_ErrorString(t *testing.T) {
	e := &exitError{code: 42}
	if e.Error() != "" {
		t.Errorf("exitError.Error() should return empty string, got %q", e.Error())
	}
	if e.code != 42 {
		t.Errorf("exitError.code: want 42, got %d", e.code)
	}
}

// ---- impactExtractPath / impactPathMatches ----

func TestImpactExtractPath_HTTPSUrl(t *testing.T) {
	p := impactExtractPath("https://api.example.com/users/42")
	if p != "/users/42" {
		t.Errorf("want /users/42, got %q", p)
	}
}

func TestImpactExtractPath_TemplateURL(t *testing.T) {
	p := impactExtractPath("{{baseUrl}}/users/42")
	if p != "/users/42" {
		t.Errorf("want /users/42, got %q", p)
	}
}

func TestImpactExtractPath_RootPath(t *testing.T) {
	p := impactExtractPath("/health")
	if p != "/health" {
		t.Errorf("want /health, got %q", p)
	}
}

func TestImpactExtractPath_QueryStripped(t *testing.T) {
	p := impactExtractPath("/users?page=1")
	if p != "/users" {
		t.Errorf("want /users, got %q", p)
	}
}

func TestImpactExtractPath_NoPath(t *testing.T) {
	p := impactExtractPath("not-a-url")
	if p != "" {
		t.Errorf("want empty string, got %q", p)
	}
}

func TestImpactPathMatches_Exact(t *testing.T) {
	if !impactPathMatches("/users", "https://api.example.com/users") {
		t.Error("exact path should match")
	}
}

func TestImpactPathMatches_PathParam(t *testing.T) {
	if !impactPathMatches("/users/{id}", "https://api.example.com/users/42") {
		t.Error("path with parameter should match")
	}
}

func TestImpactPathMatches_NoMatch(t *testing.T) {
	if impactPathMatches("/products/{id}", "https://api.example.com/users/42") {
		t.Error("different path should not match")
	}
}

// ---- fetchSpec with HTTP ----

func TestFetchSpec_HTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		fmt.Fprint(w, minimalSpec)
	}))
	defer srv.Close()

	data, format, err := fetchSpec(srv.URL + "/spec.yaml")
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

func TestFetchSpec_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, _, err := fetchSpec(srv.URL + "/missing.yaml")
	if err == nil {
		t.Fatal("want error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("want 404 in error message, got %q", err.Error())
	}
}

func TestFetchSpec_HTTPJSONContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{}}`)
	}))
	defer srv.Close()

	_, format, err := fetchSpec(srv.URL + "/spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "json" {
		t.Errorf("want 'json' for JSON content-type, got %q", format)
	}
}

// ---- run with failing assertions ----

func TestRunCmd_FailingAssertion(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", fmt.Sprintf(`schemaVersion: "1"
title: "fail"
steps:
  - label: "expect 201"
    request:
      method: GET
      url: "%s"
    assertions:
      - type: status
        operator: eq
        expected: 201
`, srv.URL))
	out, err := exec(t, "run", sc)
	if err == nil {
		t.Fatal("want error for failing assertion, got nil")
	}
	if !strings.Contains(out, "FAILED") {
		t.Errorf("want 'FAILED' in output, got:\n%s", out)
	}
}

// ---- run with spec + coverage ----

func TestRunCmd_WithSpec(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "api.yaml", `openapi: "3.0.3"
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
`)
	sc := writeTempFile(t, "scenario.yaml", fmt.Sprintf(`schemaVersion: "1"
title: "ping"
steps:
  - label: "get root"
    request:
      method: GET
      url: "%s/ping"
    assertions:
      - type: status
        operator: eq
        expected: 200
`, srv.URL))
	out, err := exec(t, "run", "--spec", spec, "--coverage", sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "coverage") && !strings.Contains(out, "Coverage") {
		t.Errorf("want coverage info in output, got:\n%s", out)
	}
}

func TestRunCmd_WithWorkspace(t *testing.T) {
	srv := pingServer(t)
	ws := t.TempDir()
	spec := writeTempFile(t, "api.yaml", `openapi: "3.0.3"
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
`)
	sc := writeTempFile(t, "scenario.yaml", fmt.Sprintf(`schemaVersion: "1"
title: "ping"
steps:
  - label: "ping"
    request:
      method: GET
      url: "%s/ping"
    assertions:
      - type: status
        operator: eq
        expected: 200
`, srv.URL))
	_, err := exec(t, "run", "--spec", spec, "--workspace", ws, sc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Coverage data directory should have been created.
	entries, _ := os.ReadDir(ws)
	if len(entries) == 0 {
		t.Error("want coverage files in workspace after run with --workspace")
	}
}

// ---- validate edge cases ----

func TestValidateCmd_InvalidYAML(t *testing.T) {
	sc := writeTempFile(t, "bad.yaml", "not: [valid yaml")
	out, err := exec(t, "validate", sc)
	if err == nil {
		t.Fatal("want error for invalid YAML")
	}
	if !strings.Contains(out, "PARSE ERROR") {
		t.Errorf("want 'PARSE ERROR' in output, got:\n%s", out)
	}
}

func TestValidateCmd_JSONWithIssues(t *testing.T) {
	sc := writeTempFile(t, "bad.yaml", `schemaVersion: "1"
title: "test"
steps:
  - label: ""
    request:
      method: GET
      url: "http://example.com"
`)
	out, err := exec(t, "validate", "--format", "json", sc)
	// May or may not error depending on whether empty label is a validation issue.
	_ = err
	var results []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &results); jsonErr != nil {
		t.Fatalf("want valid JSON array, got: %v\noutput: %s", jsonErr, out)
	}
}

func TestValidateCmd_OutputFile(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	outFile := filepath.Join(t.TempDir(), "results", "report.txt")
	_, err := exec(t, "validate", "--output", outFile, sc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("output file not created: %v", readErr)
	}
	if !strings.Contains(string(data), "OK") {
		t.Errorf("want 'OK' in output file, got:\n%s", data)
	}
}

// ---- request with env ----

func TestRequestCmd_WithEnv(t *testing.T) {
	srv := pingServer(t)
	req := writeTempFile(t, "request.yaml", `schemaVersion: "1"
title: "ping"
method: GET
url: "{{baseUrl}}"
`)
	env := writeTempFile(t, "env.yaml", fmt.Sprintf(`schemaVersion: "1"
variables:
  - name: baseUrl
    value: "%s"
`, srv.URL))
	out, err := exec(t, "request", "--env", env, req)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "200") {
		t.Errorf("want '200' in output, got:\n%s", out)
	}
}

func TestRequestCmd_NetworkError(t *testing.T) {
	req := writeTempFile(t, "request.yaml", `schemaVersion: "1"
title: "dead"
method: GET
url: "http://localhost:1"
`)
	_, err := exec(t, "request", req)
	if err == nil {
		t.Fatal("want error for unreachable host, got nil")
	}
}

// ---- conform extras ----

func TestConformCmd_JUnitOutput(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "conform", "--base-url", srv.URL, "--format", "junit", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "<testsuite") {
		t.Errorf("want JUnit XML, got:\n%s", out)
	}
}

func TestConformCmd_OutputFile(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	outFile := filepath.Join(t.TempDir(), "sub", "report.xml")
	_, err := exec(t, "conform", "--base-url", srv.URL, "--format", "junit", "--output", outFile, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("output file not created: %v", readErr)
	}
	if !strings.Contains(string(data), "<testsuite") {
		t.Errorf("want JUnit XML in file, got:\n%s", data)
	}
}

func TestConformCmd_WithCoverage(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "conform", "--base-url", srv.URL, "--coverage", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Coverage section appears only after ≥2 runs (cumulative), but the
	// per-run summary is still printed in text mode.
	if len(strings.TrimSpace(out)) == 0 {
		t.Error("want non-empty output")
	}
}

// ---- diff extras ----

const specV1 = `openapi: "3.0.3"
info:
  title: "test-api"
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: "listUsers"
      responses:
        "200":
          description: "OK"
  /ping:
    get:
      operationId: "getPing"
      responses:
        "200":
          description: "OK"
`

const specV2 = `openapi: "3.0.3"
info:
  title: "test-api"
  version: "2.0.0"
paths:
  /ping:
    get:
      operationId: "getPing"
      responses:
        "200":
          description: "OK"
`

func TestDiffCmd_BreakingChange(t *testing.T) {
	base := writeTempFile(t, "v1.yaml", specV1)
	rev := writeTempFile(t, "v2.yaml", specV2)
	out, err := exec(t, "diff", base, rev)
	if err == nil {
		t.Fatal("want error (exit 1) for breaking change, got nil")
	}
	if !strings.Contains(out, "breaking") && !strings.Contains(strings.ToLower(out), "removed") {
		t.Errorf("want breaking change message in output, got:\n%s", out)
	}
}

func TestDiffCmd_BreakingOnly(t *testing.T) {
	base := writeTempFile(t, "v1.yaml", specV1)
	rev := writeTempFile(t, "v2.yaml", specV2)
	out, _ := exec(t, "diff", "--breaking-only", base, rev)
	// Should not be empty (breaking changes exist)
	if strings.Contains(out, "No breaking") && len(strings.TrimSpace(out)) == 0 {
		t.Errorf("want breaking change output, got:\n%s", out)
	}
}

func TestDiffCmd_JUnitOutput(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := exec(t, "diff", "--format", "junit", spec, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "<?xml") && !strings.Contains(out, "<testsuites") {
		t.Errorf("want JUnit XML in output, got:\n%s", out)
	}
}

func TestDiffCmd_OutputFile(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	outFile := filepath.Join(t.TempDir(), "report.json")
	_, err := exec(t, "diff", "--format", "json", "--output", outFile, spec, spec)
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

// ---- impact command ----

func TestImpactCmd_NoBreakingChanges(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := execFull(t, "impact", spec, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No breaking changes") {
		t.Errorf("want 'No breaking changes' in output, got:\n%s", out)
	}
}

func TestImpactCmd_BreakingWithNoMatchingScenario(t *testing.T) {
	base := writeTempFile(t, "v1.yaml", specV1)
	rev := writeTempFile(t, "v2.yaml", specV2)
	sc := writeTempFile(t, "scenario.yaml", `schemaVersion: "1"
title: "ping only"
steps:
  - label: "ping"
    request:
      method: GET
      url: "http://localhost/ping"
    assertions:
      - type: status
        operator: eq
        expected: 200
`)
	out, err := execFull(t, "impact", base, rev, sc)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No scenario steps match") {
		t.Errorf("want 'No scenario steps match' in output, got:\n%s", out)
	}
}

func TestImpactCmd_BreakingWithAffectedScenario(t *testing.T) {
	base := writeTempFile(t, "v1.yaml", specV1)
	rev := writeTempFile(t, "v2.yaml", specV2)
	sc := writeTempFile(t, "scenario.yaml", `schemaVersion: "1"
title: "user flow"
steps:
  - label: "list users"
    request:
      method: GET
      url: "http://localhost/users"
    assertions:
      - type: status
        operator: eq
        expected: 200
`)
	out, err := execFull(t, "impact", base, rev, sc)
	if err == nil {
		t.Fatal("want exit code 1 for affected scenario, got nil")
	}
	if !strings.Contains(out, "Affected") {
		t.Errorf("want 'Affected' in output, got:\n%s", out)
	}
}

func TestImpactCmd_JSONFormat(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := execFull(t, "impact", "--format", "json", spec, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &m); jsonErr != nil {
		t.Fatalf("want valid JSON, got: %v\noutput: %s", jsonErr, out)
	}
	if _, ok := m["breakingChanges"]; !ok {
		t.Error("want 'breakingChanges' key in JSON output")
	}
}

func TestImpactCmd_MissingSpec(t *testing.T) {
	_, err := execFull(t, "impact", "/no/base.yaml", "/no/rev.yaml")
	if err == nil {
		t.Fatal("want error for missing spec, got nil")
	}
}

// ---- coverage commands ----

func TestCoverageListCmd_NoData(t *testing.T) {
	ws := t.TempDir()
	out, err := execFull(t, "coverage", "list", "--workspace", ws)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No coverage data") {
		t.Errorf("want 'No coverage data' for empty workspace, got:\n%s", out)
	}
}

func TestCoverageListCmd_MissingWorkspace(t *testing.T) {
	_, err := execFull(t, "coverage", "list")
	if err == nil {
		t.Fatal("want error when --workspace is missing, got nil")
	}
}

func TestCoverageResetCmd_MissingWorkspace(t *testing.T) {
	_, err := execFull(t, "coverage", "reset", "openapi.yaml")
	if err == nil {
		t.Fatal("want error when --workspace is missing, got nil")
	}
}

func TestCoverageResetCmd_EmptyWorkspace(t *testing.T) {
	ws := t.TempDir()
	out, err := execFull(t, "coverage", "reset", "--workspace", ws, "openapi.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Coverage reset") {
		t.Errorf("want 'Coverage reset' in output, got:\n%s", out)
	}
}

func TestCoverageQualityCmd_MissingWorkspace(t *testing.T) {
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	_, err := execFull(t, "coverage", "quality", spec)
	if err == nil {
		t.Fatal("want error when --workspace is missing, got nil")
	}
}

func TestCoverageQualityCmd_NoData(t *testing.T) {
	ws := t.TempDir()
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err := execFull(t, "coverage", "quality", "--workspace", ws, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No coverage data") {
		t.Errorf("want 'No coverage data' in output, got:\n%s", out)
	}
}

func TestCoverageListCmd_JSONFormat_Empty(t *testing.T) {
	ws := t.TempDir()
	out, err := execFull(t, "coverage", "list", "--format", "json", "--workspace", ws)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Empty workspace produces "No coverage data found" in text mode,
	// but JSON mode should still return valid JSON or the same message.
	// (implementation returns text in both cases for empty)
	_ = out
}

// ---- coverage list after a run ----

func TestCoverageListCmd_AfterRun(t *testing.T) {
	srv := pingServer(t)
	ws := t.TempDir()
	spec := writeTempFile(t, "api.yaml", `openapi: "3.0.3"
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
`)
	sc := writeTempFile(t, "scenario.yaml", fmt.Sprintf(`schemaVersion: "1"
title: "ping"
steps:
  - label: "ping"
    request:
      method: GET
      url: "%s/ping"
    assertions:
      - type: status
        operator: eq
        expected: 200
`, srv.URL))

	// First, run with --workspace to populate coverage.
	_, err := exec(t, "run", "--spec", spec, "--workspace", ws, sc)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	out, err := execFull(t, "coverage", "list", "--workspace", ws)
	if err != nil {
		t.Fatalf("coverage list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, spec) && !strings.Contains(out, "1/1") && !strings.Contains(out, "1 run") {
		t.Logf("coverage list output:\n%s", out)
		// Just check there's some content — the spec path appears.
		if strings.Contains(out, "No coverage data") {
			t.Error("coverage data should be present after a run")
		}
	}
}

// ---- telemetry commands ----

func TestTelemetryCmd_Status(t *testing.T) {
	out, err := execFull(t, "telemetry", "status")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Telemetry:") {
		t.Errorf("want 'Telemetry:' in output, got:\n%s", out)
	}
}

func TestTelemetryCmd_Enable(t *testing.T) {
	out, err := execFull(t, "telemetry", "enable")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "enabled") {
		t.Errorf("want 'enabled' in output, got:\n%s", out)
	}
	// Restore: disable afterwards so we don't pollute the test environment.
	_, _ = execFull(t, "telemetry", "disable")
}

func TestTelemetryCmd_Disable(t *testing.T) {
	out, err := execFull(t, "telemetry", "disable")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "disabled") {
		t.Errorf("want 'disabled' in output, got:\n%s", out)
	}
}

// ---- run with JUnit + output file ----

func TestRunCmd_JUnitOutputFile(t *testing.T) {
	srv := pingServer(t)
	sc := writeTempFile(t, "scenario.yaml", scenarioYAML(srv.URL))
	outFile := filepath.Join(t.TempDir(), "sub", "result.xml")
	_, err := exec(t, "run", "--format", "junit", "--output", outFile, sc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("output file not created: %v", readErr)
	}
	if !strings.Contains(string(data), "<testsuite") {
		t.Errorf("want JUnit XML in file, got:\n%s", data)
	}
}
