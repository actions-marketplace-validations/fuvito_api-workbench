package conformance

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

func ve(msg string) openapi.ValidationError { return openapi.ValidationError{Message: msg} }

func TestDiagnose_MissingField(t *testing.T) {
	f := Diagnose(OperationResult{Violations: []openapi.ValidationError{ve(`Error at "/0/name": property "name" is missing`)}})
	if len(f) != 1 || f[0].Kind != FindingMissingField || f[0].Where != "[0].name" {
		t.Fatalf("%+v", f)
	}
	if f[0].Summary != `"name" is missing from the response` || !strings.Contains(f[0].Why, `requires`) && !strings.Contains(f[0].Why, `required`) {
		t.Fatalf("summary/why: %+v", f[0])
	}
	if !strings.Contains(f[0].Fix, `Return "name"`) || f[0].Detail == "" {
		t.Fatalf("fix/detail: %+v", f[0])
	}
}

func TestDiagnose_WrongTypeWithValue(t *testing.T) {
	msg := "Error at \"/price\": value must be a number\nSchema:\n  {\n    \"type\": \"number\"\n  }\n\nValue:\n  \"9.99\"\n"
	f := Diagnose(OperationResult{Violations: []openapi.ValidationError{ve(msg)}})
	if len(f) != 1 {
		t.Fatalf("%+v", f)
	}
	g := f[0]
	if g.Kind != FindingWrongType || g.Where != "price" || g.Expected != "number" || g.Got != `"9.99"` {
		t.Fatalf("%+v", g)
	}
	if g.Summary != `"price" is text, the spec says number` {
		t.Fatalf("summary = %q", g.Summary)
	}
	if !strings.Contains(g.Why, `text ("9.99")`) || !strings.Contains(g.Why, "a number") || g.Detail != msg {
		t.Fatalf("why = %q", g.Why)
	}
}

func TestDiagnose_WrongTypeNames(t *testing.T) {
	for want, wantPhrase := range map[string]string{"an integer": "an integer", "a string": "a string", "a boolean": "a boolean"} {
		msg := "Error at \"/0/id\": value must be " + want + "\nSchema:\n  {}\n\nValue:\n  1.5\n"
		g := Diagnose(OperationResult{Violations: []openapi.ValidationError{ve(msg)}})[0]
		if !strings.Contains(g.Why, wantPhrase) || g.Where != "[0].id" {
			t.Errorf("%s: %+v", want, g)
		}
	}
}

func TestDiagnose_OtherAndRequestAndNoResponse(t *testing.T) {
	other := Diagnose(OperationResult{Violations: []openapi.ValidationError{ve(`Error at "/age": number must be at most 120`)}})[0]
	if other.Kind != FindingConstraint || other.CheckID != "CS-RESP-007" || other.Where != "age" || other.Summary != "\"age\" is at most 120" {
		t.Fatalf("%+v", other)
	}
	unknown := Diagnose(OperationResult{Violations: []openapi.ValidationError{ve("something unexpected\nmore")}})[0]
	if unknown.Kind != FindingOther || unknown.Summary != "something unexpected" {
		t.Fatalf("%+v", unknown)
	}

	req := Diagnose(OperationResult{RequestViolations: []openapi.ValidationError{ve("value %7Bid%7D: an invalid integer: invalid syntax")}})[0]
	if req.Kind != FindingRequest || !strings.Contains(req.Fix, "environment") {
		t.Fatalf("%+v", req)
	}

	none := Diagnose(OperationResult{Error: "dial tcp: connection refused"})[0]
	if none.Kind != FindingNoResponse || none.Got != "dial tcp: connection refused" || !strings.Contains(none.Fix, "base URL") {
		t.Fatalf("%+v", none)
	}
}

