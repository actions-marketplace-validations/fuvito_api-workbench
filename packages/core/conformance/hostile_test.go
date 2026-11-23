package conformance_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const hostileSpec = `openapi: "3.0.3"
info: {title: hostile, version: "1"}
servers: [{url: "%s"}]
paths:
  /utf8:
    get:
      responses: {"200": {description: ok, content: {application/json: {schema: {$ref: "#/components/schemas/Item"}}}}}
  /trailing:
    get:
      responses: {"200": {description: ok, content: {application/json: {schema: {$ref: "#/components/schemas/Item"}}}}}
  /huge:
    get:
      responses: {"200": {description: ok, content: {application/json: {schema: {$ref: "#/components/schemas/Item"}}}}}
  /bigint:
    get:
      responses: {"200": {description: ok, content: {application/json: {schema: {$ref: "#/components/schemas/Item"}}}}}
components:
  schemas:
    Item:
      type: object
      required: [id]
      properties:
        id: {type: integer}
        text: {type: string}
        ratio: {type: number}
`

func hostileRun(t *testing.T) *conformance.ConformResult {
	t.Helper()
	bodies := map[string]string{
		"/utf8":     "{\"id\":1,\"text\":\"bad \xff\xfe byte\"}",
		"/trailing": `{"id":1} garbage`,
		"/huge":     `{"id":1,"ratio":1e400}`,
		"/bigint":   `{"id":9223372036854775808}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bodies[r.URL.Path]))
	}))
	t.Cleanup(srv.Close)
	spec, err := openapi.ParseBytes([]byte(fmt.Sprintf(hostileSpec, srv.URL)), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, "", domain.NewEnvironment("e"), conformance.Options{})
}

// F-026 and F-027: a body that is not strictly JSON is a failure, not a pass.
func TestConform_RejectsBodiesThatAreNotStrictJSON(t *testing.T) {
	res := hostileRun(t)
	for path, want := range map[string]string{"/utf8": "not UTF-8", "/trailing": "text after the JSON value"} {
		f := findingOf(t, res, path)
		if f.Kind != conformance.FindingInvalidBody || f.CheckID != "CS-RESP-008" || !strings.Contains(f.Got+f.Detail, want) {
			t.Errorf("%s: kind=%s id=%s got=%q detail=%q, want invalid-body CS-RESP-008 naming %q", path, f.Kind, f.CheckID, f.Got, f.Detail, want)
		}
	}
}

// F-028: a number beyond a 64-bit float is its own finding with a check ID and plain words.
func TestConform_NumberBeyondFloat64HasItsOwnFinding(t *testing.T) {
	f := findingOf(t, hostileRun(t), "/huge")
	if f.Kind != conformance.FindingNumberRange || f.CheckID != "CS-RESP-015" || f.Where != "ratio" {
		t.Fatalf("kind=%s id=%s where=%q\n%+v", f.Kind, f.CheckID, f.Where, f)
	}
	if strings.Contains(f.Summary, "strconv") || !strings.Contains(f.Summary, "1e400") {
		t.Errorf("the summary names the number, not a Go function: %s", f.Summary)
	}
}

// F-031 (conform side): an integer above 2^53 in a body is accepted as an integer.
func TestConform_AcceptsAnIntegerBeyondInt64(t *testing.T) {
	for _, op := range hostileRun(t).Operations {
		if op.Path == "/bigint" && !op.Passed() {
			t.Errorf("a large integer is valid for type integer: %+v", op.Violations)
		}
	}
}

const ndjsonSpec = `openapi: "3.0.3"
info: {title: nd, version: "1"}
servers: [{url: "%s"}]
paths:
  /events:
    get:
      responses:
        "200":
          description: ok
          content:
            application/x-ndjson: {schema: {type: string}}
  /problem:
    get:
      responses:
        "200":
          description: ok
          content:
            application/problem+json: {schema: {type: object}}
`

// F-033: a media type that only contains "json" is not read as one JSON document, but +json is.
func TestConform_NDJSONIsNotOneDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events" {
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte("{\"a\":1}\n{\"a\":2}\n{\"a\":3}\n"))
			return
		}
		w.Header().Set("Content-Type", "application/problem+json")
		_, _ = w.Write([]byte(`{"a":1} trailing`))
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(fmt.Sprintf(ndjsonSpec, srv.URL)), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	res := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, "", domain.NewEnvironment("e"), conformance.Options{})
	for _, op := range res.Operations {
		switch op.Path {
		case "/events":
			if !op.Passed() {
				t.Errorf("NDJSON must not fail as a single document: %+v", op.Violations)
			}
		case "/problem":
			if op.Passed() {
				t.Error("a +json body with text after the value still fails")
			}
		}
	}
}
