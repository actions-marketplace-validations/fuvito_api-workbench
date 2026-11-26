package reporting_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// A run of negative probes that sent none must not read as a pass.
func TestNoProbesIsNothingCheckedNotPassed(t *testing.T) {
	for name, c := range map[string]struct {
		probes []conformance.NegativeProbeResult
		why    string
	}{
		"none applies": {nil, "no negative probe applies"},
		"all skipped":  {[]conformance.NegativeProbeResult{{Method: "POST", Path: "/x", Description: "POST /x — no credentials", Skipped: true}}, "every probe of the selected operations was skipped"},
	} {
		t.Run(name, func(t *testing.T) {
			neg := &conformance.NegativeResult{SpecTitle: "a", SpecVersion: "1", Probes: c.probes}
			html, err := reporting.ConformHTML(&conformance.ConformResult{SpecTitle: "a"}, nil, neg)
			if err != nil {
				t.Fatal(err)
			}
			for format, out := range map[string]string{
				"text": reporting.NegativeText(neg), "html": string(html),
				"markdown": string(reporting.ConformMarkdown(&conformance.ConformResult{SpecTitle: "a"}, nil, neg)),
			} {
				if !strings.Contains(out, "NOTHING CHECKED") || !strings.Contains(out, c.why) {
					t.Errorf("%s: %s", format, out)
				}
			}
			if text := reporting.NegativeText(neg); strings.Contains(text, "PASSED") {
				t.Errorf("reads as a pass:\n%s", text)
			}
		})
	}
	ran := &conformance.NegativeResult{Probes: []conformance.NegativeProbeResult{{Description: "p", StatusCode: 422}}}
	if text := reporting.NegativeText(ran); !strings.Contains(text, "PASSED  1 probe ") || strings.Contains(text, "NOTHING CHECKED") {
		t.Errorf("a run with probes keeps its verdict:\n%s", text)
	}
}

func TestAnAssertionWithoutAnExpectedValueHasAnEmptyOne(t *testing.T) {
	result := &scenario.Result{Scenario: &domain.Scenario{Title: "s"}, Steps: []scenario.StepResult{{Label: "a", Passed: true, Assertions: []domain.AssertionResult{
		{Assertion: domain.Assertion{Type: domain.AssertionSchema, Schema: "getX"}, Passed: true},
		{Assertion: domain.Assertion{Type: domain.AssertionStatus, Expected: 0}, Passed: true},
	}}}}
	out, err := reporting.JSON(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "nil") {
		t.Errorf("%s", out)
	}
	// zero is a value, not a missing one
	if !strings.Contains(string(out), `"expected": "0"`) {
		t.Errorf("%s", out)
	}
}
