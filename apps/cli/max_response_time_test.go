package main

import (
	"github.com/fuvito/api-workbench/packages/core/coverage"
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

// Scenario runs with --workspace record each operation's response time in the Contracts store.
func TestRun_RecordsResponseTimesInTheContractsStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(minimalSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	sc := filepath.Join(dir, "s.yaml")
	if err := os.WriteFile(sc, []byte(scenarioExpecting("Timed", srv.URL+"/ping", 200)), 0o600); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	if out, err := exec(t, "run", "--spec", specPath, "--workspace", ws, sc); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	stored := coverage.Load(ws, specPath)
	if len(stored.OperationResults) == 0 {
		t.Fatalf("no operation results stored: %+v", stored)
	}
	for op, r := range stored.OperationResults {
		if r.LastDurationMs < 30 || r.SlowestDurationMs < r.LastDurationMs {
			t.Errorf("%s: last=%d slowest=%d, want at least the 40 ms the server took", op, r.LastDurationMs, r.SlowestDurationMs)
		}
	}
}
