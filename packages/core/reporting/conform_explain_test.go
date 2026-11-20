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

	js, err := ConformJSON(r, nil)
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

	junit, err := ConformJUnit(r)
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
