package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const validationSpec = `
openapi: "3.0.3"
info: {title: Validation API, version: "2.1"}
security:
  - key: []
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - {name: id, in: path, required: true, schema: {type: integer}}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id: {type: integer}
        "404": {description: missing}
        default: {description: other}
  /users:
    post:
      operationId: createUser
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string}
      responses:
        "201": {description: created}
        "4XX": {description: client error}
`

func loadValidationSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	s, err := openapi.ParseBytes([]byte(validationSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func jsonResponse(status int, body string) *domain.Response {
	return &domain.Response{
		StatusCode: status,
		Headers:    []domain.Header{{Name: "Content-Type", Value: "application/json; charset=utf-8"}},
		Body:       []byte(body),
	}
}

func TestValidateResponseByPath(t *testing.T) {
	s := loadValidationSpec(t)
	tests := []struct {
		name    string
		method  string
		path    string
		status  int
		body    string
		wantErr bool
	}{
		{"conforming", "GET", "/users/5", 200, `{"id":5}`, false},
		{"schema violation", "GET", "/users/5", 200, `{"id":"five"}`, true},
		{"missing required", "GET", "/users/5", 200, `{}`, true},
		{"status covered by default response", "GET", "/users/5", 418, `{}`, false},
		{"declared status without content", "GET", "/users/5", 404, ``, false},
		{"path not in spec is skipped", "GET", "/nope", 200, `{}`, false},
		{"method not in spec is skipped", "DELETE", "/users/5", 204, ``, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := openapi.ValidateResponseByPath(s, tc.method, tc.path, tc.status, jsonResponse(tc.status, tc.body))
			if (len(errs) > 0) != tc.wantErr {
				t.Fatalf("wantErr=%v got %v", tc.wantErr, errs)
			}
			for _, e := range errs {
				if e.Error() == "" {
					t.Error("validation error should have a message")
				}
			}
		})
	}
}

func TestValidateResponseByPath_DefaultContentType(t *testing.T) {
	s := loadValidationSpec(t)
	resp := &domain.Response{StatusCode: 200, Body: []byte(`{"id":1}`)}
	if errs := openapi.ValidateResponseByPath(s, "GET", "/users/1", 200, resp); len(errs) != 0 {
		t.Errorf("missing Content-Type should default to JSON: %v", errs)
	}
}

func TestValidateRequest(t *testing.T) {
	s := loadValidationSpec(t)
	post := func(bt domain.BodyType, body string) *domain.Request {
		return &domain.Request{
			Method: domain.MethodPOST, URL: "http://localhost/users?x=1",
			Headers: []domain.Header{{Name: "X-Trace", Value: "1"}, {Name: "", Value: "ignored"}},
			Body:    &domain.RequestBody{Type: bt, Content: body},
		}
	}

	if errs := openapi.ValidateRequest(s, post(domain.BodyJSON, `{"name":"a"}`)); len(errs) != 0 {
		t.Errorf("valid body rejected: %v", errs)
	}
	if errs := openapi.ValidateRequest(s, post(domain.BodyJSON, `{}`)); len(errs) == 0 {
		t.Error("missing required field should be reported")
	}
	_ = openapi.ValidateRequest(s, post(domain.BodyRaw, `{"name":"a"}`)) // exercises the text/plain mapping

	getBadPath := &domain.Request{
		Method: domain.MethodGET, URL: "http://localhost/users/abc",
		QueryParams: []domain.QueryParam{{Name: "q", Value: "1"}, {Name: "", Value: "skip"}},
	}
	if errs := openapi.ValidateRequest(s, getBadPath); len(errs) == 0 {
		t.Error("non-integer path param should be reported")
	}
	if errs := openapi.ValidateRequest(s, &domain.Request{Method: domain.MethodGET, URL: "http://localhost/unknown"}); errs != nil {
		t.Errorf("unknown path should be skipped: %v", errs)
	}
	if errs := openapi.ValidateRequest(s, &domain.Request{Method: domain.MethodGET, URL: "http://localhost/users/1"}); len(errs) != 0 {
		t.Errorf("security should not be enforced: %v", errs)
	}
}

func TestValidateRequest_OtherBodyTypesDoNotPanic(t *testing.T) {
	s := loadValidationSpec(t)
	for _, bt := range []domain.BodyType{domain.BodyFormURLEncoded, domain.BodyMultipart, domain.BodyRaw} {
		_ = openapi.ValidateRequest(s, &domain.Request{
			Method: domain.MethodPOST, URL: "http://localhost/users",
			Body: &domain.RequestBody{Type: bt, Content: "name=a"},
		})
	}
}

func TestExtractSpecMeta(t *testing.T) {
	meta := openapi.ExtractSpecMeta(loadValidationSpec(t))

	if got := meta.DeclaredStatusCodes["GET /users/{id}"]; len(got) != 2 || got[0] != 200 || got[1] != 404 {
		t.Errorf("default must be ignored and codes sorted: %v", got)
	}
	if got := meta.DeclaredStatusCodes["POST /users"]; len(got) != 1 || got[0] != 201 {
		t.Errorf("non-numeric codes such as 4XX must be ignored: %v", got)
	}
	if !meta.SecuredOps["GET /users/{id}"] {
		t.Error("global security should apply to getUser")
	}
	if meta.SecuredOps["POST /users"] {
		t.Error("security: [] opts createUser out")
	}
}

func TestAllOperationsAndInfo(t *testing.T) {
	s := loadValidationSpec(t)
	ops := s.AllOperations()
	if len(ops) != 2 || ops[0].String() != "POST /users" || ops[1].String() != "GET /users/{id}" {
		t.Errorf("unexpected operations: %v", ops)
	}
	title, version := s.Info()
	if title != "Validation API" || version != "2.1" {
		t.Errorf("info: %q %q", title, version)
	}
	if !strings.Contains(ops[0].String(), " ") {
		t.Error("String should be METHOD path")
	}
}
