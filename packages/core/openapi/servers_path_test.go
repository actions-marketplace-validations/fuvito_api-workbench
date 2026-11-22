package openapi_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const serversSpec = `openapi: 3.0.3
info: {title: t, version: "1"}
servers:
  - url: http://localhost:8080/base
paths:
  /cases/{id}:
    get:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [name]
                properties:
                  name: {type: string}
`

// A spec written the standard way (servers with a base path) must still be matched by the path a
// real request carries, with or without the base path; before, nothing matched and every response
// passed unchecked.
func TestServersBasePathIsHonoured(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(serversSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	bad := &domain.Response{StatusCode: 200, Headers: []domain.Header{{Name: "Content-Type", Value: "application/json"}}, Body: []byte(`{"name": 1}`)}
	for _, p := range []string{"/base/cases/x", "/cases/x"} {
		if !spec.HasOperation("GET", p) {
			t.Errorf("HasOperation(%q) = false", p)
		}
		if len(openapi.ValidateResponseByPath(spec, "GET", p, 200, bad)) == 0 {
			t.Errorf("a wrong body at %q was not caught", p)
		}
	}
	res := spec.ScenarioCoverage([]openapi.ExecutedRequest{{Method: "GET", URL: "http://localhost:8080/base/cases/x", StatusCode: 200}})
	if res == nil || res.Covered != 1 {
		t.Errorf("coverage = %+v, want 1 covered", res)
	}
}
