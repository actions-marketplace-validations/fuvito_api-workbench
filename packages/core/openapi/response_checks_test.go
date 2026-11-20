package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const responseChecksSpec = `
openapi: "3.0.3"
info: {title: p, version: "1"}
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-Total-Count: {required: true, schema: {type: integer}}
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
        "404": {description: not found}
  /plain:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
        default: {description: any other}
  /gone:
    delete:
      responses:
        "204": {description: gone}
`

func check(t *testing.T, spec *openapi.Spec, method, path string, status int, headers map[string]string, body string) []openapi.ValidationError {
	t.Helper()
	var hs []domain.Header
	for k, v := range headers {
		hs = append(hs, domain.Header{Name: k, Value: v})
	}
	return openapi.ValidateResponseByPath(spec, method, path, status, &domain.Response{StatusCode: status, Headers: hs, Body: []byte(body)})
}

func TestResponseChecks(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(responseChecksSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	json := map[string]string{"Content-Type": "application/json", "X-Total-Count": "3"}

	cases := []struct {
		name    string
		method  string
		path    string
		status  int
		headers map[string]string
		body    string
		kind    string   // "" means no violation
		mention []string // the message must name what and where
	}{
		{"conforming response", "GET", "/a", 200, json, `{"id":1}`, "", nil},
		// These two were silently accepted before: a declared header switched body checking off.
		{"body violation with a declared header present", "GET", "/a", 200, json, `{"x":1}`, openapi.KindBody, []string{"id"}},
		{"invalid JSON with a declared header present", "GET", "/a", 200, json, `{not json`, openapi.KindBody, nil},
		{"required header not sent", "GET", "/a", 200, map[string]string{"Content-Type": "application/json"}, `{"id":1}`,
			openapi.KindMissingHeader, []string{"GET /a", "200", `"X-Total-Count"`, "required", "not sent"}},
		{"header with the wrong type", "GET", "/a", 200, map[string]string{"Content-Type": "application/json", "X-Total-Count": "many"}, `{"id":1}`,
			openapi.KindBadHeader, []string{"GET /a", "200", `"X-Total-Count"`}},
		{"wrong Content-Type", "GET", "/a", 200, map[string]string{"Content-Type": "text/plain", "X-Total-Count": "3"}, `{"id":1}`,
			openapi.KindContentType, []string{"GET /a", "200", `"text/plain"`, "application/json", "body was not checked"}},
		{"undeclared status", "GET", "/a", 500, json, `{}`,
			openapi.KindUndeclaredStatus, []string{"GET /a", "answered 500", "200, 404"}},
		{"a declared 404 is fine", "GET", "/a", 404, nil, ``, "", nil},
		{"undeclared status is fine when there is a default response", "GET", "/plain", 503, json, `{}`, "", nil},
		{"wrong Content-Type without any declared header", "GET", "/plain", 200, map[string]string{"Content-Type": "text/html"}, `<html>`,
			openapi.KindContentType, []string{"GET /plain", `"text/html"`}},
		{"declared 204", "DELETE", "/gone", 204, nil, ``, "", nil},
		{"undeclared 200 on a 204-only operation", "DELETE", "/gone", 200, nil, `ok`,
			openapi.KindUndeclaredStatus, []string{"DELETE /gone", "answered 200", "204"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := check(t, spec, c.method, c.path, c.status, c.headers, c.body)
			if c.kind == "" {
				if len(got) != 0 {
					t.Fatalf("expected no violation, got %v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Kind != c.kind {
				t.Fatalf("expected one %s violation, got %+v", c.kind, got)
			}
			msg := got[0].Error()
			for _, want := range c.mention {
				if !strings.Contains(msg, want) {
					t.Errorf("the message must say %q so the user can find the problem:\n%s", want, msg)
				}
			}
		})
	}
}

// The spec location a violation is about travels with it, so reports can point at the header, the
// content type or the status.
func TestResponseChecks_SubjectNamesWhatIsWrong(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(responseChecksSpec), "yaml")
	if v := check(t, spec, "GET", "/a", 200, map[string]string{"Content-Type": "application/json"}, `{"id":1}`); v[0].Subject != "X-Total-Count" {
		t.Errorf("missing header subject = %q", v[0].Subject)
	}
	if v := check(t, spec, "GET", "/a", 500, nil, ``); v[0].Subject != "500" {
		t.Errorf("status subject = %q", v[0].Subject)
	}
	if v := check(t, spec, "GET", "/a", 200, map[string]string{"Content-Type": "text/plain", "X-Total-Count": "1"}, `x`); v[0].Subject != "text/plain" {
		t.Errorf("content type subject = %q", v[0].Subject)
	}
}

const requestChecksSpec = `
openapi: "3.0.3"
info: {title: s, version: "1"}
paths:
  /items/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: integer}}
      - {name: limit, in: query, required: true, schema: {type: integer}}
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, required: [price], properties: {price: {type: number}}}
      responses: {"201": {description: ok}}
`

// A request that does not fit the spec says which parameter or which part of the body, not only
// that "a value is invalid".
func TestRequestChecks_NameTheParameterOrTheBody(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(requestChecksSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	do := func(url, limit, body string) []openapi.ValidationError {
		return openapi.ValidateRequest(spec, &domain.Request{
			Method: "POST", URL: url, QueryParams: []domain.QueryParam{{Name: "limit", Value: limit}},
			Body: &domain.RequestBody{Type: domain.BodyJSON, Content: body},
		})
	}
	if v := do("http://x/items/1", "5", `{"price":1.5}`); len(v) != 0 {
		t.Fatalf("a valid request has no violation: %v", v)
	}

	for _, c := range []struct {
		name         string
		got          []openapi.ValidationError
		kind, in, on string
		mention      []string
	}{
		{"bad path value", do("http://x/items/abc", "5", `{"price":1}`), openapi.KindRequestParameter, "path", "id", []string{"POST /items/{id}", `path parameter "id"`, "abc"}},
		{"bad query value", do("http://x/items/1", "many", `{"price":1}`), openapi.KindRequestParameter, "query", "limit", []string{"POST /items/{id}", `query parameter "limit"`, "many"}},
		{"body field of the wrong type", do("http://x/items/1", "5", `{"price":"abc"}`), openapi.KindRequestBody, "", "", []string{"POST /items/{id}", "request body", "/price"}},
		{"body that is not JSON", do("http://x/items/1", "5", `{"price":abc}`), openapi.KindRequestBody, "", "", []string{"POST /items/{id}", "request body"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if len(c.got) != 1 || c.got[0].Kind != c.kind || c.got[0].In != c.in || c.got[0].Subject != c.on {
				t.Fatalf("got %+v, want one %s (in=%q, subject=%q)", c.got, c.kind, c.in, c.on)
			}
			for _, want := range c.mention {
				if !strings.Contains(c.got[0].Error(), want) {
					t.Errorf("the message must say %q:\n%s", want, c.got[0].Error())
				}
			}
		})
	}
}
