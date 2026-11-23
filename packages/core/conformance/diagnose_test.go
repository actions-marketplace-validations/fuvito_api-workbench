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
	if other.Kind != FindingConstraint || other.CheckID != "CS-RESP-007" || other.Where != "age" || other.Summary != "number must be at most 120" {
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
