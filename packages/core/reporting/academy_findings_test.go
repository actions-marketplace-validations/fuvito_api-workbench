package reporting_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestConformText_RequestSideFailureIsLabelled(t *testing.T) {
	res := &conformance.ConformResult{Operations: []conformance.OperationResult{{
		Method: "GET", Path: "/items/abc", StatusCode: 400,
		RequestViolations: []openapi.ValidationError{{Kind: openapi.KindRequestParameter, Subject: "itemId", Message: "itemId must be an integer"}},
	}}}
	out := reporting.ConformText(res, nil)
	if !strings.Contains(out, "FAIL  GET     /items/abc  400") || !strings.Contains(out, "the request does not fit the spec; the answer was not judged") {
		t.Fatalf("%s", out)
	}
}

func TestNegativeText_AuthProbeSaysToFixAuthentication(t *testing.T) {
	res := &conformance.NegativeResult{Probes: []conformance.NegativeProbeResult{
		{Description: "no credentials", Kind: openapi.ProbeMissingAuth, StatusCode: 200},
	}}
	out := reporting.NegativeText(res)
	if !strings.Contains(out, "1 probe  ·") || !strings.Contains(out, "fix the authentication rule") || strings.Contains(out, "fix server validation") {
		t.Fatalf("%s", out)
	}
}

func TestText_SkippedStepSaysWhy(t *testing.T) {
	res := &scenario.Result{
		Scenario: &domain.Scenario{Title: "s"},
		Steps: []scenario.StepResult{
			{Label: "first", Passed: false, Error: "boom"},
			{Label: "second", Skipped: true, SkipReason: "step 1 (first) failed and its failPolicy is stop; set failPolicy: continue to run the steps after a failure"},
		},
	}
	out := reporting.Text(res)
	if !strings.Contains(out, "SKIP  second") || !strings.Contains(out, "skipped: step 1 (first) failed") || !strings.Contains(out, "failPolicy: continue") {
		t.Fatalf("%s", out)
	}
}

func TestText_FailedSchemaAssertionCarriesACheckID(t *testing.T) {
	msg := "Error at \"/error\": property \"error\" is missing\nSchema:\n  {}\n\nValue:\n  {}\n"
	res := &scenario.Result{
		Scenario: &domain.Scenario{Title: "s"},
		Steps: []scenario.StepResult{{Label: "get", Assertions: []domain.AssertionResult{
			{Assertion: domain.Assertion{Type: domain.AssertionSchema}, Passed: false, Message: msg},
		}}},
	}
	out := reporting.Text(res)
	if !strings.Contains(out, "[CS-RESP-005]") || !strings.Contains(out, "Why:") || !strings.Contains(out, "Fix:") {
		t.Fatalf("%s", out)
	}
	js, err := reporting.JSON(res)
	if err != nil || !strings.Contains(string(js), `"checkId": "CS-RESP-005"`) {
		t.Fatalf("%v\n%s", err, js)
	}
}
