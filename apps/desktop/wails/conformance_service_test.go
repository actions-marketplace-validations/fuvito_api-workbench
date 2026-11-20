package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

func newTestConformanceService(t *testing.T) *ConformanceService {
	t.Helper()
	isolateHome(t)
	return NewConformanceService(NewWorkspaceService(), NewSettingsService())
}

func conformanceSpecFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "conformance-openapi.yaml")
	if err := os.WriteFile(path, conformanceOpenAPIYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// conformanceEnvFile is the environment shipped with the server: it sets the path parameter and
// body value the generated requests use as {{placeholders}}.
func conformanceEnvFile(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("help", "conformance-environment.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConformanceService_ReadOnlyByDefaultAndKnownViolations(t *testing.T) {
	var writes atomic.Int32
	inner := newConformanceMux()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writes.Add(1)
		}
		inner.ServeHTTP(w, r)
	}))
	defer srv.Close()
	svc := newTestConformanceService(t)
	spec := conformanceSpecFile(t)

	info := svc.InspectConformSpec(spec)
	if info.Error != "" || info.Writes != 2 || info.Reads != 5 {
		t.Fatalf("inspect: %+v (want 5 reads, 2 writes)", info)
	}

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, EnvPath: conformanceEnvFile(t)})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if writes.Load() != 0 {
		t.Fatalf("read-only run sent %d write(s)", writes.Load())
	}
	if out.Total != 7 || out.Covered != 5 || out.NotCovered != 2 || out.Passing != 2 || out.Failing != 3 {
		t.Fatalf("totals: %+v", out)
	}
	// Each known violation must fail for its own reason: a bad field in the response body, not a
	// request that could not be built or a 404.
	cause := map[string]string{"getProduct": "/price", "listCategories": "name", "listOrders": "/id"}
	var failing []string
	for _, op := range out.Operations {
		if op.Status != "failed" {
			continue
		}
		failing = append(failing, op.OperationID)
		if op.StatusCode != 200 || len(op.Violations) != 1 ||
			!strings.Contains(op.Violations[0], cause[op.OperationID]) || strings.HasPrefix(op.Violations[0], "request:") {
			t.Errorf("%s failed for the wrong reason: status %d, %q", op.OperationID, op.StatusCode, op.Violations)
		}
	}
	want := append([]string(nil), conformanceKnownViolations...)
	sort.Strings(want)
	sort.Strings(failing)
	if strings.Join(failing, ",") != strings.Join(want, ",") {
		t.Fatalf("failing = %v, want the known violations %v", failing, want)
	}

	withWrites := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, EnvPath: conformanceEnvFile(t), AllowWrites: true})
	if writes.Load() == 0 || withWrites.NotCovered != 0 || withWrites.Passing != 4 || withWrites.Failing != 3 {
		t.Fatalf("with AllowWrites the write operations must run: writes=%d %+v", writes.Load(), withWrites)
	}
}

func TestConformanceService_RefusesUnconfirmedRemoteBaseURL(t *testing.T) {
	svc := newTestConformanceService(t)
	spec := conformanceSpecFile(t)

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: "https://api.example.com"})
	if !strings.Contains(out.Error, "confirm the base URL") {
		t.Fatalf("remote URL must need confirmation, got %+v", out)
	}
	if out := svc.RunConformance(ConformRunInput{SpecPath: spec}); !strings.Contains(out.Error, "enter a base URL") {
		t.Fatalf("spec without servers and no base URL: %+v", out)
	}
}

func TestIsLocalBaseURL(t *testing.T) {
	for url, want := range map[string]bool{
		"http://localhost:8080": true, "http://127.0.0.1:1": true, "http://[::1]:9": true,
		"https://api.example.com": false, "http://localhost.evil.com": false, "not a url": false,
	} {
		if got := IsLocalBaseURL(url); got != want {
			t.Errorf("IsLocalBaseURL(%q) = %v, want %v", url, got, want)
		}
	}
}

func localConformanceServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(newConformanceMux())
	t.Cleanup(srv.Close)
	return srv
}

