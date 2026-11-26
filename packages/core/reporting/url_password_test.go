package reporting_test

import (
	"context"
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
	"github.com/fuvito/api-workbench/packages/core/redact"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// Too short and plain to look like a token, under names that say nothing: only the way it is
// written gives it away.
const urlPw = "s3cretPw1"

const dsn = "postgres://app:" + urlPw + "@db.internal:5432/orders"

var passwordForms = map[string]string{
	"address":           dsn,
	"address in prose":  "cannot connect to amqp://app:" + urlPw + "@mq:5672/ (timeout)",
	"connection string": "Server=db;User Id=app;Password=" + urlPw + ";",
	"ODBC":              "Driver={SQL};Uid=app;Pwd=" + urlPw + ";",
	"query string":      "https://api.example.com/v1/items?user=app&password=" + urlPw + "&page=2",
	"assignment":        "login failed for password=" + urlPw + " retry later",
}

func conformReports(t *testing.T, r *conformance.ConformResult) map[string]string {
	t.Helper()
	violations := conformance.Violations(r, nil)
	baseline := conformance.NewBaseline(violations)
	out := map[string]string{
		"text":     reporting.ConformTextExplained(r, nil, true),
		"markdown": string(reporting.ConformMarkdown(r, nil, nil)),
		"github":   reporting.ConformGitHub(r, nil, nil),
		"diff":     conformance.Compare(violations, &conformance.Baseline{}).Text(),
	}
	// the file as it is written, in both formats
	for _, name := range []string{"b.yaml", "b.json"} {
		b, err := baseline.Marshal(name)
		if err != nil {
			t.Fatal(err)
		}
		out["baseline "+name] = string(b)
	}
	for _, op := range r.Operations {
		out["page findings"] += asJSON(reporting.Explain(op, r.Secrets))
		for _, v := range op.Violations {
			out["violations"] += v.Error()
		}
	}
	for name, render := range map[string]func() ([]byte, error){
		"json":  func() ([]byte, error) { return reporting.ConformJSON(r, nil, nil) },
		"junit": func() ([]byte, error) { return reporting.ConformJUnit(r, nil) },
		"html":  func() ([]byte, error) { return reporting.ConformHTML(r, nil, nil) },
	} {
		b, err := render()
		if err != nil {
			t.Fatal(err)
		}
		out[name] = string(b)
	}
	return out
}

func runReports(t *testing.T, r *scenario.Result) map[string]string {
	t.Helper()
	out := map[string]string{
		"text": reporting.Text(r), "markdown": string(reporting.Markdown(r, nil)),
		"github": reporting.RunGitHub([]reporting.GitHubRun{{Result: r}}),
	}
	for _, step := range r.Steps {
		for _, a := range step.Assertions {
			out["assertion messages and values"] += a.Message + "\n" + a.Actual + "\n"
		}
	}
	for name, render := range map[string]func() ([]byte, error){
		"json":  func() ([]byte, error) { return reporting.JSON(r) },
		"junit": func() ([]byte, error) { return reporting.JUnit(r) },
		"html":  func() ([]byte, error) { return reporting.HTML(r, nil) },
	} {
		b, err := render()
		if err != nil {
			t.Fatal(err)
		}
		out[name] = string(b)
	}
	return out
}

func noPassword(t *testing.T, outputs map[string]string) {
	t.Helper()
	var in []string
	for name, text := range outputs {
		if strings.Contains(text, urlPw) {
			in = append(in, name)
		}
	}
	sort.Strings(in)
	if len(in) > 0 {
		t.Errorf("the password is in: %s\n%s", strings.Join(in, ", "), outputs[in[0]])
	}
}

func passwordCases(value string) []leakCase {
	obj := func(extra string) string { return `{"type":"object",` + extra + `}` }
	body := fmt.Sprintf(`{"message":%q}`, value)
	prop := func(extra string) string { return obj(`"properties":{"message":{` + extra + `}}`) }
	return []leakCase{
		{name: "wrong type", schema: prop(`"type":"integer"`), body: body},
		{name: "enum", schema: prop(`"type":"string","enum":["a"]`), body: body},
		{name: "pattern", schema: prop(`"type":"string","pattern":"^[0-9]+$"`), body: body},
		{name: "format", schema: prop(`"type":"string","format":"uuid"`), body: body},
		{name: "error excerpt, JSON", schema: obj(`"required":["a"]`), status: 500, body: body},
		{name: "error excerpt, text", schema: obj(`"required":["a"]`), status: 500, ctype: "text/plain", body: value},
		{name: "header of the wrong type", schema: obj(`"required":[]`), body: `{}`, hdrSpec: `"X-Upstream":{"schema":{"type":"integer"}}`, headers: map[string]string{"X-Upstream": value}},
		{name: "header breaks a pattern", schema: obj(`"required":[]`), body: `{}`, hdrSpec: `"Location":{"schema":{"type":"string","pattern":"^[0-9]+$"}}`, headers: map[string]string{"Location": value}},
		{name: "content type", schema: obj(`"required":["a"]`), ctype: `text/plain; note="` + value + `"`, body: `{}`},
	}
}

func TestNoConformOutputCarriesAWrittenOutPassword(t *testing.T) {
	for form, value := range passwordForms {
		for _, version := range []string{"3.0.3", "3.1.0"} {
			for _, c := range passwordCases(value) {
				t.Run(form+"/"+version+"/"+c.name, func(t *testing.T) {
					spec, err := openapi.ParseBytes([]byte(c.spec(version)), "json")
					if err != nil {
						t.Fatal(err)
					}
					r := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, c.serve(t), nil, conformance.Options{})
					noPassword(t, conformReports(t, r))
				})
			}
		}
	}
}

