package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// testWorkspaces gives each test one workspace folder, shared by the helpers below: the service reads
// spec and environment files only from inside the open workspace.
var testWorkspaces sync.Map

func testWorkspaceDir(t *testing.T) string {
	t.Helper()
	if dir, ok := testWorkspaces.Load(t.Name()); ok {
		return dir.(string)
	}
	dir := t.TempDir()
	testWorkspaces.Store(t.Name(), dir)
	return dir
}

func newTestConformanceService(t *testing.T) *ConformanceService {
	t.Helper()
	isolateHome(t)
	ws := NewWorkspaceService()
	ws.workspacePath = testWorkspaceDir(t)
	return NewConformanceService(ws, NewSettingsService())
}

func conformanceSpecFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(testWorkspaceDir(t), "conformance-openapi.yaml")
	if err := os.WriteFile(path, conformanceOpenAPIYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// conformanceEnvFile is the environment shipped with the server: it sets the path parameter and
// body value the generated requests use as {{placeholders}}.
func conformanceEnvFile(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("help", "conformance-environment.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(testWorkspaceDir(t), "conformance-environment.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
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
	if info.Error != "" || info.Writes != 2 || info.Reads != 8 {
		t.Fatalf("inspect: %+v (want 8 reads, 2 writes)", info)
	}

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, EnvPath: conformanceEnvFile(t)})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if writes.Load() != 0 {
		t.Fatalf("read-only run sent %d write(s)", writes.Load())
	}
	if out.Total != 10 || out.Covered != 8 || out.NotCovered != 2 || out.Passing != 2 || out.Failing != 6 {
		t.Fatalf("totals: %+v", out)
	}
	// Each known violation must fail for its own reason, and say what and where it is: a bad field
	// in the body, a wrong Content-Type, a missing header, an undeclared status; never a request
	// that could not be built.
	known := map[string]struct {
		kind   string
		status int
		inMsg  []string // the raw message names the operation and the thing that is wrong
		where  string   // Finding.Where
	}{
		"getProduct":     {"wrong-type", 200, []string{"/price"}, "price"},
		"listCategories": {"missing-field", 200, []string{"name"}, "[0].name"},
		"listOrders":     {"wrong-type", 200, []string{"/id"}, "[0].id"},
		"listWidgets":    {"wrong-content-type", 200, []string{"GET /widgets", `"text/plain"`, "application/json"}, "Content-Type"},
		"getInventory":   {"missing-header", 200, []string{"GET /inventory", `"X-Total-Count"`}, "X-Total-Count"},
		"getStatus":      {"undeclared-status", 503, []string{"GET /status", "answered 503", "200"}, "503"},
	}
	var failing []string
	for _, op := range out.Operations {
		if op.Status != "failed" {
			continue
		}
		failing = append(failing, op.OperationID)
		k := known[op.OperationID]
		if len(op.Findings) != 1 || op.Findings[0].Kind != k.kind || op.Findings[0].Why == "" || op.Findings[0].Fix == "" ||
			op.Findings[0].Part == "" || op.Findings[0].Spec == "" || op.Findings[0].Where != k.where {
			t.Errorf("%s findings: %+v, want one %s at %q with part, spec location, why and fix", op.OperationID, op.Findings, k.kind, k.where)
		}
		if op.StatusCode != k.status || len(op.Violations) != 1 || strings.HasPrefix(op.Violations[0], "request:") {
			t.Errorf("%s failed for the wrong reason: status %d, %q", op.OperationID, op.StatusCode, op.Violations)
			continue
		}
		for _, want := range k.inMsg {
			if !strings.Contains(op.Violations[0], want) {
				t.Errorf("%s: the message should say %q:\n%s", op.OperationID, want, op.Violations[0])
			}
		}
	}
	want := append([]string(nil), conformanceKnownViolations...)
	sort.Strings(want)
	sort.Strings(failing)
	if strings.Join(failing, ",") != strings.Join(want, ",") {
		t.Fatalf("failing = %v, want the known violations %v", failing, want)
	}

	withWrites := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, EnvPath: conformanceEnvFile(t), AllowWrites: true})
	if writes.Load() == 0 || withWrites.NotCovered != 0 || withWrites.Passing != 4 || withWrites.Failing != 6 {
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
	ws := testWorkspaceDir(t)
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
	if stored.TotalRuns != 1 || len(stored.Exercised) != 10 || stored.FailedRuns != 1 {
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

func TestConformanceService_ContextNamesTheActiveEnvironment(t *testing.T) {
	svc := newTestConformanceService(t)
	if got := svc.GetConformContext().ActiveEnvironment; got != "" {
		t.Fatalf("no environment selected, got %q", got)
	}
	svc.ws.activeEnvPath = "/ws/environments/local.yaml"
	if got := svc.GetConformContext().ActiveEnvironment; got != "/ws/environments/local.yaml" {
		t.Fatalf("got %q", got)
	}
}

func TestConformanceService_WarnsAboutUnsetEnvironmentReferences(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	envPath := filepath.Join(testWorkspaceDir(t), "env.yaml")
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

func TestConformanceService_VariableOverrides(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	envPath := filepath.Join(testWorkspaceDir(t), "env.yaml")
	if err := os.WriteFile(envPath, []byte("schemaVersion: \"1\"\nname: e\nvariables:\n  - name: name\n    value: Lamp\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec := conformanceSpecFile(t)
	op := func(out ConformRunOutputIO, id string) ConformOperationIO {
		for _, o := range out.Operations {
			if o.OperationID == id {
				return o
			}
		}
		t.Fatalf("no operation %s", id)
		return ConformOperationIO{}
	}

	// Without a value for the path parameter the request is not sent at all.
	var paths []string
	counting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		newConformanceMux().ServeHTTP(w, r)
	}))
	defer counting.Close()
	outWithout := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: counting.URL, EnvPath: envPath})
	without := op(outWithout, "getProduct")
	if without.Status != "blocked" || without.StatusCode != 0 || len(without.Findings) != 1 || without.Findings[0].Kind != "missing-value" {
		t.Fatalf("getProduct without id: %+v", without)
	}
	// Read-only skips deleteProduct before it needs {{id}}, so only getProduct is blocked.
	if len(outWithout.MissingValues) != 1 || outWithout.MissingValues[0] != "id" || outWithout.BlockedOperations != 1 {
		t.Fatalf("missing values %v for %d blocked operations", outWithout.MissingValues, outWithout.BlockedOperations)
	}
	for _, p := range paths {
		if strings.HasPrefix(p, "/products/") {
			t.Fatalf("a request for %s reached the server although {{id}} had no value: %v", p, paths)
		}
	}
	if outWithout.Covered+outWithout.NotCovered != outWithout.Total {
		t.Fatalf("Covered + Not covered must equal the total: %+v", outWithout)
	}
	// An override supplies it for this run, so the real violation shows.
	with := op(svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, EnvPath: envPath, Vars: []string{"id=1"}}), "getProduct")
	if with.StatusCode != 200 || len(with.Findings) != 1 || with.Findings[0].Kind != "wrong-type" {
		t.Fatalf("getProduct with id=1: %+v", with)
	}
	// The environment file itself was not changed.
	if data, _ := os.ReadFile(envPath); strings.Contains(string(data), "id") && strings.Contains(string(data), "value: \"1\"") {
		t.Fatal("an override must not be written to the environment file")
	}

	bad := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, EnvPath: envPath, Vars: []string{"oops"}})
	if !strings.Contains(bad.Error, "variable overrides") || !strings.Contains(bad.Error, "name=value") {
		t.Fatalf("a malformed override must be reported: %q", bad.Error)
	}
}

