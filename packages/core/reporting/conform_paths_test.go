package reporting_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

// A run with one operation of each outcome; two have a path parameter.
func mixedRun() *conformance.ConformResult {
	return &conformance.ConformResult{SpecTitle: "Shop", SpecVersion: "1", BaseURL: "http://x", Operations: []conformance.OperationResult{
		{Method: "GET", Path: "/ok/{{id}}", SpecPath: "/ok/{id}", StatusCode: 200},
		{Method: "GET", Path: "/bad/{{id}}", SpecPath: "/bad/{id}", StatusCode: 500,
			Violations: []openapi.ValidationError{{Message: "status 500 is not declared"}}},
		{Method: "PUT", Path: "/unsent/{{userId}}", SpecPath: "/unsent/{userId}", Blocked: "not sent: no value for {{userId}}", Missing: []string{"userId"}},
		{Method: "DELETE", Path: "/skipped/{{id}}", SpecPath: "/skipped/{id}", Skipped: true},
	}}
}

// Every report a person reads names an operation the way the spec does, in the operation list and in
// the coverage lists. (The message of an unsent operation still names the missing {{userId}}.)
func TestConformReportsNameOperationsByTheSpecPath(t *testing.T) {
	r := mixedRun()
	cov := conformance.ComputeCoverage(r)
	junit, err := reporting.ConformJUnit(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	html, err := reporting.ConformHTML(r, cov, nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]string{
		"text":     reporting.ConformTextExplained(r, cov, true),
		"markdown": string(reporting.ConformMarkdown(r, cov, nil)),
		"html":     string(html),
		"junit":    string(junit),
		"github":   reporting.ConformGitHub(r, nil, nil),
	} {
		for _, sent := range []string{"/ok/{{", "/bad/{{", "/unsent/{{", "/skipped/{{"} {
			if strings.Contains(out, sent) {
				t.Errorf("%s shows the sent path %s…:\n%s", name, sent, out)
			}
		}
		if !strings.Contains(out, "/bad/{id}") {
			t.Errorf("%s does not show the spec path of the failing operation:\n%s", name, out)
		}
	}
	for _, want := range []string{`name="GET /ok/{id}"`, `name="PUT /unsent/{userId}"`, `name="DELETE /skipped/{id}"`} {
		if !strings.Contains(string(junit), want) {
			t.Errorf("junit is missing %s", want)
		}
	}
}

// In JSON, path keeps its value (consumers and baselines rely on it) and specPath is added.
func TestConformJSONKeepsPathAndAddsSpecPath(t *testing.T) {
	r := mixedRun()
	data, err := reporting.ConformJSON(r, conformance.ComputeCoverage(r), nil)
	if err != nil {
		t.Fatal(err)
	}
	type op struct{ Path, SpecPath string }
	var got struct {
		Operations []op
		Coverage   struct{ CoveredOps, NotCoveredOps []op }
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	all := append(append(got.Operations, got.Coverage.CoveredOps...), got.Coverage.NotCoveredOps...)
	if len(all) != 8 {
		t.Fatalf("expected 4 operations and 4 coverage entries, got %d", len(all))
	}
	for _, o := range all {
		if !strings.Contains(o.Path, "{{") || o.SpecPath != conformance.SpecStyle(o.Path) {
			t.Errorf("path %q, specPath %q", o.Path, o.SpecPath)
		}
	}
}

// "not sent" is counted apart from "failed"; the verdict and the run's own counts do not change.
func TestConformSummariesSeparateNotSentFromFailed(t *testing.T) {
	r := mixedRun()
	html, err := reporting.ConformHTML(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	const tally = "4 operations  ·  1 passed  ·  1 failed  ·  1 not sent  ·  1 skipped"
	for name, out := range map[string]string{
		"text": reporting.ConformText(r, nil), "markdown": string(reporting.ConformMarkdown(r, nil, nil)), "html": string(html),
	} {
		if !strings.Contains(out, tally) {
			t.Errorf("%s: missing %q in\n%s", name, tally, out)
		}
	}
	if out := reporting.ConformText(r, nil); !strings.Contains(out, "FAILED  4 operations") {
		t.Errorf("the verdict stays FAILED:\n%s", out)
	}
	if out := reporting.ConformGitHub(r, nil, nil); !strings.Contains(out, "4 operations: 1 passed, 1 failed, 1 not sent, 1 skipped.") {
		t.Errorf("github notice:\n%s", out)
	}
	if _, failed, _ := r.Counts(); failed != 2 {
		t.Errorf("Counts still counts an unsent operation as failed: %d", failed)
	}

	r.Operations = append(r.Operations[:2], r.Operations[3])
	if out := reporting.ConformText(r, nil); !strings.Contains(out, "3 operations  ·  1 passed  ·  1 failed  ·  1 skipped") || strings.Contains(out, "not sent") {
		t.Errorf("with nothing unsent the line is as it was:\n%s", out)
	}
}

// Only what is shown changes: a baseline still identifies the operation by the stored path, so a
// baseline recorded before this change still matches.
func TestBaselineKeysKeepTheStoredPath(t *testing.T) {
	violations := conformance.Violations(mixedRun(), nil)
	if len(violations) == 0 {
		t.Fatal("no violation to record")
	}
	for _, v := range violations {
		if v.Operation != "GET /bad/{{id}}" {
			t.Errorf("stored operation changed: %q", v.Operation)
		}
		if s := v.String(); !strings.HasPrefix(s, "GET /bad/{id}  ") {
			t.Errorf("shown as %q", s)
		}
	}
}
