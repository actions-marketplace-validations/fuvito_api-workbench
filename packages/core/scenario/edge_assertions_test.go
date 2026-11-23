package scenario_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func edgeServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Dup", "one")
		w.Header().Add("X-Dup", "two")
		w.Header().Add("Set-Cookie", "a=1; Path=/")
		w.Header().Add("Set-Cookie", "b=2; Path=/")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9007199254740993,"big":9223372036854775808,"n":1234567,"price":12.5,"items":[{"price":3},{"price":20}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runEdge(t *testing.T, url string, step domain.ScenarioStep) *scenario.Result {
	t.Helper()
	step.Label = "s"
	step.Request = &domain.Request{Method: domain.MethodGET, URL: url}
	sc := &domain.Scenario{Title: "edge", Steps: []domain.ScenarioStep{step}}
	return scenario.New(executor.New(executor.Options{})).Run(context.Background(), sc, domain.NewEnvironment("e"))
}

// F-030: a header sent more than once is one list; every value can be asserted and extracted.
func TestHeaders_RepeatedValuesAreJoined(t *testing.T) {
	srv := edgeServer(t)
	res := runEdge(t, srv.URL, domain.ScenarioStep{
		Assertions: []domain.Assertion{
			{Type: domain.AssertionHeader, Path: "X-Dup", Operator: domain.OperatorContains, Expected: "two"},
			{Type: domain.AssertionHeader, Path: "Set-Cookie", Operator: domain.OperatorContains, Expected: "b=2"},
		},
		Extractions: []domain.Extraction{{Name: "dup", Source: domain.ExtractHeader, Path: "x-dup"}},
	})
	if !res.Passed || res.Steps[0].Extracted["dup"] != "one, two" {
		t.Errorf("passed=%v extracted=%v assertions=%+v", res.Passed, res.Steps[0].Extracted, res.Steps[0].Assertions)
	}
}

// F-031: integers keep their digits, and a number is never written in exponent form.
func TestJSONPath_KeepsLargeIntegersAndPlainNumbers(t *testing.T) {
	srv := edgeServer(t)
	res := runEdge(t, srv.URL, domain.ScenarioStep{
		Assertions: []domain.Assertion{
			{Type: domain.AssertionJSONPath, Path: "$.id", Operator: domain.OperatorEq, Expected: "9007199254740993"},
			{Type: domain.AssertionJSONPath, Path: "$.big", Operator: domain.OperatorEq, Expected: "9223372036854775808"},
			{Type: domain.AssertionJSONPath, Path: "$.n", Operator: domain.OperatorEq, Expected: "1234567"},
			{Type: domain.AssertionJSONPath, Path: "$.price", Operator: domain.OperatorGt, Expected: "12"},
		},
		Extractions: []domain.Extraction{{Name: "bigId", Source: domain.ExtractJSONPath, Path: "$.big"}},
	})
	if !res.Passed || res.Steps[0].Extracted["bigId"] != "9223372036854775808" {
		t.Errorf("passed=%v extracted=%v assertions=%+v", res.Passed, res.Steps[0].Extracted, res.Steps[0].Assertions)
	}
}

// F-032: an extraction that finds nothing fails the step and the run.
func TestExtraction_FailureFailsTheStepAndTheRun(t *testing.T) {
	srv := edgeServer(t)
	res := runEdge(t, srv.URL, domain.ScenarioStep{
		Extractions: []domain.Extraction{
			{Name: "nope", Source: domain.ExtractJSONPath, Path: "$.not.there"},
			{Name: "nohdr", Source: domain.ExtractHeader, Path: "X-Not-There"},
		},
	})
	p, f, _ := res.Counts()
	if res.Passed || res.Steps[0].Passed || p != 0 || f != 1 {
		t.Errorf("run passed=%v step passed=%v counts=%d/%d: a failed extraction must fail the run", res.Passed, res.Steps[0].Passed, p, f)
	}
	if res.Steps[0].Error == "" {
		t.Error("the step says why")
	}
}
