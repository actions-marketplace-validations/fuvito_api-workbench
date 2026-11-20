package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const findingsSpec = `openapi: "3.0.3"
info: {title: f, version: "1"}
paths:
  /headered:
    get:
      operationId: headered
      responses:
        "200":
          description: ok
          headers:
            X-Total-Count: {required: true, schema: {type: integer}}
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /badheader:
    get:
      operationId: badHeader
      responses:
        "200":
          description: ok
          headers:
            X-Total-Count: {required: true, schema: {type: integer}}
          content:
            application/json:
              schema: {type: object}
  /text:
    get:
      operationId: text
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /unplanned:
    get:
      operationId: unplanned
      responses:
        "200": {description: ok}
        "404": {description: not found}
  /bodybad:
    get:
      operationId: bodyBad
      responses:
        "200":
          description: ok
          headers:
            X-Total-Count: {required: true, schema: {type: integer}}
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
`

func findingsServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/headered": // body is fine but the required header is missing
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1}`))
		case "/badheader":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Total-Count", "many")
			_, _ = w.Write([]byte(`{}`))
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(`{"x":1}`)) // would also fail the schema, but is never compared
		case "/unplanned":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/bodybad": // header present, body wrong: must still be found
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Total-Count", "3")
			_, _ = w.Write([]byte(`{"x":1}`))
		}
	}))
}

func TestFindings_TellWhatWhyWhereAndHowToFix(t *testing.T) {
	srv := findingsServer()
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(findingsSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	res := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, nil, conformance.Options{})

	want := map[string]struct {
		kind, part, where, spec string
		inSummary, inExpected   string
		inGot, inWhy, inFix     string
	}{
		"headered": {conformance.FindingMissingHeader, "response header", "X-Total-Count", "GET /headered → response 200 → headers → X-Total-Count",
			`"X-Total-Count"`, "X-Total-Count", "not sent", "required", "Send"},
		"badHeader": {conformance.FindingBadHeader, "response header", "X-Total-Count", "GET /badheader → response 200 → headers → X-Total-Count",
			`"X-Total-Count"`, "", "many", "type", "Send a value"},
		"text": {conformance.FindingWrongContentType, "response header", "Content-Type", "GET /text → response 200 → content",
			`"text/plain"`, "application/json", "text/plain", "was not compared", "Content-Type application/json"},
		"unplanned": {conformance.FindingUndeclaredStatus, "response status", "503", "GET /unplanned → responses",
			"503", "200, 404", "503", "200, 404", "add 503"},
		"bodyBad": {conformance.FindingMissingField, "response body", "id", "GET /bodybad → response 200 → body schema",
			`"id"`, "id", "no such field", "required", `Return "id"`},
	}
	for _, op := range res.Operations {
		w, ok := want[op.OperationID]
		if !ok {
			t.Fatalf("unexpected operation %s", op.OperationID)
		}
		if op.Passed() {
			t.Errorf("%s must fail", op.OperationID)
			continue
		}
		f := conformance.Diagnose(op)
		if len(f) != 1 {
			t.Errorf("%s: %d findings: %+v", op.OperationID, len(f), f)
			continue
		}
		g := f[0]
		if g.Kind != w.kind || g.Part != w.part || g.Where != w.where || g.Spec != w.spec {
			t.Errorf("%s: kind/part/where/spec = %q/%q/%q/%q, want %q/%q/%q/%q", op.OperationID, g.Kind, g.Part, g.Where, g.Spec, w.kind, w.part, w.where, w.spec)
		}
		for name, pair := range map[string][2]string{
			"summary": {g.Summary, w.inSummary}, "expected": {g.Expected, w.inExpected}, "got": {g.Got, w.inGot}, "why": {g.Why, w.inWhy}, "fix": {g.Fix, w.inFix},
		} {
			if !strings.Contains(pair[0], pair[1]) {
				t.Errorf("%s: %s = %q, should mention %q", op.OperationID, name, pair[0], pair[1])
			}
		}
		if g.Detail == "" || g.WhereText() == "" {
			t.Errorf("%s: a finding always has the raw message and a location: %+v", op.OperationID, g)
		}
	}
	if _, failed, _ := res.Counts(); failed != 5 {
		t.Fatalf("all five misbehaving operations must fail, failed = %d", failed)
	}
}

func TestFinding_WhereText(t *testing.T) {
	for f, want := range map[conformance.Finding]string{
		{}:                                       "",
		{Part: "response status", Where: "503"}:  "response status 503",
		{Part: "response header", Where: "X-A"}:  `response header "X-A"`,
		{Part: "response body", Where: "[0].id"}: "response body, field [0].id",
		{Part: "response body"}:                  "response body",
		{Part: "request", Where: "id, name"}:     "request id, name",
	} {
		if got := f.WhereText(); got != want {
			t.Errorf("%+v: %q, want %q", f, got, want)
		}
	}
}
