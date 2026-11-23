package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// A response-time limit fails a slow operation with a finding of its own, keeps the response so it
// is still compared with the spec, and leaves baselines stable (no measured time in the violation).
func TestMaxResponseTime_SlowOperationFailsWithAStableViolation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"c":1}`))
	}))
	defer srv.Close()
	spec, _ := openapi.ParseBytes([]byte(`openapi: "3.0.3"
info: {title: s, version: "1"}
paths:
  /obj:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [a], properties: {a: {type: string}}}
`), "yaml")
	run := func(limit int) *conformance.ConformResult {
		return conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, domain.NewEnvironment("e"), conformance.Options{MaxResponseTimeMs: limit})
	}

	res := run(20)
	op := res.Operations[0]
	if len(op.Violations) != 2 {
		t.Fatalf("want the spec violation and the slow one, got %+v", op.Violations)
	}
	fs := conformance.Diagnose(op)
	var slow *conformance.Finding
	for i := range fs {
		if fs[i].Kind == conformance.FindingSlowResponse {
			slow = &fs[i]
		}
	}
	if slow == nil || slow.CheckID != "CS-RESP-014" || !strings.Contains(slow.Summary, "limit of 20 ms") {
		t.Fatalf("slow finding: %+v", slow)
	}
	for _, v := range conformance.Violations(res, nil) {
		if strings.Contains(v.Detail, " ms)") || strings.Contains(v.Detail, "took") {
			t.Errorf("a baseline entry must not hold the measured time: %+v", v)
		}
	}

	if got := run(0).Operations[0].Violations; len(got) != 1 {
		t.Errorf("no limit must add no violation, got %+v", got)
	}
}
