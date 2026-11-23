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
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

const shapesSpec = `openapi: "3.0.3"
info: {title: shapes, version: "1"}
servers: [{url: "%s/base"}]
paths:
  /obj:
    get:
      operationId: obj
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [a], properties: {a: {type: string}}}
  /one:
    get:
      operationId: one
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - {type: object, required: [a], properties: {a: {type: string}}}
                  - {type: object, required: [b], properties: {b: {type: string}}}
  /price:
    post:
      operationId: price
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, required: [amount], properties: {amount: {type: number}}}
      responses: {"201": {description: ok}}
`

// shapesServer answers each path with a different kind of wrong body.
func shapesServer(body func(path string) string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(201)
			return
		}
		_, _ = w.Write([]byte(body(strings.TrimPrefix(r.URL.Path, "/base"))))
	}))
}

func runShapes(t *testing.T, srv *httptest.Server, env *domain.Environment) *conformance.ConformResult {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(fmt.Sprintf(shapesSpec, srv.URL)), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, "", env, conformance.Options{})
}

func findingOf(t *testing.T, res *conformance.ConformResult, path string) conformance.Finding {
	t.Helper()
	for _, op := range res.Operations {
		if op.Path == path || op.SpecPath == path {
			fs := conformance.Diagnose(op)
			if len(fs) == 0 {
				t.Fatalf("%s: no finding (%+v)", path, op)
			}
			return fs[0]
		}
	}
	t.Fatalf("no operation %s", path)
	return conformance.Finding{}
}

// F-011 and F-012: body problems have their own kinds and check IDs, and a type mismatch at the
// root is a wrong type like one at a field.
func TestFindings_BodyShapesHaveTheirOwnKindsAndCheckIDs(t *testing.T) {
	cases := []struct {
		name, objBody, oneBody string
		path                   string
		kind, id               string
	}{
		{"empty body", ``, `{"a":"x"}`, "/obj", conformance.FindingEmptyBody, "CS-RESP-008"},
		{"not JSON", `{nope`, `{"a":"x"}`, "/obj", conformance.FindingInvalidBody, "CS-RESP-008"},
		{"array for object", `[1]`, `{"a":"x"}`, "/obj", conformance.FindingWrongType, "CS-RESP-006"},
		{"no oneOf branch", `{"a":"x"}`, `{"c":1}`, "/one", conformance.FindingNoBranch, "CS-RESP-009"},
		{"several oneOf branches", `{"a":"x"}`, `{"a":"x","b":"y"}`, "/one", conformance.FindingNoBranch, "CS-RESP-009"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := shapesServer(func(p string) string {
				if p == "/obj" {
					return c.objBody
				}
				return c.oneBody
			})
			defer srv.Close()
			f := findingOf(t, runShapes(t, srv, domain.NewEnvironment("e")), c.path)
			if f.Kind != c.kind || f.CheckID != c.id {
				t.Errorf("kind=%s id=%s, want %s %s\n%+v", f.Kind, f.CheckID, c.kind, c.id, f)
			}
			if c.kind == conformance.FindingNoBranch && strings.Contains(f.Summary, "missing") {
				t.Errorf("a oneOf mismatch must not be described as a missing field: %s", f.Summary)
			}
		})
	}
}

// F-017: an environment value that is not valid JSON where a number is declared is explained at
// field level, naming the variable, the value and the declared type.
func TestFindings_WrongTypedBodyValueNamesFieldAndType(t *testing.T) {
	srv := shapesServer(func(string) string { return `{"a":"x"}` })
	defer srv.Close()
	env := domain.NewEnvironment("e")
	env.Variables = []domain.EnvironmentVariable{{Name: "amount", Value: "cheap"}}
	res := runShapes(t, srv, env)
	for _, op := range res.Operations {
		if op.SpecPath != "/price" {
			continue
		}
		if op.Error != "" || len(op.RequestViolations) != 1 {
			t.Fatalf("%+v", op)
		}
		f := conformance.Diagnose(op)[0]
		if f.CheckID != "CS-REQ-002" || f.Where != "amount" || f.Expected != "a number" || f.Got != "cheap" ||
			!strings.Contains(f.Summary, "{{amount}}") || !strings.Contains(f.Fix, "amount=9.99") {
			t.Errorf("%+v", f)
		}
		return
	}
	t.Fatal("no /price operation")
}

// F-007: the base URL is the spec's server when none is given.
func TestRun_ShowsTheSpecServerAsBaseURL(t *testing.T) {
	srv := shapesServer(func(string) string { return `{"a":"x"}` })
	defer srv.Close()
	res := runShapes(t, srv, domain.NewEnvironment("e"))
	if res.BaseURL != srv.URL+"/base" {
		t.Errorf("BaseURL = %q, want %q", res.BaseURL, srv.URL+"/base")
	}
	if !strings.Contains(reporting.ConformText(res, nil), "Base URL: "+srv.URL+"/base") {
		t.Error("the text report must show the base URL")
	}
}

// F-012: the JSON report carries the check ID of every finding.
func TestJSON_FindingsCarryCheckIDs(t *testing.T) {
	srv := shapesServer(func(p string) string { return `{"c":1}` })
	defer srv.Close()
	res := runShapes(t, srv, domain.NewEnvironment("e"))
	js, err := reporting.ConformJSON(res, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `"checkId": "CS-RESP-005"`) {
		t.Errorf("no check ID in:\n%s", js)
	}
}
