package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/checks"
)

const failOnSpec = `openapi: "3.0.3"
info: {title: lvl, version: "1"}
paths:
  /n:
    get:
      operationId: getN
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [n], properties: {n: {type: integer}}}
`

func exitCodeOf(err error) int {
	var e *exitError
	if errors.As(err, &e) {
		return e.code
	}
	if err != nil {
		return -1
	}
	return 0
}

// A warning is reported and does not fail the run; --fail-on warning makes it fail. No default
// check is a warning yet, so the test lowers one.
func TestConformFailOn(t *testing.T) {
	t.Cleanup(checks.WithSeverity("CS-RESP-006", checks.Warning))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"n":"text"}`))
	}))
	defer srv.Close()
	spec := filepath.Join(t.TempDir(), "lvl.yaml")
	if err := os.WriteFile(spec, []byte(failOnSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec(t, "conform", "--base-url", srv.URL, spec)
	if exitCodeOf(err) != 0 || !strings.Contains(out, "PASSED with 1 warning") || !strings.Contains(out, "WARN") {
		t.Fatalf("default: exit %d, want 0 with a warning:\n%s", exitCodeOf(err), out)
	}

	out, err = exec(t, "conform", "--base-url", srv.URL, "--fail-on", "warning", spec)
	if exitCodeOf(err) != 1 || !strings.Contains(out, "FAILED") {
		t.Fatalf("--fail-on warning: exit %d, want 1:\n%s", exitCodeOf(err), out)
	}

	_, err = exec(t, "conform", "--base-url", srv.URL, "--fail-on", "note", spec)
	if err == nil || !strings.Contains(err.Error(), "--fail-on") {
		t.Fatalf("--fail-on note must be refused before anything is sent, got %v", err)
	}
}
