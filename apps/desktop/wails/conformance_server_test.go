package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// Each operation is called once and validated against the embedded spec, the
// same way scenario steps are. Exactly the documented operations must fail.
func TestConformanceServerViolationsAreExactlyTheKnownOnes(t *testing.T) {
	spec, err := openapi.ParseBytes(conformanceOpenAPIYAML, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	calls := []struct {
		op, method, path, body string
	}{
		{"getHealth", "GET", "/health", ""},
		{"listProducts", "GET", "/products", ""},
		{"createProduct", "POST", "/products", `{"name":"Lamp","price":12.5}`},
		{"getProduct", "GET", "/products/1", ""},
		{"deleteProduct", "DELETE", "/products/2", ""},
		{"listCategories", "GET", "/categories", ""},
		{"listOrders", "GET", "/orders", ""},
		{"listWidgets", "GET", "/widgets", ""},
		{"getInventory", "GET", "/inventory", ""},
		{"getStatus", "GET", "/status", ""},
	}
	var failing []string
	for _, c := range calls {
		code, hdr, body := doDemo(t, newConformanceMux(), demoCall{
			method: c.method, path: c.path, body: c.body,
			headers: map[string]string{"Content-Type": "application/json"},
		})
		resp := &domain.Response{StatusCode: code, Body: []byte(body), Headers: toDomainHeaders(hdr)}
		if v := openapi.ValidateResponseByPath(spec, c.method, c.path, code, resp); len(v) > 0 {
			failing = append(failing, c.op)
		}
	}
	want := append([]string(nil), conformanceKnownViolations...)
	sort.Strings(want)
	sort.Strings(failing)
	if len(failing) != len(want) {
		t.Fatalf("failing = %v, want %v", failing, want)
	}
	for i := range want {
		if failing[i] != want[i] {
			t.Fatalf("failing = %v, want %v", failing, want)
		}
	}
}

func toDomainHeaders(h http.Header) []domain.Header {
	var out []domain.Header
	for k, vs := range h {
		for _, v := range vs {
			out = append(out, domain.Header{Name: k, Value: v})
		}
	}
	return out
}

func TestChoosePort(t *testing.T) {
	// Automatic: the first free port from the start.
	auto, err := choosePort(0, 8300, "x")
	if err != nil || auto < 8300 || auto >= 8330 {
		t.Fatalf("auto = %d, %v", auto, err)
	}

	// Configured and free: exactly that port.
	free, err := findFreePort(8400)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := choosePort(free, 8300, "x"); err != nil || got != free {
		t.Fatalf("configured free port: got %d, %v", got, err)
	}

	// Configured and taken: an error that says where to change it, never another port.
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", free))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, err = choosePort(free, 8300, "the Conformance Server port")
	if err == nil || !strings.Contains(err.Error(), "already in use") || !strings.Contains(err.Error(), "the Conformance Server port") {
		t.Fatalf("taken port: %v", err)
	}
}

func TestConformanceServerStartUsesTheConfiguredPort(t *testing.T) {
	free, err := findFreePort(8500)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewConformanceServerService(nil)
	svc.portFn = func() int { return free }
	res := svc.Start()
	defer svc.Stop()
	if res.Error != "" || res.Port != free {
		t.Fatalf("start: %+v, want port %d", res, free)
	}

	other := NewConformanceServerService(nil)
	other.portFn = func() int { return free }
	if res := other.Start(); res.Error == "" || res.Port != 0 {
		t.Fatalf("a taken configured port must fail: %+v", res)
	}
}

func TestSanitizePort(t *testing.T) {
	for in, want := range map[int]int{0: 0, 80: 0, 1023: 0, 1024: 1024, 8090: 8090, 65535: 65535, 65536: 0, -5: 0} {
		if got := sanitizePort(in); got != want {
			t.Errorf("sanitizePort(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestConformanceServerServesItsSpecAndEnvironment(t *testing.T) {
	h := newConformanceMux()
	for path, want := range map[string]string{"/openapi.yaml": "Clear Signal Conformance Server", "/environment.yaml": "name: id"} {
		code, _, body := doDemo(t, h, demoCall{method: "GET", path: path})
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("%s: %d, missing %q", path, code, want)
		}
	}
}

func TestConformanceOpenWorkspace_OpensAndRestores(t *testing.T) {
	isolateHome(t)
	ws := NewWorkspaceService()
	previous := t.TempDir()
	ws.workspacePath = previous
	svc := NewConformanceServerService(ws)
	svc.portFn = func() int { return 0 }

	res := svc.OpenWorkspace()
	if res.Error != "" || res.WorkspacePath == "" || res.Port == 0 {
		t.Fatalf("open: %+v", res)
	}
	if ws.workspacePath != res.WorkspacePath {
		t.Fatalf("workspace = %q, want the temporary one %q", ws.workspacePath, res.WorkspacePath)
	}
	spec := filepath.Join(res.WorkspacePath, "specs", "conformance-openapi.yaml")
	if _, err := os.Stat(spec); err != nil {
		t.Fatalf("spec missing: %v", err)
	}
	if ws.activeEnv == nil {
		t.Fatal("the local environment should be active")
	}
	vars := map[string]string{}
	for _, v := range ws.activeEnv.Variables {
		vars[v.Name] = v.Value
	}
	if vars["id"] != "1" || vars["name"] != "Lamp" || vars["baseUrl"] != fmt.Sprintf("http://localhost:%d", res.Port) {
		t.Fatalf("environment variables: %v", vars)
	}

	// Calling it again is harmless and keeps the same workspace.
	if again := svc.OpenWorkspace(); again.WorkspacePath != res.WorkspacePath || again.Port != res.Port {
		t.Fatalf("second open: %+v", again)
	}

	// The page works against it with no further setup: exactly the known violations.
	out := NewConformanceService(ws, NewSettingsService()).RunConformance(ConformRunInput{
		SpecPath: spec, BaseURL: fmt.Sprintf("http://localhost:%d", res.Port),
	})
	if out.Error != "" || out.Failing != len(conformanceKnownViolations) || out.Passing != 2 {
		t.Fatalf("run in the opened workspace: %+v", out)
	}

	svc.Stop()
	if ws.workspacePath != previous {
		t.Fatalf("workspace = %q, want the previous %q", ws.workspacePath, previous)
	}
	if _, err := os.Stat(res.WorkspacePath); !os.IsNotExist(err) {
		t.Fatalf("the temporary workspace should be removed: %v", err)
	}
	if svc.IsRunning() {
		t.Fatal("the server should be stopped")
	}
}

func TestConformanceOpenWorkspace_RefusedWhileDemoWorkspaceIsOpen(t *testing.T) {
	isolateHome(t)
	ws := NewWorkspaceService()
	ws.workspacePath = filepath.Join(t.TempDir(), demoWorkspacePrefix+"123")
	svc := NewConformanceServerService(ws)
	defer svc.Stop()
	if res := svc.OpenWorkspace(); !strings.Contains(res.Error, "Demo Server") || svc.IsRunning() {
		t.Fatalf("must refuse and not start: %+v", res)
	}

	ws.workspacePath = filepath.Join(t.TempDir(), conformanceWorkspacePrefix+"123")
	demo := NewDemoService(ws)
	if res := demo.StartDemo(); !strings.Contains(res.Error, "Conformance Server") {
		t.Fatalf("the demo must refuse while the conformance workspace is open: %+v", res)
	}
}

// The sample scenario in the Conformance workspace must parse, name the spec, and show the two
// documented spec violations (and the passing step) when it is run against the server.
func TestConformanceWorkspaceSampleScenario(t *testing.T) {
	dir := t.TempDir()
	if err := writeConformanceWorkspace(dir, 0); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"requests/health.yaml", "requests/products/get-product.yaml", "requests/categories/list-categories.yaml", "requests/products/list-products.yaml"} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := serialization.ParseRequestFile(data, filepath.Base(name)); err != nil {
			t.Errorf("%s does not parse: %v", name, err)
		}
	}

	scnPath := filepath.Join(dir, "scenarios", "product-lifecycle.yaml")
	data, err := os.ReadFile(scnPath)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := serialization.ParseScenarioFile(data, "product-lifecycle.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if sc.SpecFile != "../specs/conformance-openapi.yaml" {
		t.Fatalf("specFile = %q", sc.SpecFile)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(scnPath), filepath.FromSlash(sc.SpecFile))); err != nil {
		t.Fatalf("the spec the scenario names is not in the workspace: %v", err)
	}

	srv := httptest.NewServer(newConformanceMux())
	defer srv.Close()
	spec, err := openapi.ParseBytes(conformanceOpenAPIYAML, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	envData, _ := os.ReadFile(conformanceEnvFile(t))
	env, err := serialization.ParseEnvironmentFile(envData, "env.yaml")
	if err != nil {
		t.Fatal(err)
	}
	env.Variables = append(env.Variables, domain.EnvironmentVariable{Name: "baseUrl", Value: srv.URL})
	res := scenario.NewWithSpec(executor.New(executor.Options{}), spec).Run(context.Background(), sc, env)

	var states []scenario.SpecState
	for _, st := range res.Steps {
		states = append(states, st.Spec.State)
	}
	want := []scenario.SpecState{scenario.SpecMatched, scenario.SpecViolations, scenario.SpecViolations}
	if len(states) != 3 || states[0] != want[0] || states[1] != want[1] || states[2] != want[2] {
		t.Fatalf("spec states = %v, want %v", states, want)
	}

	// The scenario result explains each violation like the Check Conformance page: what, where, why, fix.
	for i, st := range res.Steps {
		got := specFindings(st, nil)
		if st.Spec.State != scenario.SpecViolations {
			if len(got) != 0 {
				t.Errorf("step %d matched the spec and must have no findings: %+v", i, got)
			}
			continue
		}
		if len(got) != len(st.Spec.Violations) {
			t.Fatalf("step %d: %d findings for %d violations", i, len(got), len(st.Spec.Violations))
		}
		for _, f := range got {
			if f.Summary == "" || f.Why == "" || f.Fix == "" || f.Detail == "" {
				t.Errorf("step %d: an explained finding needs a summary, why, fix and the raw detail: %+v", i, f)
			}
		}
		if got[0].Spec == "" || !strings.Contains(got[0].Spec, string(st.Request.Method)) {
			t.Errorf("step %d: the finding must say where in the spec the expectation is: %+v", i, got[0])
		}
	}
}
