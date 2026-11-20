package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const placeholderSpec = `openapi: "3.0.3"
info: {title: ph, version: "1"}
paths:
  /health:
    get:
      operationId: health
      responses: {"200": {description: ok}}
  /items/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: integer}}
    get:
      operationId: getItem
      responses:
        "200": {description: ok}
        "404": {description: not found}
`

// A request whose placeholder has no value is not sent: before, the server's 404 (declared in the
// spec) made such an operation pass although nothing had been compared.
func TestConformDoesNotSendWhenAValueIsMissing(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/health" || r.URL.Path == "/items/1" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	spec := filepath.Join(t.TempDir(), "ph.yaml")
	if err := os.WriteFile(spec, []byte(placeholderSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec(t, "conform", "--base-url", srv.URL, spec)
	if exitCode(err) != 1 {
		t.Fatalf("an operation that could not be sent fails the run: %v\n%s", err, out)
	}
	for _, want := range []string{
		"FAIL  GET     /items/{{id}}", "✗ not sent: no value for {{id}}",
		"1 operation was not sent because nothing provides: id.", "1 passed  ·  1 failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in:\n%s", want, out)
		}
	}
	mu.Lock()
	for _, p := range paths {
		if strings.HasPrefix(p, "/items") {
			t.Fatalf("%s was sent although {{id}} had no value: %v", p, paths)
		}
	}
	mu.Unlock()

	// Giving the value (here as an override) sends it and the run passes.
	out, err = exec(t, "conform", "--base-url", srv.URL, "--var", "id=1", spec)
	if err != nil || !strings.Contains(out, "PASSED") {
		t.Fatalf("with id=1 it should pass: %v\n%s", err, out)
	}

	// A variable that is defined but never used changes nothing and is not reported.
	out, err = exec(t, "conform", "--base-url", srv.URL, "--var", "id=1", "--var", "neverUsed=x", spec)
	if err != nil || strings.Contains(out, "neverUsed") {
		t.Fatalf("an unused variable must be ignored: %v\n%s", err, out)
	}
}
