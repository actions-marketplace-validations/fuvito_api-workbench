package reporting

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// warnFixture has one passing operation and one whose only finding is CS-RESP-006, which the
// caller lowers to a warning.
func warnFixture(failOn checks.Severity) *conformance.ConformResult {
	r := explainFixture()
	r.FailOn = failOn
	return r
}

func TestConformReports_WarningsPassTheRun(t *testing.T) {
	t.Cleanup(checks.WithSeverity("CS-RESP-006", checks.Warning))
	r := warnFixture("")

	text := ConformTextExplained(r, nil, true)
	for _, want := range []string{"WARN  GET     /bad", "PASSED with 1 warning", "! ", "(warning) [CS-RESP-006]"} {
		if !strings.Contains(text, want) {
			t.Errorf("text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "FAILED") || strings.Contains(text, "✗") {
		t.Errorf("a warning-only run must not look failed:\n%s", text)
	}

	js, err := ConformJSON(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var rep JSONConformReport
	if err := json.Unmarshal(js, &rep); err != nil {
		t.Fatal(err)
	}
	if !rep.Passed || rep.Warnings != 1 || !rep.Operations[1].Passed || rep.Operations[1].Warnings != 1 {
		t.Errorf("json: passed=%v warnings=%d op=%+v", rep.Passed, rep.Warnings, rep.Operations[1])
	}
	if f := rep.Operations[1].Findings; len(f) != 1 || f[0].Severity != "warning" {
		t.Errorf("finding severity = %+v", f)
	}

	xml, err := ConformJUnit(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(xml), `failures="0"`) || !strings.Contains(string(xml), "<system-out>warning: ") {
		t.Errorf("junit: a warning is a passing test case with system-out:\n%s", xml)
	}

	html, err := ConformHTML(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "WARN") || !strings.Contains(string(html), "1 with warnings") {
		t.Errorf("html lacks the warning")
	}
	if md := string(ConformMarkdown(r, nil, nil)); !strings.Contains(md, "⚠️") {
		t.Errorf("markdown lacks the warning icon:\n%s", md)
	}
}

func TestConformReports_FailOnWarningFailsTheRun(t *testing.T) {
	t.Cleanup(checks.WithSeverity("CS-RESP-006", checks.Warning))
	r := warnFixture(checks.Warning)

	text := ConformText(r, nil)
	if !strings.Contains(text, "FAIL  GET     /bad") || !strings.Contains(text, "FAILED") {
		t.Errorf("text:\n%s", text)
	}
	js, _ := ConformJSON(r, nil, nil)
	var rep JSONConformReport
	if err := json.Unmarshal(js, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Passed || rep.Operations[1].Passed {
		t.Errorf("json passed=%v op=%v, want false", rep.Passed, rep.Operations[1].Passed)
	}
	xml, _ := ConformJUnit(r, nil)
	if !strings.Contains(string(xml), `failures="1"`) {
		t.Errorf("junit:\n%s", xml)
	}
}

func TestConformText_DefaultSeveritiesChangeNothing(t *testing.T) {
	text := ConformText(explainFixture(), nil)
	if !strings.Contains(text, "FAIL  GET     /bad") || !strings.Contains(text, "✗") || strings.Contains(text, "warning") {
		t.Errorf("a failure must look as before:\n%s", text)
	}
}
