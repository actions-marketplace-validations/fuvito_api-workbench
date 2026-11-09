package conformance_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const minimalSpec = `openapi: "3.0.3"
info:
  title: "test-api"
  version: "1.0.0"
paths:
  /ping:
    get:
      operationId: "getPing"
      tags:
        - health
      responses:
        "200":
          description: "OK"
          content:
            application/json:
              schema:
                type: object
  /items:
    post:
      operationId: "createItem"
      tags:
        - items
      responses:
        "201":
          description: "Created"
`

func parseSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(minimalSpec), "yaml")
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	return spec
}

func pingServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/ping":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{}`)
		case r.Method == http.MethodPost && r.URL.Path == "/items":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ---- OperationResult.Passed ----

func TestOperationResult_Passed(t *testing.T) {
	pass := conformance.OperationResult{Method: "GET", Path: "/ping", StatusCode: 200}
	if !pass.Passed() {
		t.Error("want Passed()=true for no error, no violations, not skipped")
	}

	skip := conformance.OperationResult{Skipped: true, StatusCode: 200}
	if skip.Passed() {
		t.Error("want Passed()=false for skipped")
	}

	withErr := conformance.OperationResult{Error: "timeout"}
	if withErr.Passed() {
		t.Error("want Passed()=false when Error is set")
	}
}

// ---- ConformResult.Counts ----

func TestCounts_Mixed(t *testing.T) {
	r := &conformance.ConformResult{
		Operations: []conformance.OperationResult{
			{Method: "GET", Path: "/a", StatusCode: 200},                    // passed
			{Method: "GET", Path: "/b", Error: "refused"},                   // failed
			{Method: "GET", Path: "/c", Skipped: true},                      // skipped
			{Method: "POST", Path: "/d", StatusCode: 201},                   // passed
		},
	}
	passed, failed, skipped := r.Counts()
	if passed != 2 {
		t.Errorf("passed: got %d, want 2", passed)
	}
	if failed != 1 {
		t.Errorf("failed: got %d, want 1", failed)
	}
	if skipped != 1 {
		t.Errorf("skipped: got %d, want 1", skipped)
	}
}

func TestCounts_Empty(t *testing.T) {
	r := &conformance.ConformResult{}
	p, f, s := r.Counts()
	if p != 0 || f != 0 || s != 0 {
		t.Errorf("empty result: got %d/%d/%d", p, f, s)
	}
}

// ---- Runner.Run ----

func TestRunner_Run_AllPass(t *testing.T) {
	srv := pingServer(t)
	spec := parseSpec(t)
	exec := executor.New(executor.Options{})
	runner := conformance.New(exec)

	result := runner.Run(context.Background(), spec, srv.URL, nil, conformance.Options{})

	if result.SpecTitle != "test-api" {
		t.Errorf("specTitle: got %q", result.SpecTitle)
	}
	passed, failed, _ := result.Counts()
	if failed != 0 {
		t.Errorf("want 0 failed, got %d (passed=%d)", failed, passed)
	}
	if passed != 2 {
		t.Errorf("want 2 passed, got %d", passed)
	}
}

func TestRunner_Run_TagFilter(t *testing.T) {
	srv := pingServer(t)
	spec := parseSpec(t)
	exec := executor.New(executor.Options{})
	runner := conformance.New(exec)

	result := runner.Run(context.Background(), spec, srv.URL, nil, conformance.Options{
		Tags: []string{"health"},
	})

	_, _, skipped := result.Counts()
	if skipped != 1 {
		t.Errorf("want 1 skipped (non-health tagged), got %d", skipped)
	}
}

func TestRunner_Run_PathFilter(t *testing.T) {
	srv := pingServer(t)
	spec := parseSpec(t)
	exec := executor.New(executor.Options{})
	runner := conformance.New(exec)

	result := runner.Run(context.Background(), spec, srv.URL, nil, conformance.Options{
		Paths: []string{"/ping"},
	})

	_, _, skipped := result.Counts()
	if skipped != 1 {
		t.Errorf("want 1 skipped (/items filtered out), got %d", skipped)
	}
}

func TestRunner_Run_FailFast(t *testing.T) {
	// Close the server immediately so every execution produces execErr != nil,
	// which is the FailFast trigger path in the runner.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	spec := parseSpec(t)
	exec := executor.New(executor.Options{})
	runner := conformance.New(exec)

	result := runner.Run(context.Background(), spec, url, nil, conformance.Options{FailFast: true})

	total := len(result.Operations)
	if total >= 2 {
		t.Errorf("FailFast: expected fewer than 2 results, got %d", total)
	}
}

func TestRunner_Run_UnreachableServer(t *testing.T) {
	spec := parseSpec(t)
	exec := executor.New(executor.Options{})
	runner := conformance.New(exec)

	result := runner.Run(context.Background(), spec, "http://127.0.0.1:1", nil, conformance.Options{})

	for _, op := range result.Operations {
		if op.Error == "" {
			t.Errorf("op %s %s: want an error for unreachable server", op.Method, op.Path)
		}
	}
}
