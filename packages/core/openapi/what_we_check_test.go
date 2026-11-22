package openapi_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// These tests are the evidence behind apps/desktop/wails/help/what-we-check.md. Each one pins what the response checks
// do TODAY, including the things they do not catch (marked KnownGap). If behaviour changes, a test
// here fails and the catalog has to be updated with it.

const whatWeCheckSpec = `
openapi: "3.0.3"
info: {title: w, version: "1"}
paths:
  /open:
    get:
      operationId: open
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /strict:
    get:
      operationId: strict
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, additionalProperties: false, required: [id], properties: {id: {type: integer}}}
  /constraints:
    get:
      operationId: constraints
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  kind: {type: string, enum: [a, b]}
                  age: {type: integer, minimum: 0, maximum: 120}
                  code: {type: string, minLength: 3, maxLength: 5, pattern: "^[A-Z]+$"}
                  when: {type: string, format: date-time}
                  mail: {type: string, format: email}
                  uid: {type: string, format: uuid}
                  tags: {type: array, items: {type: string}, uniqueItems: true, minItems: 1}
                  maybe: {type: string, nullable: true}
                  secret: {type: string, writeOnly: true}
  /either:
    get:
      operationId: either
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - {type: object, required: [a], properties: {a: {type: string}}}
                  - {type: object, required: [b], properties: {b: {type: string}}}
  /problem:
    get:
      operationId: problem
      responses:
        "200":
          description: ok
          content:
            application/problem+json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /text:
    get:
      operationId: text
      responses:
        "200":
          description: ok
          content:
            text/plain:
              schema: {type: string, minLength: 3}
  /binary:
    get:
      operationId: binary
      responses:
        "200":
          description: ok
          content:
            application/octet-stream:
              schema: {type: string, format: binary}
  /xml:
    get:
      operationId: xml
      responses:
        "200":
          description: ok
          content:
            application/xml:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /wild:
    get:
      operationId: wild
      responses:
        "2XX":
          description: any success
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /withdefault:
    get:
      operationId: withdefault
      responses:
        "200": {description: ok}
        default: {description: any other}
  /hdr:
    get:
      operationId: hdr
      responses:
        "200":
          description: ok
          headers:
            X-Opt: {schema: {type: integer}}
          content:
            application/json:
              schema: {type: object}
  /nocontent:
    get:
      operationId: nocontent
      responses:
        "200": {description: ok}
`

