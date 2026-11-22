package reporting

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const explainSecret = "sk-live-ABCDEFGH12345"

// A failing operation whose response value contains a secret, so the explanation (which quotes the
// value) has to be redacted like the raw violation line.
func explainFixture() *conformance.ConformResult {
	msg := "Error at \"/token\": value must be a number\nSchema:\n  {\n    \"type\": \"number\"\n  }\n\nValue:\n  \"" + explainSecret + "\"\n"
	return &conformance.ConformResult{
		SpecTitle: "Shop", SpecVersion: "1", BaseURL: "http://x",
		Secrets: []string{explainSecret},
		Operations: []conformance.OperationResult{
			{OperationID: "ok", Method: "GET", Path: "/ok", StatusCode: 200},
			{OperationID: "bad", Method: "GET", Path: "/bad", StatusCode: 200, Violations: []openapi.ValidationError{{Message: msg}}},
		},
	}
}

func TestConformText_ExplanationIsOptIn(t *testing.T) {
	r := explainFixture()
	plain := ConformText(r, nil)
	if strings.Contains(plain, "→") || strings.Contains(plain, "Why:") {
		t.Fatalf("the default text output must not change:\n%s", plain)
	}
	explained := ConformTextExplained(r, nil, true)
	if !strings.Contains(explained, `→ "token" is text, the spec says number. Why:`) || !strings.Contains(explained, "Fix:") {
		t.Fatalf("explanation missing:\n%s", explained)
	}
	if ConformTextExplained(r, nil, false) != plain {
		t.Fatal("explain=false must equal ConformText")
	}
}

