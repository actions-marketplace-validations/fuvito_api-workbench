package reporting_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// What a server sends and no environment knows: no report may carry any of it. The first looks like
// an issued token; the second is a short password, recognisable only by the name of its field; the
// third is personal data in a field that is not at fault.
const (
	leakToken    = "tk_9fK2mQx7LpZr4TnB8vWcAa31"
	leakPassword = "hunter2pw"
	leakPerson   = "pii.person@example.net"
)

var leaks = []string{leakToken, leakPassword, leakPerson}

type leakCase struct {
	name    string
	schema  string // JSON schema of the declared 200 response
	status  int
	ctype   string
	body    string
	headers map[string]string
	hdrSpec string
}

const goodBody = `{"token":"` + leakToken + `","password":"` + leakPassword + `","email":"` + leakPerson + `","count":1}`

func leakCases() []leakCase {
	obj := func(extra string) string { return `{"type":"object",` + extra + `}` }
	return []leakCase{
		{name: "a required field is missing", schema: obj(`"required":["expiresIn"]`), body: goodBody},
		{name: "the token has the wrong type", schema: obj(`"properties":{"token":{"type":"integer"}}`), body: goodBody},
		{name: "the password has the wrong type", schema: obj(`"properties":{"password":{"type":"integer"}}`), body: goodBody},
		{name: "another field is wrong", schema: obj(`"properties":{"count":{"type":"string"}}`), body: goodBody},
		{name: "an object where text is declared", schema: obj(`"properties":{"data":{"type":"string"}}`), body: `{"data":` + goodBody + `}`},
		{name: "a list where text is declared", schema: obj(`"properties":{"data":{"type":"string"}}`), body: `{"data":[` + goodBody + `]}`},
		{name: "password breaks an enum", schema: obj(`"properties":{"password":{"type":"string","enum":["a"]}}`), body: goodBody},
		{name: "password breaks a pattern", schema: obj(`"properties":{"password":{"type":"string","pattern":"^[0-9]+$"}}`), body: goodBody},
		{name: "password breaks a length", schema: obj(`"properties":{"password":{"type":"string","maxLength":2}}`), body: goodBody},
		{name: "token breaks a format", schema: obj(`"properties":{"token":{"type":"string","format":"uuid"}}`), body: goodBody},
		{name: "token breaks an email format", schema: obj(`"properties":{"token":{"type":"string","format":"email"}}`), body: goodBody},
		{name: "token breaks a date format", schema: obj(`"properties":{"token":{"type":"string","format":"date-time"}}`), body: goodBody},
		{name: "a secret number is out of range", schema: obj(`"properties":{"pin":{"type":"integer","maximum":1}}`), body: `{"pin":9988776655}`},
		{name: "an undeclared property", schema: obj(`"additionalProperties":false,"properties":{"count":{"type":"integer"}}`), body: goodBody},
		{name: "a writeOnly property comes back", schema: obj(`"properties":{"password":{"type":"string","writeOnly":true}}`), body: goodBody},
		{name: "no branch of a oneOf matches", schema: `{"oneOf":[{"type":"object","required":["a"]},{"type":"object","required":["b"]}]}`, body: goodBody},
		{name: "text where an object is declared", schema: obj(`"required":["a"]`), body: `"` + leakToken + `"`},
		{name: "a list where an object is declared", schema: obj(`"required":["a"]`), body: `[` + goodBody + `]`},
		{name: "not JSON", schema: obj(`"required":["a"]`), body: `token=` + leakToken + `; password=` + leakPassword},
		{name: "text after the JSON", schema: obj(`"required":["count"]`), body: `{"count":1} ` + leakToken},
		{name: "an undeclared status with a JSON body", schema: obj(`` + `"required":["a"]`), status: 201, body: goodBody},
		{name: "an undeclared status with a text body", schema: obj(`"required":["a"]`), status: 400, ctype: "text/plain", body: "bad session_token=" + leakToken + " password: " + leakPassword},
		{name: "an undeclared status with an error message", schema: obj(`"required":["a"]`), status: 401, body: `{"error":"token ` + leakToken + ` expired","access_token":"` + leakToken + `","user":{"password":"` + leakPassword + `"}}`},
		{name: "the wrong content type", schema: obj(`"required":["a"]`), ctype: "text/plain", body: goodBody},
		{name: "a bad secret header", schema: obj(`"required":[]`), body: goodBody, hdrSpec: `"X-Session-Token":{"schema":{"type":"integer"}}`, headers: map[string]string{"X-Session-Token": leakToken}},
		{name: "a bad cookie header", schema: obj(`"required":[]`), body: goodBody, hdrSpec: `"Set-Cookie":{"schema":{"type":"integer"}}`, headers: map[string]string{"Set-Cookie": "sid=" + leakPassword}},
	}
}

func (c leakCase) spec(version string) string {
	headers := ""
	if c.hdrSpec != "" {
		headers = `,"headers":{` + c.hdrSpec + `}`
	}
	return fmt.Sprintf(`{"openapi":%q,"info":{"title":"Leak","version":"1"},"paths":{"/session":{"get":{"operationId":"getSession","responses":{"200":{"description":"ok"%s,"content":{"application/json":{"schema":%s}}}}}}}}`, version, headers, c.schema)
}

