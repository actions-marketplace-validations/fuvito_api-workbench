package reporting_test

import (
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func mustHTML(t *testing.T, b []byte, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("html report: %v", err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "<!DOCTYPE html>") || !strings.Contains(s, "</html>") {
		t.Fatalf("not a complete HTML document:\n%s", s)
	}
	return s
}

func TestHTML_Passed(t *testing.T) {
	b, err := reporting.HTML(passedResult("Lifecycle"), nil)
	s := mustHTML(t, b, err)
	for _, want := range []string{"Lifecycle", "PASSED", "step one", `badge pass`, "status eq 200"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(s, "FAILED") {
		t.Error("passed report must not say FAILED")
	}
}

func TestHTML_Failed(t *testing.T) {
	b, err := reporting.HTML(failedResult("Broken"), nil)
	s := mustHTML(t, b, err)
	for _, want := range []string{"FAILED", `badge fail`, "got 500, want 200"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestHTML_SkippedAndError(t *testing.T) {
	b, err := reporting.HTML(skippedResult("S"), nil)
	if s := mustHTML(t, b, err); !strings.Contains(s, `badge skip`) {
		t.Error("skipped step should render a skip badge")
	}
	b, err = reporting.HTML(errorResult("E"), nil)
	if s := mustHTML(t, b, err); !strings.Contains(s, `badge fail`) || !strings.Contains(s, "error:") {
		t.Error("errored step should render a fail badge and its error")
	}
}

func TestHTML_EscapesUntrustedText(t *testing.T) {
	r := failedResult(`<script>alert("t")</script>`)
	r.Steps[0].Label = `<img src=x onerror=alert(1)>`
	r.Steps[0].Assertions[0].Message = `</li><script>alert(2)</script>`
	b, err := reporting.HTML(r, nil)
	s := mustHTML(t, b, err)
	if strings.Contains(s, "<script>alert") || strings.Contains(s, "<img src=x") {
		t.Errorf("untrusted text was not escaped:\n%s", s)
	}
	if !strings.Contains(s, "&lt;script&gt;") {
		t.Error("expected escaped script text")
	}
}

func TestHTML_OmitsExtractedValues(t *testing.T) {
	r := passedResult("Auth")
	r.Steps[0].Extracted = map[string]string{"token": "s3cr3t-value"}
	b, err := reporting.HTML(r, nil)
	s := mustHTML(t, b, err)
	if strings.Contains(s, "s3cr3t-value") {
		t.Error("extracted value leaked into the HTML report")
	}
	if !strings.Contains(s, "token extracted") {
		t.Error("expected the extracted variable name")
	}
}

func TestHTML_WithCoverage(t *testing.T) {
	b, err := reporting.HTML(passedResult("Cov"), makeCovResult())
	s := mustHTML(t, b, err)
	for _, want := range []string{"Spec coverage", "2 / 3 operations (67%)", "GET /widgets", "DELETE /widgets/{id}", "Not covered"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestHTMLMulti(t *testing.T) {
	results := []*scenario.Result{passedResult("One"), failedResult("Two")}
	b, err := reporting.HTMLMulti("My bundle", results, 80*time.Millisecond)
	s := mustHTML(t, b, err)
	for _, want := range []string{"My bundle", "1/2 scenarios passed", "One — PASSED", "Two — FAILED"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	if !strings.Contains(s, `class="verdict bad"`) {
		t.Error("overall verdict should be failed when one scenario fails")
	}
}

func TestConformHTML(t *testing.T) {
	cov := conformance.ComputeCoverage(makeConformResult(true))
	b, err := reporting.ConformHTML(makeConformResult(true), cov, nil)
	s := mustHTML(t, b, err)
	for _, want := range []string{"test-api 1.0.0", "http://localhost:8080", "GET /health", "PASSED", "Coverage"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}

	b, err = reporting.ConformHTML(makeConformResult(false), nil, nil)
	s = mustHTML(t, b, err)
	if !strings.Contains(s, "FAILED") || !strings.Contains(s, "missing required field") {
		t.Error("failed conformance run should show FAILED and the violation")
	}
}

func TestConformHTML_NegativeFailureFailsVerdict(t *testing.T) {
	neg := &conformance.NegativeResult{
		SpecTitle: "test-api", SpecVersion: "1.0.0",
		Probes: []conformance.NegativeProbeResult{
			{Description: "POST /users without auth", StatusCode: 201},
		},
	}
	b, err := reporting.ConformHTML(makeConformResult(true), nil, neg)
	s := mustHTML(t, b, err)
	if !strings.Contains(s, "Negative conformance probes") || !strings.Contains(s, "server accepted invalid request with 201") {
		t.Error("negative section missing")
	}
	if !strings.Contains(s, `class="verdict bad"`) {
		t.Error("an accepted negative probe must fail the verdict")
	}
}

func TestDiffHTML(t *testing.T) {
	changes := []openapi.DiffChange{
		{ID: "endpoint-removed", Text: "api removed without deprecation", Method: "POST", Path: "/users", Breaking: true, Level: "ERR"},
		{ID: "endpoint-added", Text: "endpoint added", Method: "GET", Path: "/users/{id}", Level: "INFO"},
	}
	b, err := reporting.DiffHTML(&openapi.DiffResult{Changes: changes, HasBreaking: true}, changes)
	s := mustHTML(t, b, err)
	for _, want := range []string{"API Spec Diff", "FAILED", "2 change(s)", "1 breaking", "POST /users", `badge err`, `badge info`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}

	b, err = reporting.DiffHTML(&openapi.DiffResult{}, nil)
	s = mustHTML(t, b, err)
	if !strings.Contains(s, "No changes.") || !strings.Contains(s, "PASSED") {
		t.Error("empty diff should pass with 'No changes.'")
	}
}