func TestConformReports_ExplanationsAreRedacted(t *testing.T) {
	r := explainFixture()
	check := func(name, out string) {
		t.Helper()
		if strings.Contains(out, explainSecret) {
			t.Errorf("%s leaks the secret:\n%s", name, out)
		}
	}
	check("text", ConformTextExplained(r, nil, true))

	js, err := ConformJSON(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("json", string(js))
	var parsed JSONConformReport
	if err := json.Unmarshal(js, &parsed); err != nil {
		t.Fatal(err)
	}
	bad := parsed.Operations[1]
	if len(bad.Violations) != 1 || len(bad.Findings) != 1 {
		t.Fatalf("violations and findings: %+v", bad)
	}
	f := bad.Findings[0]
	if f.Kind != "wrong-type" || f.Where != "token" || f.Why == "" || f.Fix == "" || f.Expected != "number" {
		t.Fatalf("finding: %+v", f)
	}
	if len(parsed.Operations[0].Findings) != 0 {
		t.Fatal("a passing operation has no findings")
	}

	junit, err := ConformJUnit(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("junit", string(junit))
	if !strings.Contains(string(junit), `message="1 violation(s)"`) || !strings.Contains(string(junit), "→ ") {
		t.Fatalf("junit: message must count violations only and the body carries the explanation:\n%s", junit)
	}

	html, err := ConformHTML(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("html", string(html))
	if !strings.Contains(string(html), "Why:") {
		t.Fatal("html is missing the explanation")
	}
	md := string(ConformMarkdown(r, nil, nil))
	check("markdown", md)
	if !strings.Contains(md, "Why:") || !strings.Contains(md, "Fix:") {
		t.Fatalf("markdown is missing the explanation:\n%s", md)
	}
}

func blockedFixture() *conformance.ConformResult {
	return &conformance.ConformResult{
		SpecTitle: "Shop", SpecVersion: "1", BaseURL: "http://x",
		Operations: []conformance.OperationResult{
			{OperationID: "ok", Method: "GET", Path: "/ok", StatusCode: 200},
			{OperationID: "a", Method: "GET", Path: "/a/{{id}}", Blocked: "not sent: no value for {{id}}", Missing: []string{"id"}},
			{OperationID: "b", Method: "POST", Path: "/b", Blocked: "not sent: no value for {{id}}, {{name}}", Missing: []string{"id", "name"}},
		},
	}
}

func TestConformReports_BlockedOperations(t *testing.T) {
	r := blockedFixture()
	text := ConformText(r, nil)
	for _, want := range []string{
		"FAIL  GET     /a/{{id}}", "✗ not sent: no value for {{id}}",
		"2 operations were not sent because nothing provides: id, name.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text is missing %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "FAILED  3 operations  ·  1 passed  ·  2 failed") {
		t.Errorf("a blocked operation counts as failed:\n%s", text)
	}
	if strings.Count(text, "provides: id, name") != 1 {
		t.Errorf("the missing names are listed once, not per operation:\n%s", text)
	}

	js, err := ConformJSON(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var parsed JSONConformReport
	if err := json.Unmarshal(js, &parsed); err != nil {
		t.Fatal(err)
	}
	if strings.Join(parsed.MissingValues, ",") != "id,name" || parsed.Passed {
		t.Errorf("json: %+v", parsed)
	}
	if b := parsed.Operations[2]; b.Blocked == "" || strings.Join(b.Missing, ",") != "id,name" || len(b.Findings) != 1 || b.Findings[0].Kind != "missing-value" {
		t.Errorf("json operation: %+v", b)
	}

	junit, err := ConformJUnit(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(junit), "not sent: no value for {{id}}") || !strings.Contains(string(junit), `failures="2"`) {
		t.Errorf("junit:\n%s", junit)
	}
	html, err := ConformHTML(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "nothing provides: id, name") {
		t.Error("html is missing the missing-values line")
	}
	if !strings.Contains(string(ConformMarkdown(r, nil, nil)), "not sent: no value for {{id}}") {
		t.Error("markdown is missing the blocked line")
	}

	clean := &conformance.ConformResult{Operations: []conformance.OperationResult{{Method: "GET", Path: "/ok", StatusCode: 200}}}
	if strings.Contains(ConformText(clean, nil), "not sent") {
		t.Error("a run with nothing blocked must not mention it")
	}
}

func headerFixture() *conformance.ConformResult {
	return &conformance.ConformResult{
		SpecTitle: "Shop", SpecVersion: "1", BaseURL: "http://x",
		Operations: []conformance.OperationResult{{
			OperationID: "list", Method: "GET", Path: "/items", SpecPath: "/items", StatusCode: 200,
			Violations: []openapi.ValidationError{{
				Kind: openapi.KindMissingHeader, Subject: "X-Total-Count",
				Message: `GET /items 200: the response header "X-Total-Count" is required by the spec but was not sent`,
			}},
		}},
	}
}

// Every failure says where it is, in the response and in the spec, in every output that explains it.
func TestConformReports_NameWhereTheProblemIs(t *testing.T) {
	r := headerFixture()
	wantInLine := `Where: response header "X-Total-Count" (spec: GET /items → response 200 → headers → X-Total-Count)`

	if text := ConformTextExplained(r, nil, true); !strings.Contains(text, wantInLine) || !strings.Contains(text, `✗ GET /items 200: the response header "X-Total-Count" is required by the spec but was not sent`) {
		t.Errorf("text must show the raw message and where:\n%s", text)
	}
	md := string(ConformMarkdown(r, nil, nil))
	if !strings.Contains(md, wantInLine) {
		t.Errorf("markdown:\n%s", md)
	}
	html, err := ConformHTML(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "X-Total-Count") {
		t.Error("html must name the header")
	}
	junit, err := ConformJUnit(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(junit), "X-Total-Count") || !strings.Contains(string(junit), "Where:") {
		t.Errorf("junit:\n%s", junit)
	}

	js, err := ConformJSON(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var parsed JSONConformReport
	if err := json.Unmarshal(js, &parsed); err != nil {
		t.Fatal(err)
	}
	f := parsed.Operations[0].Findings
	if len(f) != 1 || f[0].Kind != "missing-header" || f[0].Part != "response header" || f[0].Where != "X-Total-Count" ||
		f[0].Spec != "GET /items → response 200 → headers → X-Total-Count" || f[0].Why == "" || f[0].Fix == "" {
		t.Errorf("json finding: %+v", f)
	}

	if strings.Contains(ConformText(r, nil), "Where:") {
		t.Error("without --explain the screen output keeps its old shape")
	}
}