func TestDiagnose_NothingToExplainForPassedOrSkipped(t *testing.T) {
	if f := Diagnose(OperationResult{}); len(f) != 0 {
		t.Fatalf("a passing operation has no findings: %+v", f)
	}
	if f := Diagnose(OperationResult{Skipped: true, Violations: []openapi.ValidationError{ve("x")}}); len(f) != 0 {
		t.Fatalf("a skipped operation has no findings: %+v", f)
	}
}

func TestDisplayPath(t *testing.T) {
	for in, want := range map[string]string{"": "", "/price": "price", "/0/id": "[0].id", "/items/2/tags/0": "items[2].tags[0]"} {
		if got := displayPath(in); got != want {
			t.Errorf("displayPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDiagnose_ConstraintNamesTheRuleAndTheValue(t *testing.T) {
	cases := []struct{ msg, rule, value string }{
		{"Error at \"/status\": value is not one of the allowed values [\"active\",\"inactive\"]\nSchema:\n  {}\n\nValue:\n  \"zzz\"\n", `is one of ["active","inactive"]`, `"zzz"`},
		{"Error at \"/n\": number must be at least 1\nSchema:\n  {}\n\nValue:\n  0\n", "is at least 1", "0"},
		{"Error at \"/s\": maximum string length is 3\nSchema:\n  {}\n\nValue:\n  \"abcdef\"\n", "has at most 3 characters", `"abcdef"`},
		{"Error at \"/at\": string doesn't match the format \"date-time\": string doesn't match pattern \"^[0-9]{4}$\"", "RFC 3339 date-time", ""},
	}
	for _, c := range cases {
		f := Diagnose(OperationResult{Violations: []openapi.ValidationError{ve(c.msg)}})[0]
		if f.Kind != FindingConstraint || !strings.Contains(f.Summary, c.rule) || !strings.Contains(f.Why, c.rule) {
			t.Errorf("%q: %+v", c.rule, f)
		}
		if c.value != "" && (f.Got != c.value || !strings.Contains(f.Why, c.value)) {
			t.Errorf("value %s not shown: %+v", c.value, f)
		}
		if strings.Contains(f.Summary, "^[0-9]") {
			t.Errorf("the pattern leaked into the summary: %q", f.Summary)
		}
	}
}

func TestDiagnose_UndeclaredStatusFitsTheStatus(t *testing.T) {
	op := OperationResult{Method: "GET", Path: "/ghost", SpecPath: "/ghost", StatusCode: 404, BodySnippet: `{"error":"no route"}`}
	f := diagnoseViolation(op, openapi.ValidationError{Kind: openapi.KindUndeclaredStatus, Subject: "404", Message: "GET /ghost 404: the spec declares only 200"})
	if !strings.Contains(f.Fix, "route exists on the server") || !strings.Contains(f.Got, `no route`) {
		t.Fatalf("%+v", f)
	}
	op.StatusCode = 400
	f = diagnoseViolation(op, openapi.ValidationError{Kind: openapi.KindUndeclaredStatus, Subject: "400", Message: "GET /ghost 400: the spec declares only 200"})
	if !strings.Contains(f.Fix, "response body under Got") {
		t.Fatalf("%+v", f)
	}
}

func TestNetworkErrorKind(t *testing.T) {
	for msg, want := range map[string]string{
		"Get \"http://x\": dial tcp 127.0.0.1:1: connect: connection refused": NetUnreachable,
		"Get \"http://x\": context deadline exceeded":                         NetTimeout,
		"Get \"http://x\": stopped after 10 redirects":                        NetRedirectLoop,
		"reading response: unexpected EOF":                                    NetBrokenBody,
		"Get \"https://x\": x509: certificate signed by unknown authority":    NetTLS,
		"boom": NetOther,
	} {
		if got := NetworkErrorKind(msg); got != want {
			t.Errorf("%q: %s, want %s", msg, got, want)
		}
	}
	if why, fix := networkAdvice(NetTimeout); strings.Contains(fix, "server is running") || why == "" {
		t.Errorf("a timeout must not tell the user to start the server: %q", fix)
	}
}
