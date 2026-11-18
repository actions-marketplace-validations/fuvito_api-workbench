package openapi

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

const auditSpec = `openapi: 3.0.3
info: {title: t, version: "1"}
servers:
  - url: https://api.example.com/v1
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties: {id: {type: integer}}
        "404": {description: nf}
`

func resp(code int, body, ct string) *domain.Response {
	r := &domain.Response{StatusCode: code, Body: []byte(body)}
	if ct != "" {
		r.Headers = []domain.Header{{Name: "Content-Type", Value: ct}}
	}
	return r
}

func TestAuditValidator(t *testing.T) {
	s, err := ParseBytes([]byte(auditSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	bad := resp(200, `{"id":"not-int"}`, "application/json")
	for _, p := range []string{"/users/5", "/v1/users/5"} {
		v := ValidateResponseByPath(s, "GET", p, 200, bad)
		t.Logf("AUDIT bad-body path=%-14s violations=%d %v", p, len(v), v)
	}
	t.Logf("AUDIT undeclared 500 (ByPath)      : %v", ValidateResponseByPath(s, "GET", "/v1/users/5", 500, resp(500, `oops`, "text/plain")))
	t.Logf("AUDIT undeclared 500 (ByOperation) : %v", ValidateResponse(s, "getUser", resp(500, `oops`, "text/plain")))
	t.Logf("AUDIT empty body on 200 (ByPath)   : %v", ValidateResponseByPath(s, "GET", "/v1/users/5", 200, resp(200, ``, "")))
	t.Logf("AUDIT empty body on 200 (ByOp)     : %v", ValidateResponse(s, "getUser", resp(200, ``, "")))
	t.Logf("AUDIT html body on 200 (ByPath)    : %v", ValidateResponseByPath(s, "GET", "/v1/users/5", 200, resp(200, `<html>`, "text/html")))
	t.Logf("AUDIT html body on 200 (ByOp)      : %v", ValidateResponse(s, "getUser", resp(200, `<html>`, "text/html")))

	cov := s.ScenarioCoverage([]ExecutedRequest{
		{Method: "GET", URL: "http://localhost:8080/v1/users/5", StatusCode: 200, Passed: true},
		{Method: "GET", URL: "http://localhost:8080/users/5", StatusCode: 200, Passed: true},
	})
	t.Logf("AUDIT coverage covered=%d/%d exercised=%v", cov.Covered, cov.Total, cov.Exercised)
}

func TestAuditSpecLoading(t *testing.T) {
	badExample := `openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, properties: {n: {type: integer}}}
              example: {n: "not an int"}
`
	_, err := ParseBytes([]byte(badExample), "yaml")
	t.Logf("AUDIT spec with a wrong example loads? err=%v", err)

	extRef := `openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {$ref: "./schemas/user.yaml"}
`
	_, err = ParseBytes([]byte(extRef), "yaml")
	t.Logf("AUDIT spec with external $ref loads? err=%v", err)

	noIDs := `openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /a: {get: {responses: {"200": {description: ok}}}}
  /a/: {get: {responses: {"200": {description: ok}}}}
`
	_, err = ParseBytes([]byte(noIDs), "yaml")
	t.Logf("AUDIT trailing-slash twin paths: err=%v", err)
}
