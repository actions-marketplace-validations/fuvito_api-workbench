package scenario_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/assertions"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// A duration assertion compares the response time with a limit (CS-GAP-005, scenario assertions).
func TestDuration_AssertionComparesResponseTime(t *testing.T) {
	ev := assertions.New()
	resp := &domain.Response{Duration: 300 * time.Millisecond}
	run := func(op domain.Operator, expected any) domain.AssertionResult {
		return ev.Evaluate(domain.Assertion{Type: domain.AssertionDuration, Operator: op, Expected: expected}, resp)
	}
	if r := run(domain.OperatorLt, 500); !r.Passed || r.Actual != "300" {
		t.Errorf("300 ms < 500 should pass: %+v", r)
	}
	r := run(domain.OperatorLt, 200)
	if r.Passed || !strings.Contains(r.Message, "took 300 ms") || !strings.Contains(r.Message, "200 ms") {
		t.Errorf("300 ms < 200 should fail and say how long it took: %+v", r)
	}
	if r := run(domain.OperatorGt, "250"); !r.Passed {
		t.Errorf("a numeric string works: %+v", r)
	}
	if got := reporting.Text(&scenario.Result{Scenario: &domain.Scenario{Title: "s"}, Steps: []scenario.StepResult{{Label: "a", Passed: true, Assertions: []domain.AssertionResult{run(domain.OperatorLt, 500)}}}, Passed: true}); !strings.Contains(got, "duration lt 500 ms") {
		t.Errorf("report label missing:\n%s", got)
	}
}

func TestDuration_ValidationNeedsMillisecondsAndASaneOperator(t *testing.T) {
	mk := func(a domain.Assertion) []scenario.Issue {
		return scenario.Validate(&domain.Scenario{Title: "s", Steps: []domain.ScenarioStep{{Label: "a", Request: getReq(), Assertions: []domain.Assertion{a}}}}, "")
	}
	if issues := mk(domain.Assertion{Type: domain.AssertionDuration, Operator: domain.OperatorLt, Expected: 500}); len(issues) != 0 {
		t.Errorf("valid: %v", issues)
	}
	if issues := mk(domain.Assertion{Type: domain.AssertionDuration, Operator: domain.OperatorLt, Expected: "{{limit}}"}); len(issues) != 0 {
		t.Errorf("a variable is allowed: %v", issues)
	}
	if issues := mk(domain.Assertion{Type: domain.AssertionDuration, Operator: domain.OperatorContains, Expected: 500}); len(issues) == 0 {
		t.Error("contains makes no sense for a duration")
	}
	if issues := mk(domain.Assertion{Type: domain.AssertionDuration, Operator: domain.OperatorLt, Expected: "fast"}); len(issues) == 0 {
		t.Error("expected must be a number of milliseconds")
	}
}

// End to end: a slow server fails the step, a fast limit-free step passes.
func TestDuration_SlowServerFailsTheStep(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	sc := &domain.Scenario{Title: "slow", Steps: []domain.ScenarioStep{{
		Label:   "slow call",
		Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
		Assertions: []domain.Assertion{
			{Type: domain.AssertionStatus, Expected: 200},
			{Type: domain.AssertionDuration, Operator: domain.OperatorLt, Expected: 50},
		},
	}}}
	res := newRunner().Run(context.Background(), sc, nil)
	if res.Passed {
		t.Fatal("a 150 ms response must fail a 50 ms limit")
	}
	a := res.Steps[0].Assertions
	if !a[0].Passed || a[1].Passed || !strings.Contains(a[1].Message, "took") {
		t.Errorf("assertions: %+v", a)
	}
}
