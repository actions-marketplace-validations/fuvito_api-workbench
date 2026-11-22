package conformance_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const traceSpec = `openapi: "3.0.3"
info: {title: t, version: "1"}
servers: [{url: "%s"}]
paths:
  /echo:
    get:
      responses: {"200": {description: ok}}
    trace:
      operationId: echoTrace
      responses:
        "200":
          description: ok
          content:
            text/plain: {schema: {type: string}}
`

// F-025: a TRACE operation is generated and run like any other, and read-only mode skips it
// visibly instead of leaving it out.
func TestConform_RunsTraceOperations(t *testing.T) {
	var traced int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "TRACE" {
			traced++
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("trace ok"))
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(fmt.Sprintf(traceSpec, srv.URL)), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	reqs, _ := openapi.GenerateRequests(spec, srv.URL)
	if len(reqs) != 2 {
		t.Fatalf("generated %d requests, want 2 (GET and TRACE)", len(reqs))
	}
	run := func(o conformance.Options) *conformance.ConformResult {
		return conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, "", domain.NewEnvironment("e"), o)
	}
	res := run(conformance.Options{})
	if len(res.Operations) != 2 || traced != 1 {
		t.Fatalf("operations=%d traced=%d, want 2 and 1", len(res.Operations), traced)
	}
	for _, op := range res.Operations {
		if !op.Passed() {
			t.Errorf("%s %s should pass: %+v", op.Method, op.Path, op)
		}
	}
	ro := run(conformance.Options{ReadOnly: true})
	if len(ro.Operations) != 2 || !ro.Operations[1].Skipped && !ro.Operations[0].Skipped || traced != 1 {
		t.Errorf("read-only lists TRACE as skipped and does not send it: %+v traced=%d", ro.Operations, traced)
	}
}