func whatWeCheck(t *testing.T) *openapi.Spec {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(whatWeCheckSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

func ctype(v string) []domain.Header { return []domain.Header{{Name: "Content-Type", Value: v}} }

func byPath(spec *openapi.Spec, path string, status int, hs []domain.Header, body string) int {
	return len(openapi.ValidateResponseByPath(spec, "GET", path, status, &domain.Response{StatusCode: status, Headers: hs, Body: []byte(body)}))
}

func byOperation(spec *openapi.Spec, op string, status int, hs []domain.Header, body string) int {
	return len(openapi.ValidateResponse(spec, op, &domain.Response{StatusCode: status, Headers: hs, Body: []byte(body)}))
}

// CS-RESP-005 .. 010, CS-GAP-001: what the schema check catches in a body.
func TestCheck_ResponseBody(t *testing.T) {
	spec := whatWeCheck(t)
	js := ctype("application/json")
	cases := []struct {
		name, path, op, body string
		caught               bool
	}{
		{"CS-RESP-005 required field missing", "/open", "open", `{"x":1}`, true},
		{"CS-RESP-006 field of the wrong type", "/open", "open", `{"id":"1"}`, true},
		{"CS-RESP-006 array where an object is declared", "/open", "open", `[1]`, true},
		{"CS-RESP-007 enum value", "/constraints", "constraints", `{"kind":"z"}`, true},
		{"CS-RESP-007 minimum", "/constraints", "constraints", `{"age":-1}`, true},
		{"CS-RESP-007 maximum", "/constraints", "constraints", `{"age":500}`, true},
		{"CS-RESP-007 minLength", "/constraints", "constraints", `{"code":"AB"}`, true},
		{"CS-RESP-007 pattern", "/constraints", "constraints", `{"code":"abc"}`, true},
		{"CS-RESP-007 date-time format", "/constraints", "constraints", `{"when":"yesterday"}`, true},
		{"CS-RESP-007 uniqueItems", "/constraints", "constraints", `{"tags":["a","a"]}`, true},
		{"CS-RESP-007 minItems", "/constraints", "constraints", `{"tags":[]}`, true},
		{"CS-RESP-007 null where not nullable", "/constraints", "constraints", `{"kind":null}`, true},
		{"CS-RESP-009 oneOf matches no branch", "/either", "either", `{"c":1}`, true},
		{"nullable accepts null", "/constraints", "constraints", `{"maybe":null}`, false},
		{"oneOf matches one branch", "/either", "either", `{"a":"x"}`, false},
		{"CS-RESP-010 extra property when additionalProperties is false", "/strict", "strict", `{"id":1,"extra":true}`, true},
		// Known gaps, pinned so the catalog stays honest.
		{"CS-RESP-010 KnownGap extra property on an open schema is allowed (the OpenAPI default)", "/open", "open", `{"id":1,"extra":true}`, false},
		{"CS-GAP-001 KnownGap format email is not validated", "/constraints", "constraints", `{"mail":"nope"}`, false},
		{"CS-GAP-001 KnownGap format uuid is not validated", "/constraints", "constraints", `{"uid":"x"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// both checkers use the same schema validation for the body
			if got := byPath(spec, c.path, 200, js, c.body) > 0; got != c.caught {
				t.Errorf("ValidateResponseByPath caught=%v, want %v", got, c.caught)
			}
			if got := byOperation(spec, c.op, 200, js, c.body) > 0; got != c.caught {
				t.Errorf("ValidateResponse (schema assertion) caught=%v, want %v", got, c.caught)
			}
		})
	}
}

// CS-RESP-011 and CS-RESP-001 are where the two checkers differ.
func TestCheck_ResponseOnlyRules(t *testing.T) {
	spec := whatWeCheck(t)
	js := ctype("application/json")
	if byPath(spec, "/constraints", 200, js, `{"secret":"x"}`) == 0 {
		t.Error("CS-RESP-011: a writeOnly property sent in a response is caught by ValidateResponseByPath")
	}
	if byOperation(spec, "constraints", 200, js, `{"secret":"x"}`) != 0 {
		t.Error("CS-RESP-011 KnownGap: the schema assertion does not apply writeOnly")
	}
}

// CS-RESP-008, CS-RESP-012: invalid or empty bodies, and media types the validator cannot read.
func TestCheck_ResponseBodyFormat(t *testing.T) {
	spec := whatWeCheck(t)
	js := ctype("application/json")
	if byPath(spec, "/open", 200, js, `{not json`) == 0 || byOperation(spec, "open", 200, js, `{not json`) == 0 {
		t.Error("CS-RESP-008: invalid JSON is caught by both checkers")
	}
	if byPath(spec, "/open", 200, js, ``) == 0 {
		t.Error("CS-RESP-008: an empty body where a schema is declared is caught by ValidateResponseByPath")
	}
	if byOperation(spec, "open", 200, js, ``) != 0 {
		t.Error("CS-RESP-008 KnownGap: the schema assertion does not check an empty body")
	}
	// CS-RESP-012: a media type without a decoder (XML, ...) is not checked; it is not a violation.
	xml := ctype("application/xml")
	if byPath(spec, "/xml", 200, xml, `<r><id>1</id></r>`) != 0 {
		t.Error("CS-RESP-012: a valid XML body must not be reported as a violation")
	}
	if byOperation(spec, "xml", 200, xml, `<r><id>1</id></r>`) != 0 {
		t.Error("CS-RESP-012: the schema assertion must not read an XML body as JSON")
	}
}

// CS-RESP-002 and the media types that are read: +json, text/plain, parameters on the type.
func TestCheck_ResponseMediaTypes(t *testing.T) {
	spec := whatWeCheck(t)
	if byPath(spec, "/problem", 200, ctype("application/problem+json"), `{"id":1}`) != 0 ||
		byPath(spec, "/problem", 200, ctype("application/problem+json"), `{"x":1}`) == 0 {
		t.Error("a +json media type is read as JSON")
	}
	if byPath(spec, "/text", 200, ctype("text/plain"), `abc`) != 0 || byPath(spec, "/text", 200, ctype("text/plain"), `ab`) == 0 {
		t.Error("text/plain bodies are checked against their schema")
	}
	if byPath(spec, "/binary", 200, ctype("application/octet-stream"), "\x00\x01") != 0 {
		t.Error("a binary body has nothing to compare")
	}
	if byPath(spec, "/open", 200, ctype("application/json; charset=utf-8"), `{"id":1}`) != 0 {
		t.Error("parameters such as charset are ignored when matching the media type")
	}
	if byPath(spec, "/open", 200, nil, `{"id":1}`) != 0 {
		t.Error("CS-NC-008: a response without a Content-Type header is read as JSON (lenient)")
	}
	if byPath(spec, "/open", 200, ctype("text/plain"), `{"id":1}`) == 0 {
		t.Error("CS-RESP-002: a Content-Type the spec does not declare is caught by ValidateResponseByPath")
	}
	if byOperation(spec, "open", 200, ctype("text/plain"), `{"x":1}`) == 0 {
		t.Error("the schema assertion ignores the Content-Type and checks the body as JSON")
	}
}

// CS-RESP-001: status codes, ranges and default responses.
func TestCheck_ResponseStatus(t *testing.T) {
	spec := whatWeCheck(t)
	js := ctype("application/json")
	if byPath(spec, "/open", 500, js, `{}`) == 0 {
		t.Error("CS-RESP-001: an undeclared status is caught by ValidateResponseByPath")
	}
	if byOperation(spec, "open", 500, js, `{"x":1}`) != 0 {
		t.Error("CS-RESP-001 KnownGap: the schema assertion does not check the status")
	}
	if byPath(spec, "/wild", 201, js, `{"id":1}`) != 0 || byPath(spec, "/wild", 201, js, `{"x":1}`) == 0 {
		t.Error("a 2XX range declares every 2xx status, and its schema applies")
	}
	if byPath(spec, "/wild", 404, js, `{}`) == 0 {
		t.Error("CS-RESP-001: a status outside the declared range is caught")
	}
	if byPath(spec, "/withdefault", 503, js, `{}`) != 0 {
		t.Error("a default response declares every other status")
	}
}

// CS-RESP-003, CS-RESP-004: declared response headers.
func TestCheck_ResponseHeaders(t *testing.T) {
	spec := whatWeCheck(t)
	js := ctype("application/json")
	if byPath(spec, "/hdr", 200, append(ctype("application/json"), domain.Header{Name: "X-Opt", Value: "abc"}), `{}`) == 0 {
		t.Error("CS-RESP-004: a declared header sent with the wrong type is caught, even when it is optional")
	}
	if byPath(spec, "/hdr", 200, js, `{}`) != 0 {
		t.Error("an optional header that is absent is fine")
	}
	if byPath(spec, "/hdr", 200, append(ctype("application/json"), domain.Header{Name: "X-Other", Value: "1"}), `{}`) != 0 {
		t.Error("CS-GAP-003 KnownGap: a header the spec does not declare is not reported")
	}
	if byOperation(spec, "hdr", 200, append(ctype("application/json"), domain.Header{Name: "X-Opt", Value: "abc"}), `{}`) != 0 {
		t.Error("the schema assertion ignores headers")
	}
}

// CS-NC-001 .. 003: outcomes that look like a pass but mean "nothing was checked".
func TestCheck_NotChecked(t *testing.T) {
	spec := whatWeCheck(t)
	js := ctype("application/json")
	if byPath(spec, "/nowhere", 200, js, `{}`) != 0 || spec.HasOperation("GET", "/nowhere") {
		t.Error("CS-NC-001: a path the spec does not declare is skipped silently; HasOperation tells it apart")
	}
	if byPath(spec, "/nocontent", 200, js, `anything`) != 0 {
		t.Error("CS-NC-002: a response with no declared content has nothing to compare")
	}
	if byPath(spec, "/binary", 200, ctype("application/octet-stream"), "x") != 0 {
		t.Error("CS-NC-003: a binary body is accepted without comparison")
	}
	if n := byOperation(spec, "nothere", 200, js, `{}`); n == 0 {
		t.Error("an unknown operationId is reported by the schema assertion")
	}
}

// CS-GAP-006: requests are generated with a JSON, form or multipart body; an operation that takes
// an XML body is generated without one.
func TestCheck_XMLRequestBodiesAreNotGenerated_KnownGap(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(`
openapi: "3.0.3"
info: {title: f, version: "1"}
paths:
  /xml:
    post:
      operationId: xml
      requestBody:
        content:
          application/xml:
            schema: {type: object, required: [a], properties: {a: {type: string}}}
      responses: {"201": {description: ok}}
`), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	reqs, err := openapi.GenerateRequests(spec, "http://x")
	if err != nil || len(reqs) != 1 {
		t.Fatalf("%v %d", err, len(reqs))
	}
	if reqs[0].Request.Body != nil {
		t.Errorf("CS-GAP-006: no body is generated for an XML request body today, got %+v", reqs[0].Request.Body)
	}
}
