package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const slowSpec = `openapi: "3.0.3"
info: {title: slow, version: "1"}
paths:
  /thing:
    get:
      operationId: getThing
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
`

// --max-response-time fails a slow operation and still compares its response with the spec.
func TestConformMaxResponseTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"no id"}`))
	}))
	defer srv.Close()
	spec := filepath.Join(t.TempDir(), "slow.yaml")
	if err := os.WriteFile(spec, []byte(slowSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec(t, "conform", "--base-url", srv.URL, "--max-response-time", "30", "--explain", spec)
	if exitCode(err) != 1 {
		t.Fatalf("a slow response must fail the run, got %v\n%s", err, out)
	}
	for _, want := range []string{"slower than the 30 ms limit", "[CS-RESP-014]", `"id" is missing`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q (the spec comparison must still happen):\n%s", want, out)
		}
	}

	// Without the flag the run reports only the spec problem.
	out, _ = exec(t, "conform", "--base-url", srv.URL, spec)
	if strings.Contains(out, "slower than") {
		t.Errorf("no limit means no slow-response violation:\n%s", out)
	}
}
