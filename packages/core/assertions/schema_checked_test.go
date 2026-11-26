package assertions_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/assertions"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// A schema assertion must not pass without having looked at the body: it checks it, or it fails and
// says why it could not.

const checkedSpec = `
openapi: "3.0.3"
info: {title: s, version: "1"}
paths:
  /thing:
    get:
      operationId: getThing
      responses:
        "200":
          description: ok
          content: {application/json: {schema: {type: object, required: [id], properties: {id: {type: integer}}}}}
        "204": {description: nothing}
        "202":
          description: a report
          content: {text/csv: {schema: {type: string}}}
        "203":
          description: no schema
          content: {application/json: {}}
        "4XX":
          description: any client error
          content: {application/problem+json: {schema: {type: object, required: [title], properties: {title: {type: string}}}}}
  /fallback:
    get:
      operationId: getFallback
      responses:
        "200":
          description: ok
          content: {application/json: {schema: {type: object, required: [id]}}}
        default:
          description: an error
          content: {application/json: {schema: {type: object, required: [error]}}}
`

func TestSchemaAssertionByOperationNeverPassesUnchecked(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(checkedSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		op     string
		status int
		ctype  string
		body   string
		pass   bool
		says   string
	}{
		"a fitting 200":                     {"getThing", 200, "application/json", `{"id":1}`, true, ""},
		"a bad 200":                         {"getThing", 200, "application/json", `{"id":"x"}`, false, "/id"},
		"an undeclared status":              {"getThing", 500, "application/json", `{}`, false, "declares no response for status 500"},
		"a bad body under a range":          {"getThing", 404, "application/problem+json", `{"title":7}`, false, "/title"},
		"a fitting body under a range":      {"getThing", 404, "application/problem+json", `{"title":"gone"}`, true, ""},
		"another JSON type than declared":   {"getThing", 404, "application/json", `{"title":7}`, false, "/title"},
		"and a fitting one":                 {"getThing", 404, "application/json", `{"title":"gone"}`, true, ""},
		"a declared status with no content": {"getThing", 204, "application/json", `{"a":1}`, false, "declares no schema for status 204"},
		"a declared type with no schema":    {"getThing", 203, "application/json", `{"a":1}`, false, "declares no schema for status 203"},
		"a type that cannot be read":        {"getThing", 202, "text/csv", "a,b\n1,2\n", false, "cannot be checked"},
		"JSON where only CSV is declared":   {"getThing", 202, "application/json", `"a"`, false, "cannot be checked"},
		"not JSON at all":                   {"getThing", 200, "text/html", `<html>`, false, "not valid JSON"},
		"default is used":                   {"getFallback", 503, "application/json", `{"error":"x"}`, true, ""},
		"default is checked":                {"getFallback", 503, "application/json", `{}`, false, "error"},
		"an unknown operation":              {"nothing", 200, "application/json", `{}`, false, "not found in spec"},
	} {
		t.Run(name, func(t *testing.T) {
			r := assertions.NewWithSpec(spec).Evaluate(domain.Assertion{Type: domain.AssertionSchema, Schema: c.op},
				&domain.Response{StatusCode: c.status, Headers: []domain.Header{{Name: "Content-Type", Value: c.ctype}}, Body: []byte(c.body)})
			if r.Passed != c.pass || !strings.Contains(r.Message, c.says) {
				t.Errorf("passed %v (want %v): %q (want it to say %q)", r.Passed, c.pass, r.Message, c.says)
			}
		})
	}
}

// A schema written in place says what the body must look like, whatever the status is.
func TestSchemaWrittenInPlaceIsCheckedForEveryStatus(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionSchema, Schema: `{"type":"object","required":["title"]}`}
	for _, status := range []int{200, 201, 404, 500} {
		good := assertions.New().Evaluate(a, &domain.Response{StatusCode: status, Body: []byte(`{"title":"x"}`)})
		if !good.Passed {
			t.Errorf("%d, a fitting body: %s", status, good.Message)
		}
		for _, body := range []string{`{}`, `[1]`, `not json`} {
			if bad := assertions.New().Evaluate(a, &domain.Response{StatusCode: status, Body: []byte(body)}); bad.Passed {
				t.Errorf("%d, %s passed", status, body)
			}
		}
	}
}
