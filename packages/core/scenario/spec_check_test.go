package scenario_test

import (
	"context"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestRunner_SpecCheckStates(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(conformSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	bad := jsonServer(`{"name":"no id"}`)
	defer bad.Close()
	good := jsonServer(`{"id":1}`)
	defer good.Close()
	off := false
	cont := func(s *domain.ScenarioStep) { s.FailPolicy = domain.FailPolicyContinue }

	s := &domain.Scenario{Steps: []domain.ScenarioStep{
		getStep("matches", good.URL+"/users/1", cont),
		getStep("violates", bad.URL+"/users/1", cont),
		getStep("opted out", bad.URL+"/users/1", cont, func(s *domain.ScenarioStep) { s.ValidateSpec = &off }),
		getStep("not in spec", good.URL+"/other", cont),
		getStep("unreachable", "http://127.0.0.1:1/users/1", cont),
	}}
	res := scenario.NewWithSpec(executor.New(executor.Options{}), spec).Run(context.Background(), s, nil)

	want := []struct {
		state  scenario.SpecState
		reason string
		nViol  int
	}{
		{scenario.SpecMatched, "", 0},
		{scenario.SpecViolations, "", 1},
		{scenario.SpecNotChecked, "spec validation is turned off for this step", 0},
		{scenario.SpecNotChecked, "no matching operation in the spec", 0},
		{scenario.SpecNotChecked, "the step did not complete", 0},
	}
	for i, w := range want {
		got := res.Steps[i].Spec
		if got.State != w.state || got.Reason != w.reason || len(got.Violations) != w.nViol {
			t.Errorf("step %d (%s): got %+v, want state=%s reason=%q violations=%d", i, res.Steps[i].Label, got, w.state, w.reason, w.nViol)
		}
	}
}

func TestRunner_NoSpecLeavesSpecCheckEmpty(t *testing.T) {
	good := jsonServer(`{"id":1}`)
	defer good.Close()
	res := scenario.New(executor.New(executor.Options{})).Run(context.Background(),
		&domain.Scenario{Steps: []domain.ScenarioStep{getStep("a", good.URL+"/users/1")}}, nil)
	if res.Steps[0].Spec.State != "" {
		t.Errorf("no spec: state = %q, want empty", res.Steps[0].Spec.State)
	}
}
