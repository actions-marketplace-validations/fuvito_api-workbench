package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// start runs the example server, as it is at startup, behind wrap (nil for the server itself).
func start(t *testing.T, wrap func(http.Handler) http.Handler) string {
	t.Helper()
	resetState()
	var h http.Handler = newMux()
	if wrap != nil {
		h = wrap(h)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func exampleSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := openapi.ParseBytes(data, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

// exampleEnv loads a shipped environment file and points it at the test server.
func exampleEnv(t *testing.T, name, baseURL string) *domain.Environment {
	t.Helper()
	path := filepath.Join("..", "environments", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	env, err := serialization.ParseEnvironmentFile(data, path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range env.Variables {
		if env.Variables[i].Name == "baseUrl" {
			env.Variables[i].Value = baseURL
		}
	}
	return env
}

// The example server is what the README sends a new user to, so it has to follow its own spec.
func TestServerConformsToItsSpec(t *testing.T) {
	url := start(t, nil)
	result := conformance.New(executor.New(executor.Options{})).Run(context.Background(), exampleSpec(t), url, exampleEnv(t, "demo-conform.yaml", url), conformance.Options{})
	if len(result.Operations) != 8 {
		t.Fatalf("operations: got %d, want 8", len(result.Operations))
	}
	for _, op := range result.Operations {
		if op.Skipped || op.StatusCode == 0 || !result.OpPassed(op) {
			t.Errorf("%s %s: status %d, blocked %q, error %q, violations %v", op.Method, op.SpecPath, op.StatusCode, op.Blocked, op.Error, op.Violations)
		}
	}
}

// A probe passes on any 4xx (CS-GAP-012), so a pass alone does not show why the server said no.
// Here each probe must be rejected for the thing it tests.
func TestNegativeProbesAreRejectedForTheTestedReason(t *testing.T) {
	url := start(t, nil)
	result := conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), exampleSpec(t), url, exampleEnv(t, "demo-conform.yaml", url), conformance.Options{})
	want := map[openapi.ProbeKind][]int{
		openapi.ProbeMissingAuth:       {http.StatusUnauthorized},
		openapi.ProbeDropRequiredField: {http.StatusBadRequest, http.StatusUnprocessableEntity},
		openapi.ProbeWrongFieldType:    {http.StatusBadRequest, http.StatusUnprocessableEntity},
	}
	seen := map[openapi.ProbeKind]int{}
	for _, p := range result.Probes {
		seen[p.Kind]++
		if p.Skipped || !slices.Contains(want[p.Kind], p.StatusCode) {
			t.Errorf("%s %s, %s: status %d (skipped %v), want one of %v", p.Method, p.Path, p.Description, p.StatusCode, p.Skipped, want[p.Kind])
		}
	}
	for kind := range want {
		if seen[kind] == 0 {
			t.Errorf("no %s probe ran, so nothing was proven about it", kind)
		}
	}
}

// Error responses are compared with the ProblemDetails schema, not just labelled with its media type.
func TestAProblemBodyThatBreaksTheSpecIsReported(t *testing.T) {
	url := start(t, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/users/999" {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"type":"/errors/not-found","title":"Not Found","status":404}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	env := exampleEnv(t, "demo-conform.yaml", url)
	for i := range env.Variables {
		if env.Variables[i].Name == "id" {
			env.Variables[i].Value = "999"
		}
	}
	result := conformance.New(executor.New(executor.Options{})).Run(context.Background(), exampleSpec(t), url, env, conformance.Options{ReadOnly: true})
	for _, op := range result.Operations {
		if op.Method == "GET" && op.SpecPath == "/users/{id}" {
			if len(op.Violations) != 1 || !strings.Contains(op.Violations[0].Error(), "detail") {
				t.Fatalf("a problem body without detail must be one violation naming it, got %v", op.Violations)
			}
			return
		}
	}
	t.Fatal("GET /users/{id} was not run")
}

func TestDeclaredAuthenticationIsEnforced(t *testing.T) {
	url := start(t, nil)
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/users", http.StatusUnauthorized},
		{"POST", "/users", http.StatusUnauthorized},
		{"GET", "/protected", http.StatusUnauthorized},
		{"DELETE", "/users", http.StatusMethodNotAllowed}, // not declared: no token needed to be told so
		{"GET", "/users/1", http.StatusOK},                // the spec declares no security here
	} {
		req, err := http.NewRequest(tc.method, url+tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("%s %s without a token: got %d, want %d", tc.method, tc.path, resp.StatusCode, tc.want)
		}
	}
}

func TestResetRestoresTheSampleUsers(t *testing.T) {
	url := start(t, nil)
	do := func(method, path string) int {
		req, err := http.NewRequest(method, url+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	for round := 1; round <= 2; round++ { // twice: a delete must not damage what reset restores from
		if got := do("DELETE", "/users/1"); got != http.StatusNoContent {
			t.Fatalf("round %d: delete: got %d", round, got)
		}
		if got := do("GET", "/users/1"); got != http.StatusNotFound {
			t.Fatalf("round %d: after delete: got %d", round, got)
		}
		if got := do("POST", "/admin/reset"); got != http.StatusOK {
			t.Fatalf("round %d: reset: got %d", round, got)
		}
		for _, id := range []string{"1", "2", "5"} {
			if got := do("GET", "/users/"+id); got != http.StatusOK {
				t.Errorf("round %d: user %s after reset: got %d", round, id, got)
			}
		}
	}
}

// Every shipped scenario does what its description says against this server, with the local
// environment the documents use. contract-violation is written to fail in every step.
func TestExampleScenarios(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "scenarios", "*.yaml"))
	if err != nil || len(files) < 10 {
		t.Fatalf("example scenarios: %v %v", files, err)
	}
	spec := exampleSpec(t)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			url := start(t, nil)
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			sc, err := serialization.ParseScenarioFile(data, file)
			if err != nil {
				t.Fatal(err)
			}
			result := scenario.NewWithSpec(executor.New(executor.Options{}), spec).Run(context.Background(), sc, exampleEnv(t, "local.yaml", url))
			passed, failed, skipped := result.Counts()
			if filepath.Base(file) == "contract-violation.yaml" {
				if passed != 0 || skipped != 0 || failed == 0 {
					t.Errorf("every step is meant to fail: %d passed, %d failed, %d skipped", passed, failed, skipped)
				}
				return
			}
			if !result.Passed || failed != 0 || skipped != 0 {
				t.Errorf("%d passed, %d failed, %d skipped", passed, failed, skipped)
				for _, step := range result.Steps {
					for _, a := range step.Assertions {
						if !a.Passed {
							t.Logf("failed assertion: %+v", a)
						}
					}
				}
			}
		})
	}
}