func TestAMaskedMessageStillSaysWhichServer(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0"} {
		for _, c := range passwordCases(dsn) {
			if c.name != "wrong type" && c.name != "error excerpt, text" && c.name != "header of the wrong type" {
				continue
			}
			spec, err := openapi.ParseBytes([]byte(c.spec(version)), "json")
			if err != nil {
				t.Fatal(err)
			}
			r := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, c.serve(t), nil, conformance.Options{})
			const want = "postgres://app:***@db.internal:5432/orders"
			if text := reporting.ConformTextExplained(r, nil, true); !strings.Contains(text, want) {
				t.Errorf("%s %s: %q is missing\n%s", version, c.name, want, text)
			}
		}
	}
}

func TestAPasswordFromTheEnvironmentOrTheAddressIsMasked(t *testing.T) {
	c := leakCase{schema: `{"type":"object","required":["a"]}`, body: `{}`}
	live := strings.Replace(c.serve(t), "http://", "http://app:"+urlPw+"@", 1)
	spec, err := openapi.ParseBytes([]byte(c.spec("3.0.3")), "json")
	if err != nil {
		t.Fatal(err)
	}
	for name, base := range map[string]string{
		"server answers": live, "port closed": "http://app:" + urlPw + "@127.0.0.1:1", "query": "http://127.0.0.1:1/x?password=" + urlPw,
	} {
		t.Run("base address/"+name, func(t *testing.T) {
			noPassword(t, conformReports(t, conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, base, nil, conformance.Options{})))
			sc := &domain.Scenario{Title: "P", Steps: []domain.ScenarioStep{{Label: "get", Request: &domain.Request{Method: "GET", URL: base},
				Assertions: []domain.Assertion{{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 201}}}}}
			noPassword(t, runReports(t, scenario.New(executor.New(executor.Options{})).Run(context.Background(), sc, nil)))
		})
	}
	t.Run("a value that does not fit the request", func(t *testing.T) {
		body, err := openapi.ParseBytes([]byte(`{"openapi":"3.0.3","info":{"title":"L","version":"1"},"paths":{"/s":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["database"],"properties":{"database":{"type":"integer"}}}}}},"responses":{"200":{"description":"ok"}}}}}}`), "json")
		if err != nil {
			t.Fatal(err)
		}
		env := &domain.Environment{Name: "e", Variables: []domain.EnvironmentVariable{{Name: "database", Value: dsn}}}
		noPassword(t, conformReports(t, conformance.New(executor.New(executor.Options{})).Run(context.Background(), body, c.serve(t), env, conformance.Options{})))
	})
	t.Run("a scenario variable", func(t *testing.T) {
		sc := &domain.Scenario{Title: "P", Variables: []domain.EnvironmentVariable{{Name: "database", Value: dsn}}, Steps: []domain.ScenarioStep{{Label: "get",
			Request:    &domain.Request{Method: "GET", URL: c.serve(t)},
			Assertions: []domain.Assertion{{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 201}}}}}
		noPassword(t, runReports(t, scenario.New(executor.New(executor.Options{})).Run(context.Background(), sc, &domain.Environment{Name: "e"})))
	})
}