func TestConformanceService_EmitsProgressEvents(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	var events []ConformProgressEvent
	svc.emit = func(name string, data ...interface{}) {
		if name != "conformance:progress" || len(data) != 1 {
			t.Errorf("unexpected event %q %v", name, data)
			return
		}
		events = append(events, data[0].(ConformProgressEvent))
	}
	out := svc.RunConformance(ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: conformanceEnvFile(t), AllowWrites: true})
	if out.Error != "" || len(events) != out.Total || out.Total != 10 {
		t.Fatalf("%d events for %d operations (%v)", len(events), out.Total, out.Error)
	}
	for i, e := range events {
		if e.Done != i+1 || e.Total != 10 || e.Operation == "" {
			t.Errorf("event %d = %+v", i, e)
		}
	}
}

func TestConformanceService_ReadsOnlyFilesInsideTheWorkspace(t *testing.T) {
	svc := newTestConformanceService(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.yaml")
	if err := os.WriteFile(outside, conformanceOpenAPIYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	inside := conformanceSpecFile(t)

	if info := svc.InspectConformSpec(outside); !strings.Contains(info.Error, "outside the open workspace") {
		t.Fatalf("inspect outside: %+v", info)
	}
	out := svc.RunConformance(ConformRunInput{SpecPath: outside, BaseURL: "http://localhost:1"})
	if !strings.Contains(out.Error, "outside the open workspace") || !strings.Contains(out.Error, "copy the file into the workspace") {
		t.Fatalf("run with a spec outside: %q", out.Error)
	}
	out = svc.RunConformance(ConformRunInput{SpecPath: inside, EnvPath: outside, BaseURL: "http://localhost:1"})
	if !strings.Contains(out.Error, "the environment") || !strings.Contains(out.Error, "outside the open workspace") {
		t.Fatalf("run with an environment outside: %q", out.Error)
	}
	if info := svc.InspectConformSpec(inside); info.Error != "" {
		t.Fatalf("inside the workspace must work: %+v", info)
	}

	closed := NewConformanceService(NewWorkspaceService(), NewSettingsService())
	if info := closed.InspectConformSpec(inside); !strings.Contains(info.Error, "open a workspace first") {
		t.Fatalf("no workspace: %+v", info)
	}
}

func TestPathInside_FollowsSymbolicLinks(t *testing.T) {
	ws := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "spec.yaml"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "spec.yaml"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !pathInside(ws, filepath.Join(ws, "spec.yaml")) || !pathInside(ws, ws) {
		t.Fatal("a file inside the workspace must count as inside")
	}
	if pathInside(ws, filepath.Join(elsewhere, "spec.yaml")) || pathInside(ws, filepath.Join(ws, "..", "x.yaml")) {
		t.Fatal("a file outside must count as outside")
	}
	if err := os.Symlink(elsewhere, filepath.Join(ws, "link")); err != nil {
		t.Skip("symbolic links are not available here:", err)
	}
	if pathInside(ws, filepath.Join(ws, "link", "spec.yaml")) {
		t.Fatal("a link that points out of the workspace must count as outside")
	}
}

// Lowering one check to a warning moves the operations that only break it from Failing to Passing
// (with a warning), and "fail on warnings" puts them back, as clrsig conform --fail-on warning does.
// The Conformance Server has two operations whose only problem is a wrong type (CS-RESP-006).
func TestConformanceService_WarningsFollowFailOnWarning(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	in := ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: conformanceEnvFile(t)}

	base := svc.RunConformance(in)
	if base.Error != "" || base.Failing != 6 || base.WithWarnings != 0 {
		t.Fatalf("default severities: %+v", base)
	}

	t.Cleanup(checks.WithSeverity("CS-RESP-006", checks.Warning))
	out := svc.RunConformance(in)
	if out.Failing != 4 || out.Passing != 4 || out.WithWarnings != 2 || !out.Failed {
		t.Fatalf("warnings: failing=%d passing=%d withWarnings=%d failed=%v", out.Failing, out.Passing, out.WithWarnings, out.Failed)
	}
	seen := 0
	for _, op := range out.Operations {
		if op.Warnings > 0 {
			seen++
			if op.Status != "passed" || len(op.Findings) != 1 || op.Findings[0].Severity != "warning" || op.Findings[0].CheckID != "CS-RESP-006" {
				t.Fatalf("operation with a warning: %+v", op)
			}
		}
	}
	if seen != 2 {
		t.Fatalf("%d operations with warnings, want 2", seen)
	}

	in.FailOnWarning = true
	out = svc.RunConformance(in)
	if out.Failing != 6 || out.WithWarnings != 0 || !out.Failed {
		t.Fatalf("fail on warning: failing=%d withWarnings=%d failed=%v", out.Failing, out.WithWarnings, out.Failed)
	}
}

// A baseline written by 0.8.9 against the Conformance Server (schemaVersion 1, identified by the
// validator's wording) still accepts exactly the six known violations with this version, and a
// baseline recorded now is in the current format and accepts them too.
func TestBaselineWrittenBy089StillWorks(t *testing.T) {
	srv := localConformanceServer(t)
	spec, err := openapi.ParseBytes(conformanceOpenAPIYAML, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	envData, err := os.ReadFile("help/conformance-environment.yaml")
	if err != nil {
		t.Fatal(err)
	}
	env, err := serialization.ParseEnvironment(envData, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "baseline-v1-0.8.9.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	old, err := conformance.ParseBaseline(golden, "baseline-v1-0.8.9.yaml")
	if err != nil || !old.Legacy || len(old.Violations) != 6 {
		t.Fatalf("golden file: %v %+v", err, old)
	}

	run := func(b *conformance.Baseline) application.ConformOutput {
		return application.RunConformance(context.Background(), application.ConformInput{
			Exec: executor.New(executor.Options{}), Spec: spec, BaseURL: srv.URL, Env: env, ReadOnly: true, Baseline: b,
		})
	}
	out := run(old)
	if out.Failed || out.Diff == nil || len(out.Diff.Known) != 6 || len(out.Diff.New) != 0 || len(out.Diff.Fixed) != 0 || !out.Diff.Legacy {
		t.Fatalf("a 0.8.9 baseline must accept the six known violations: failed=%v %+v", out.Failed, out.Diff)
	}

	current := conformance.NewBaseline(out.Violations)
	if current.Legacy || current.SchemaVersion != "2" || len(current.Violations) != 6 {
		t.Fatalf("recorded baseline: %+v", current)
	}
	checksSeen := map[string]bool{}
	for _, v := range current.Violations {
		checksSeen[v.Check] = true
		if v.Check == "" || v.Check == "other" {
			t.Errorf("every violation of the Conformance Server has a check: %+v", v)
		}
	}
	if again := run(current); again.Failed || len(again.Diff.Known) != 6 || again.Diff.Legacy {
		t.Fatalf("the current format accepts them too: %+v", again.Diff)
	}
}

func TestConformanceService_PreselectsFromTheProjectFile(t *testing.T) {
	svc := newTestConformanceService(t)
	ws := testWorkspaceDir(t)
	if d := svc.GetConformContext().ProjectDefaults; d.Found {
		t.Fatalf("no project file: %+v", d)
	}

	conformanceSpecFile(t) // <ws>/conformance-openapi.yaml
	conformanceEnvFile(t)  // <ws>/conformance-environment.yaml
	writeProject := func(text string) {
		if err := os.WriteFile(filepath.Join(ws, "clrsig.yaml"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeProject("schemaVersion: \"1\"\nspec: conformance-openapi.yaml\nenvironment: conformance-environment.yaml\nbaseUrl: http://localhost:9999\n" +
		"conform:\n  readOnly: false\n  negative: true\n  failOn: warning\n  requestTimeoutMs: 3000\n  maxResponseTimeMs: 800\n  maxResponseTimeFor: [\"/slow=5000\"]\n  tags: [products]\n  paths: [/products]\n")
	d := svc.GetConformContext().ProjectDefaults
	if !d.Found || d.File != "clrsig.yaml" || d.SpecPath != filepath.Join(ws, "conformance-openapi.yaml") || d.EnvironmentPath != filepath.Join(ws, "conformance-environment.yaml") || d.BaseURL != "http://localhost:9999" {
		t.Fatalf("paths and base URL: %+v", d)
	}
	if !d.Negative || !d.FailOnWarning || d.RequestTimeoutMs != 3000 || d.MaxResponseTimeMs != 800 || len(d.MaxResponseTimeFor) != 1 || d.Tags[0] != "products" || d.Paths[0] != "/products" {
		t.Fatalf("options: %+v", d)
	}
	if !strings.Contains(d.Note, "readOnly: false is not applied") {
		t.Errorf("the page stays read-only whatever the file says, and says so: %q", d.Note)
	}

	// a path outside the workspace is not preselected
	writeProject("schemaVersion: \"1\"\nspec: ../elsewhere.yaml\n")
	d = svc.GetConformContext().ProjectDefaults
	if d.SpecPath != "" || !strings.Contains(d.Note, "outside the workspace") {
		t.Errorf("a path outside the workspace is not used: %+v", d)
	}
	// a file that cannot be used is reported, not guessed at
	writeProject("schemaVersion: \"1\"\nbaseURL: x\n")
	d = svc.GetConformContext().ProjectDefaults
	if !d.Found || !strings.Contains(d.Note, "baseURL") || len(d.Applied) != 0 {
		t.Errorf("an unusable file: %+v", d)
	}
}

const setupTestSpec = `openapi: "3.0.3"
info: {title: secured, version: "1"}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
paths:
  /secure:
    get:
      operationId: secure
      security: [{bearerAuth: []}]
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [ok], properties: {ok: {type: boolean}}}
        "401": {description: no}
`

const setupTestLogin = `schemaVersion: "1"
title: Login
steps:
  - label: Log in
    request:
      method: POST
      url: "{{baseUrl}}/login"
    extractions:
      - name: token
        source: jsonpath
        path: $.token
    assertions:
      - type: status
        operator: eq
        expected: 200
`

func setupServer(t *testing.T, loginStatus int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login":
			w.WriteHeader(loginStatus)
			_, _ = w.Write([]byte(`{"token":"tkn-s3cr3t-12345"}`))
		case "/secure":
			if r.Header.Get("Authorization") != "Bearer tkn-s3cr3t-12345" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestConformanceService_RunsASetupScenarioFirst(t *testing.T) {
	svc := newTestConformanceService(t)
	ws := testWorkspaceDir(t)
	spec := filepath.Join(ws, "api.yaml")
	login := filepath.Join(ws, "scenarios", "login.yaml")
	for p, text := range map[string]string{spec: setupTestSpec, login: setupTestLogin} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if ctx := svc.GetConformContext(); len(ctx.Scenarios) != 1 || ctx.Scenarios[0].Label != "scenarios/login.yaml" && ctx.Scenarios[0].Label != `scenarios\login.yaml` {
		t.Fatalf("the page offers the workspace scenarios: %+v", ctx.Scenarios)
	}

	srv := setupServer(t, 200)
	without := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL})
	if without.Error != "" || without.BlockedOperations != 1 || !without.Failed {
		t.Fatalf("without a setup the secured operation is not sent: %+v", without)
	}

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, SetupScenario: login})
	if out.Error != "" || out.Failed || out.Passing != 1 || !strings.Contains(out.SetupNote, "values available to the check: token") {
		t.Fatalf("with the setup: %+v", out)
	}
	if strings.Contains(out.SetupNote, "tkn-s3cr3t") {
		t.Error("the note names values, never shows them")
	}

	bad := setupServer(t, 401)
	fail := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: bad.URL, SetupScenario: login})
	if !strings.Contains(fail.Error, "did not pass, so the check was not started") || strings.Contains(fail.Error, "tkn-s3cr3t") {
		t.Fatalf("a failed setup stops the run and does not print the token: %q", fail.Error)
	}

	if outside := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, SetupScenario: filepath.Join(t.TempDir(), "x.yaml")}); !strings.Contains(outside.Error, "outside the open workspace") {
		t.Errorf("a setup scenario outside the workspace is refused: %q", outside.Error)
	}
}

