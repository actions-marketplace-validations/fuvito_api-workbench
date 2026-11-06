package scenario_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func step(name string, req *domain.Request) domain.ScenarioStep {
	return domain.ScenarioStep{Name: name, Request: req}
}

func getReq() *domain.Request {
	return &domain.Request{Method: domain.MethodGET, URL: "https://api.example.com/items"}
}

func TestValidate_Valid(t *testing.T) {
	sc := &domain.Scenario{
		Name: "My scenario",
		Steps: []domain.ScenarioStep{
			{
				Name:    "Get items",
				Request: getReq(),
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 200},
				},
				Extractions: []domain.Extraction{
					{Name: "firstId", Source: domain.ExtractJSONPath, Path: "$.items[0].id"},
				},
			},
		},
	}
	if issues := scenario.Validate(sc, ""); len(issues) != 0 {
		t.Errorf("expected no issues, got: %v", issues)
	}
}

func TestValidate_MissingName(t *testing.T) {
	sc := &domain.Scenario{Steps: []domain.ScenarioStep{step("Step 1", getReq())}}
	issues := scenario.Validate(sc, "")
	if !hasMessage(issues, "scenario name is required") {
		t.Error("expected missing name error")
	}
}

func TestValidate_NoSteps(t *testing.T) {
	sc := &domain.Scenario{Name: "Empty"}
	issues := scenario.Validate(sc, "")
	if !hasMessage(issues, "scenario has no steps") {
		t.Error("expected no-steps error")
	}
}

func TestValidate_StepMissingName(t *testing.T) {
	sc := &domain.Scenario{
		Name:  "Test",
		Steps: []domain.ScenarioStep{{Request: getReq()}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessage(issues, "step name is required") {
		t.Error("expected step name error")
	}
}

func TestValidate_DuplicateStepName(t *testing.T) {
	sc := &domain.Scenario{
		Name:  "Test",
		Steps: []domain.ScenarioStep{step("Login", getReq()), step("Login", getReq())},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessageContaining(issues, "duplicate step name") {
		t.Error("expected duplicate step name error")
	}
}

func TestValidate_StepNeitherRequestNorRef(t *testing.T) {
	sc := &domain.Scenario{
		Name:  "Test",
		Steps: []domain.ScenarioStep{{Name: "Empty step"}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessage(issues, "step must have either request or requestRef") {
		t.Error("expected missing request error")
	}
}

func TestValidate_StepBothRequestAndRef(t *testing.T) {
	sc := &domain.Scenario{
		Name: "Test",
		Steps: []domain.ScenarioStep{{
			Name:       "Ambiguous",
			Request:    getReq(),
			RequestRef: "some/request.yaml",
		}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessage(issues, "step must not have both request and requestRef") {
		t.Error("expected both-set error")
	}
}

func TestValidate_UnknownAssertionType(t *testing.T) {
	sc := &domain.Scenario{
		Name: "Test",
		Steps: []domain.ScenarioStep{{
			Name:    "Check",
			Request: getReq(),
			Assertions: []domain.Assertion{
				{Type: "bogus", Operator: domain.OperatorEq, Expected: 200},
			},
		}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessageContaining(issues, `unknown assertion type "bogus"`) {
		t.Error("expected unknown type error")
	}
}

func TestValidate_UnknownAssertionOperator(t *testing.T) {
	sc := &domain.Scenario{
		Name: "Test",
		Steps: []domain.ScenarioStep{{
			Name:    "Check",
			Request: getReq(),
			Assertions: []domain.Assertion{
				{Type: domain.AssertionStatus, Operator: "badop", Expected: 200},
			},
		}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessageContaining(issues, `unknown assertion operator "badop"`) {
		t.Error("expected unknown operator error")
	}
}

func TestValidate_HeaderAssertionMissingPath(t *testing.T) {
	sc := &domain.Scenario{
		Name: "Test",
		Steps: []domain.ScenarioStep{{
			Name:    "Check",
			Request: getReq(),
			Assertions: []domain.Assertion{
				{Type: domain.AssertionHeader, Operator: domain.OperatorEq, Expected: "application/json"},
			},
		}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessageContaining(issues, "header assertion requires a path") {
		t.Error("expected missing path error")
	}
}

func TestValidate_UnknownExtractionSource(t *testing.T) {
	sc := &domain.Scenario{
		Name: "Test",
		Steps: []domain.ScenarioStep{{
			Name:    "Extract",
			Request: getReq(),
			Extractions: []domain.Extraction{
				{Name: "val", Source: "xml", Path: "//id"},
			},
		}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessageContaining(issues, `unknown extraction source "xml"`) {
		t.Error("expected unknown source error")
	}
}

func TestValidate_UnknownFailPolicy(t *testing.T) {
	sc := &domain.Scenario{
		Name: "Test",
		Steps: []domain.ScenarioStep{{
			Name:       "Step",
			Request:    getReq(),
			FailPolicy: "ignore",
		}},
	}
	issues := scenario.Validate(sc, "")
	if !hasMessageContaining(issues, `unknown failPolicy "ignore"`) {
		t.Error("expected unknown failPolicy error")
	}
}

func TestValidate_IssueString(t *testing.T) {
	fileLvl := scenario.Issue{Step: 0, Message: "scenario name is required"}
	if fileLvl.String() != "scenario name is required" {
		t.Errorf("file-level String: %q", fileLvl.String())
	}
	stepLvl := scenario.Issue{Step: 2, Name: "Login", Message: "duplicate step name (also step 1)"}
	if stepLvl.String() == "" {
		t.Error("step-level String should be non-empty")
	}
}

// helpers

func hasMessage(issues []scenario.Issue, msg string) bool {
	for _, iss := range issues {
		if iss.Message == msg {
			return true
		}
	}
	return false
}

func hasMessageContaining(issues []scenario.Issue, substr string) bool {
	for _, iss := range issues {
		if len(iss.Message) >= len(substr) {
			for i := 0; i <= len(iss.Message)-len(substr); i++ {
				if iss.Message[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