func (c leakCase) serve(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ct := c.ctype
		if ct == "" {
			ct = "application/json"
		}
		w.Header().Set("Content-Type", ct)
		for k, v := range c.headers {
			w.Header().Set(k, v)
		}
		status := c.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(c.body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func found(outputs map[string]string) []string {
	var out []string
	for name, text := range outputs {
		for _, secret := range leaks {
			if strings.Contains(text, secret) {
				out = append(out, name+": "+secret)
			}
		}
		if strings.Contains(text, "9988776655") {
			out = append(out, name+": the pin")
		}
	}
	sort.Strings(out)
	return out
}

func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestNoConformOutputCarriesWhatOnlyTheResponseHolds(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0"} {
		for _, c := range leakCases() {
			t.Run(version+"/"+c.name, func(t *testing.T) {
				spec, err := openapi.ParseBytes([]byte(c.spec(version)), "json")
				if err != nil {
					t.Fatal(err)
				}
				result := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, c.serve(t), nil, conformance.Options{})
				violations := conformance.Violations(result, nil)
				outputs := map[string]string{
					"text":     reporting.ConformTextExplained(result, nil, true),
					"markdown": string(reporting.ConformMarkdown(result, nil, nil)),
					"github":   reporting.ConformGitHub(result, nil, nil),
					"baseline": asJSON(conformance.NewBaseline(violations)),
					"diff":     conformance.Compare(violations, &conformance.Baseline{}).Text(),
				}
				for _, op := range result.Operations {
					// what the desktop page and the MCP server receive
					outputs["page findings"] = asJSON(reporting.Explain(op, result.Secrets))
					for i, v := range op.Violations {
						outputs[fmt.Sprintf("violation %d", i)] = v.Error()
					}
				}
				if b, err := reporting.ConformJSON(result, nil, nil); err == nil {
					outputs["json"] = string(b)
				}
				if b, err := reporting.ConformJUnit(result, nil); err == nil {
					outputs["junit"] = string(b)
				}
				if b, err := reporting.ConformHTML(result, nil, nil); err == nil {
					outputs["html"] = string(b)
				}
				if got := found(outputs); len(got) > 0 {
					t.Errorf("leaked: %s", strings.Join(got, "; "))
				}
			})
		}
	}
}

func TestNoRunReportCarriesWhatOnlyTheResponseHolds(t *testing.T) {
	assertions := map[string]domain.Assertion{
		"inline schema that passes":       {Type: domain.AssertionSchema, Schema: `{"type":"object"}`},
		"inline schema that fails":        {Type: domain.AssertionSchema, Schema: `{"type":"object","required":["expiresIn"]}`},
		"inline schema, token wrong type": {Type: domain.AssertionSchema, Schema: `{"type":"object","properties":{"token":{"type":"integer"}}}`},
		"schema by operation id":          {Type: domain.AssertionSchema, Schema: "getSession"},
		"token is present":                {Type: domain.AssertionJSONPath, Path: "$.token", Operator: domain.OperatorNe, Expected: ""},
		"token is something else":         {Type: domain.AssertionJSONPath, Path: "$.token", Operator: domain.OperatorEq, Expected: "x"},
		"password is something else":      {Type: domain.AssertionJSONPath, Path: "$.password", Operator: domain.OperatorEq, Expected: "x"},
		"the whole body":                  {Type: domain.AssertionJSONPath, Path: "$", Operator: domain.OperatorEq, Expected: "x"},
		"the cookie passes":               {Type: domain.AssertionHeader, Path: "Set-Cookie", Operator: domain.OperatorContains, Expected: "sid="},
		"the cookie fails":                {Type: domain.AssertionHeader, Path: "Set-Cookie", Operator: domain.OperatorEq, Expected: "x"},
	}
	c := leakCase{schema: `{"type":"object","required":["expiresIn"]}`, body: goodBody, headers: map[string]string{"Set-Cookie": "sid=" + leakPassword}}
	spec, err := openapi.ParseBytes([]byte(c.spec("3.0.3")), "json")
	if err != nil {
		t.Fatal(err)
	}
	for name, assertion := range assertions {
		t.Run(name, func(t *testing.T) {
			url := c.serve(t)
			sc := &domain.Scenario{Title: "Leak", Steps: []domain.ScenarioStep{{
				Label:      "get",
				Request:    &domain.Request{Method: "GET", URL: url + "/session"},
				Assertions: []domain.Assertion{assertion},
			}}}
			result := scenario.NewWithSpec(executor.New(executor.Options{}), spec).Run(context.Background(), sc, nil)
			outputs := map[string]string{
				"text":     reporting.Text(result),
				"markdown": string(reporting.Markdown(result, nil)),
				"github":   reporting.RunGitHub([]reporting.GitHubRun{{Result: result}}),
			}
			// what the history database keeps and the desktop shows beside the assertion
			for _, step := range result.Steps {
				outputs["assertion results"] = asJSON(step.Assertions)
				outputs["spec check"] = asJSON(step.Spec)
				outputs["step error"] = step.Error
			}
			if b, err := reporting.JSON(result); err == nil {
				outputs["json"] = string(b)
			}
			if b, err := reporting.JUnit(result); err == nil {
				outputs["junit"] = string(b)
			}
			if b, err := reporting.HTML(result, nil); err == nil {
				outputs["html"] = string(b)
			}
			if got := found(outputs); len(got) > 0 {
				t.Errorf("leaked: %s", strings.Join(got, "; "))
			}
		})
	}
}