// Only what is written changes: checks compare, and later steps send, the real values.
func TestAssertionsAndExtractionsUseTheRealPassword(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/use" {
			received = r.Header.Get("X-Database")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Upstream", dsn)
		_, _ = fmt.Fprintf(w, `{"database":%q}`, dsn)
	}))
	t.Cleanup(srv.Close)
	jp := func(op domain.Operator, expected string) domain.Assertion {
		return domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.database", Operator: op, Expected: expected}
	}
	sc := &domain.Scenario{Title: "P", Steps: []domain.ScenarioStep{
		{Label: "get", FailPolicy: domain.FailPolicyContinue, Request: &domain.Request{Method: "GET", URL: srv.URL + "/get"},
			Assertions: []domain.Assertion{
				jp(domain.OperatorEq, dsn),
				// a pattern, so that the expected value does not itself spell the password out
				jp(domain.OperatorMatches, ":s3cr.tPw1@"),
				{Type: domain.AssertionHeader, Path: "X-Upstream", Operator: domain.OperatorEq, Expected: dsn},
				jp(domain.OperatorEq, "postgres://app:***@db.internal:5432/orders"), // must fail: the mask is not the value
				jp(domain.OperatorContains, "nowhere"),
			},
			Extractions: []domain.Extraction{{Name: "database", Source: "jsonpath", Path: "$.database"}}},
		{Label: "use", Request: &domain.Request{Method: "GET", URL: srv.URL + "/use", Headers: []domain.Header{{Name: "X-Database", Value: "{{database}}"}}}},
	}}
	r := scenario.New(executor.New(executor.Options{})).Run(context.Background(), sc, nil)
	var passed []bool
	for _, a := range r.Steps[0].Assertions {
		passed = append(passed, a.Passed)
	}
	if want := []bool{true, true, true, false, false}; fmt.Sprint(passed) != fmt.Sprint(want) {
		t.Errorf("assertions %v, want %v", passed, want)
	}
	if r.Steps[0].Extracted["database"] != dsn || received != dsn {
		t.Errorf("extracted %q, the next step sent %q", r.Steps[0].Extracted["database"], received)
	}
	noPassword(t, runReports(t, r))
	// --show-extracted is asking to see the value
	r.RevealExtracted = true
	if !strings.Contains(reporting.Text(r), `database = "`+dsn+`"`) {
		t.Errorf("--show-extracted shows the extracted value as it is:\n%s", reporting.Text(r))
	}
}

// Baselines written by 0.13.2 can hold a password: an OpenAPI 3.1 violation this package does not
// recognise is stored with the validator's sentence, and that sentence quotes the value. These two
// files were written by the released 0.13.2; they must still match, and not print it again.
func TestBaselineWrittenBy0132WithAPasswordStillMatches(t *testing.T) {
	for form, value := range map[string]string{"address": dsn, "connection string": passwordForms["connection string"]} {
		t.Run(form, func(t *testing.T) {
			old := fmt.Sprintf(`schemaVersion: "2"
violations:
    - operation: GET /session
      check: other
      rule: 'validation failed due to: error at "/message": at ''/message'': ''%[1]s'' does not match pattern ''^[0-9]+$'''
      summary: 'validation failed due to: error at "/message": at ''/message'': ''%[1]s'' does not match pattern ''^[0-9]+$'''
`, value)
			b, err := conformance.ParseBaseline([]byte(old), "old.yaml")
			if err != nil {
				t.Fatal(err)
			}
			var c leakCase
			for _, x := range passwordCases(value) {
				if x.name == "pattern" {
					c = x
				}
			}
			spec, err := openapi.ParseBytes([]byte(c.spec("3.1.0")), "json")
			if err != nil {
				t.Fatal(err)
			}
			r := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, c.serve(t), nil, conformance.Options{})
			d := conformance.Compare(conformance.Violations(r, nil), b)
			if len(d.Known) != 1 || len(d.New) != 0 || len(d.Fixed) != 0 {
				t.Errorf("known %d, new %d, fixed %d\n%s", len(d.Known), len(d.New), len(d.Fixed), d.Text())
			}
			// when it is fixed, the old entry is listed, without the password
			noPassword(t, map[string]string{"diff of a fixed violation": conformance.Compare(nil, b).Text()})
		})
	}
}

// Pins CS-GAP-017: a password is found by how it is written. These ways say nothing, so a value
// that is itself at fault, under an ordinary name, is still quoted.
func TestCredentials_UnmarkedWaysOfWritingAPasswordAreQuoted_KnownGap(t *testing.T) {
	for name, in := range map[string]string{
		"an address without a scheme":        "app:s3cretPw1@tcp(db:3306)/orders",
		"JSON written inside a text":         `retry with {"password":"s3cretPw1"}`,
		"a colon in prose":                   "the password: s3cretPw1 was refused",
		"another name":                       "login?pass=s3cretPw1&u=app",
		"a command-line option":              "mysql --password s3cretPw1",
		"an encoded query string":            "next=%2Flogin%3Fpassword%3Ds3cretPw1",
		"the tail of an unquoted value":      "Password=first s3cretPw1;Server=db",
		"an unencoded slash in the userinfo": "ftp://app:s3cretPw1/x@host",
	} {
		if got := redact.Credentials(in); !strings.Contains(got, "s3cretPw1") {
			t.Errorf("%s is now masked (%q): update CS-GAP-017", name, got)
		}
	}
}