func TestConformanceService_FindingsShownInThePageHideSecrets(t *testing.T) {
	const secret = "sk-live-ABCDEFGH12345"
	msg := "Error at \"/token\": value must be a number\nSchema:\n  {\n    \"type\": \"number\"\n  }\n\nValue:\n  \"" + secret + "\"\n"
	out := application.ConformOutput{Result: &conformance.ConformResult{
		Secrets: []string{secret},
		Operations: []conformance.OperationResult{
			{Method: "GET", Path: "/t", StatusCode: 200, Violations: []openapi.ValidationError{{Message: msg}}},
		},
	}}
	res := conformOutputIO(out, "http://x", false)
	if len(res.Operations) != 1 || len(res.Operations[0].Findings) != 1 {
		t.Fatalf("%+v", res)
	}
	f := res.Operations[0].Findings[0]
	for _, text := range []string{f.Summary, f.Expected, f.Got, f.Why, f.Fix, f.Detail} {
		if strings.Contains(text, secret) {
			t.Fatalf("the finding leaks a secret: %+v", f)
		}
	}
}

func TestConformanceService_RecordAndCompareABaseline(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	ws := testWorkspaceDir(t)
	in := ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: conformanceEnvFile(t)}

	if r := svc.RecordConformBaseline(filepath.Join(ws, "conform-baseline.yaml")); !strings.Contains(r.Error, "run the check first") {
		t.Fatalf("nothing to record before a run: %+v", r)
	}
	first := svc.RunConformance(in)
	if first.Error != "" || !first.Failed || first.Baseline != nil {
		t.Fatalf("without a baseline the known violations fail the run: %+v", first)
	}

	path := filepath.Join(ws, "conform-baseline.yaml")
	rec := svc.RecordConformBaseline(path)
	if rec.Error != "" || rec.Violations != 6 || rec.Replaced {
		t.Fatalf("record: %+v", rec)
	}
	if again := svc.RecordConformBaseline(path); !again.Replaced {
		t.Errorf("recording over an existing file says so: %+v", again)
	}
	if ctx := svc.GetConformContext(); len(ctx.Baselines) != 1 || ctx.Baselines[0].Path != path {
		t.Errorf("the page offers the baseline files of the workspace: %+v", ctx.Baselines)
	}

	in.BaselinePath = path
	second := svc.RunConformance(in)
	if second.Error != "" || second.Failed || second.Baseline == nil || len(second.Baseline.Known) != 6 || len(second.Baseline.New) != 0 || len(second.Baseline.Fixed) != 0 || second.Baseline.Legacy {
		t.Fatalf("with the baseline the same violations are accepted: %+v", second.Baseline)
	}
	if k := second.Baseline.Known[0]; k.Operation == "" || k.Check == "" || k.Summary == "" {
		t.Errorf("a known violation names the operation, the check and says what it was: %+v", k)
	}

	// a limited run cannot become a baseline, and a baseline outside the workspace is refused
	limited := in
	limited.BaselinePath, limited.Paths = "", []string{"/products"}
	svc.RunConformance(limited)
	if r := svc.RecordConformBaseline(path); !strings.Contains(r.Error, "limited by tags or path prefixes") {
		t.Errorf("%+v", r)
	}
	bad := in
	bad.BaselinePath = filepath.Join(t.TempDir(), "b.yaml")
	if r := svc.RunConformance(bad); !strings.Contains(r.Error, "outside the open workspace") {
		t.Errorf("%q", r.Error)
	}
	svc.RunConformance(in)
	if r := svc.RecordConformBaseline(filepath.Join(ws, "baseline.txt")); !strings.Contains(r.Error, ".yaml") {
		t.Errorf("%+v", r)
	}
}

func TestConformanceService_ABaselineFromAnOlderVersionStillWorks(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	ws := testWorkspaceDir(t)
	golden, err := os.ReadFile(filepath.Join("testdata", "baseline-v1-0.8.9.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ws, "old-baseline.yaml")
	if err := os.WriteFile(path, golden, 0o600); err != nil {
		t.Fatal(err)
	}
	out := svc.RunConformance(ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: conformanceEnvFile(t), BaselinePath: path})
	if out.Error != "" || out.Failed || out.Baseline == nil || !out.Baseline.Legacy || len(out.Baseline.Known) != 6 {
		t.Fatalf("a 0.8.9 baseline is accepted and flagged: %+v %s", out.Baseline, out.Error)
	}
}