func TestConformanceService_FiltersAndCoverageRecording(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	ws := t.TempDir()
	svc.ws.workspacePath = ws
	spec := filepath.Join(ws, "specs", "api.yaml")
	if err := os.MkdirAll(filepath.Dir(spec), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec, conformanceOpenAPIYAML, 0o600); err != nil {
		t.Fatal(err)
	}

	filtered := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, Tags: []string{"nothing"}, RecordCoverage: true})
	if filtered.Covered != 0 || !strings.Contains(filtered.CoverageNote, "would skew") {
		t.Fatalf("a filtered run must not be recorded: %+v", filtered)
	}
	if got := coverage.Load(ws, "specs/api.yaml"); got.TotalRuns != 0 {
		t.Fatalf("filtered run was recorded: %+v", got)
	}

	notAsked := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, AllowWrites: true, EnvPath: conformanceEnvFile(t)})
	if notAsked.CoverageNote != "" || coverage.Load(ws, "specs/api.yaml").TotalRuns != 0 {
		t.Fatal("coverage must be recorded only when asked")
	}

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, AllowWrites: true, RecordCoverage: true, EnvPath: conformanceEnvFile(t)})
	if !strings.HasPrefix(out.CoverageNote, "Added to coverage as specs/api.yaml") {
		t.Fatalf("note = %q", out.CoverageNote)
	}
	stored := coverage.Load(ws, "specs/api.yaml")
	if stored.TotalRuns != 1 || len(stored.Exercised) != 7 || stored.FailedRuns != 1 {
		t.Fatalf("stored coverage: %+v", stored)
	}
}

func TestConformanceService_NegativeProbesAndExport(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	spec := conformanceSpecFile(t)

	if r := svc.ExportConformReport("html"); r.Error == "" {
		t.Fatal("export before any run must fail")
	}

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, Negative: true, FailFast: true, TimeoutMs: 5000})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if len(out.Probes) == 0 {
		t.Fatal("negative probes were requested")
	}
	for _, p := range out.Probes {
		switch p.Status {
		case "rejected", "accepted", "error", "skipped":
		default:
			t.Fatalf("probe status %q", p.Status)
		}
	}

	for _, format := range []string{"html", "junit", "json", "markdown"} {
		if r := svc.ExportConformReport(format); r.Error != "" || r.Content == "" {
			t.Errorf("export %s: %+v", format, r.Error)
		}
	}
	if r := svc.ExportConformReport("pdf"); r.Error == "" {
		t.Fatal("unknown format must be an error")
	}
}

func TestScenarioSpecRef_ReusesTheNameAScenarioUses(t *testing.T) {
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "api.yaml")
	dir := filepath.Join(ws, "scenarios", "shop")
	for _, d := range []string{filepath.Dir(spec), dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	scn := "schemaVersion: \"1\"\ntitle: t\nspecFile: ../../specs/api.yaml\nsteps:\n  - label: a\n    request: {method: GET, url: \"http://x/\"}\n"
	if err := os.WriteFile(filepath.Join(dir, "s.yaml"), []byte(scn), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := scenarioSpecRef(ws, spec); got != "../../specs/api.yaml" {
		t.Fatalf("got %q, want the scenario's own spelling", got)
	}
	if got := scenarioSpecRef(ws, filepath.Join(ws, "specs", "other.yaml")); got != "" {
		t.Fatalf("a spec no scenario uses must give \"\", got %q", got)
	}
}

func TestConformanceService_WarnsAboutUnsetEnvironmentReferences(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	envPath := filepath.Join(t.TempDir(), "env.yaml")
	env := "schemaVersion: \"1\"\nname: e\nvariables:\n  - name: id\n    value: \"1\"\n  - name: token\n    value: \"${CS_SURELY_UNSET_VAR}\"\n"
	if err := os.WriteFile(envPath, []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	out := svc.RunConformance(ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: envPath})
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "CS_SURELY_UNSET_VAR") || !strings.Contains(out.Warnings[0], `"token"`) {
		t.Fatalf("warnings = %v", out.Warnings)
	}
	if clean := svc.RunConformance(ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: conformanceEnvFile(t)}); len(clean.Warnings) != 0 {
		t.Fatalf("no warning expected: %v", clean.Warnings)
	}
}
