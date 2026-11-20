package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
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

const requestFindingsSpec = `openapi: "3.0.3"
info: {title: r, version: "1"}
paths:
  /items/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: integer}}
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, required: [price, title], properties: {price: {type: number}, title: {type: string}}}
      responses: {"201": {description: ok}}
`

// When the environment gives a value of the wrong type, the report says which field or parameter,
// what it should have been, and which environment value to change.
func TestFindings_RequestProblemsSayWhatWhereWhyAndHowToFix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) }))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(requestFindingsSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	run := func(id, price string) conformance.OperationResult {
		env := &domain.Environment{Name: "e", Variables: []domain.EnvironmentVariable{
			{Name: "id", Value: id}, {Name: "price", Value: price}, {Name: "title", Value: "Desk"},
		}}
		return conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, env, conformance.Options{}).Operations[0]
	}
	only := func(op conformance.OperationResult) conformance.Finding {
		t.Helper()
		f := conformance.Diagnose(op)
		if op.Passed() || len(op.RequestViolations) != 1 || len(f) != 1 {
			t.Fatalf("expected one request violation (the first problem found): %+v", op)
		}
		return f[0]
	}

	// First problem: the path value is not an integer.
	param := only(run("abc", `"cheap"`))
	if param.Kind != conformance.FindingRequest || param.Part != "request path parameter" || param.Where != "id" ||
		param.Spec != "POST /items/{id} → parameters → id (in path)" {
		t.Fatalf("parameter finding: %+v", param)
	}
	if !strings.Contains(param.Summary, `"id"`) || !strings.Contains(param.Got, "abc") || !strings.Contains(param.Fix, "id=…") ||
		!strings.Contains(param.Why, "environment") && !strings.Contains(param.Why, "declared type") || param.Detail == "" {
		t.Errorf("the parameter finding must name the parameter, the value and the fix: %+v", param)
	}

	// With that fixed, the next one shows: a text where a number is needed.
	body := only(run("1", `"cheap"`))
	if body.Kind != conformance.FindingRequest || body.Part != "request body" || body.Where != "price" || body.Spec != "POST /items/{id} → request body" {
		t.Fatalf("body finding: %+v", body)
	}
	if body.Summary != `The request sends "price" as text, the spec says number` || body.Expected != "number" ||
		!strings.Contains(body.Fix, "Set price to a number in the environment") || !strings.Contains(body.Why, "environment") {
		t.Errorf("the body finding must say what, why and how to fix it: %+v", body)
	}

	// And with both right, the operation passes.
	if op := run("1", "9.99"); !op.Passed() {
		t.Fatalf("a valid request must pass: %+v", op)
	}
}
