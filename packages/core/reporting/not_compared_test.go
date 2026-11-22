package reporting_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// A step whose response was not compared with the spec is said so in the text report (CS-NC-001),
// so a run that compared nothing cannot be read as a run that found nothing.
func TestText_SaysWhenResponsesWereNotComparedWithTheSpec(t *testing.T) {
	res := &scenario.Result{
		Scenario: &domain.Scenario{Title: "s"},
		Passed:   true,
		Steps: []scenario.StepResult{
			{Label: "a", Passed: true, Spec: scenario.SpecCheck{State: scenario.SpecMatched}},
			{Label: "b", Passed: true, Spec: scenario.SpecCheck{State: scenario.SpecNotChecked, Reason: "no matching operation in the spec"}},
		},
	}
	out := reporting.Text(res)
	if !strings.Contains(out, "Spec: 1 of 2 responses were not compared with the spec (no matching operation in the spec: 1)") {
		t.Errorf("missing line:\n%s", out)
	}
	res.Steps[1].Spec = scenario.SpecCheck{State: scenario.SpecMatched}
	if strings.Contains(reporting.Text(res), "were not compared") {
		t.Error("no line expected when every response was compared")
	}
}
