package reporting_test

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// ---- helpers ----

func makeScenario(title string) *domain.Scenario {
	return &domain.Scenario{SchemaVersion: "1", Title: title}
}

func makeStep(title string, passed bool, statusCode int, assertions []domain.AssertionResult) scenario.StepResult {
	var resp *domain.Response
	if statusCode > 0 {
		resp = &domain.Response{StatusCode: statusCode, Duration: 50 * time.Millisecond}
	}
	return scenario.StepResult{
		Title:      title,
		Response:   resp,
		Assertions: assertions,
		Passed:     passed,
	}
}

func assertion(t domain.AssertionType, path string, op domain.Operator, expected interface{}, passed bool, msg string) domain.AssertionResult {
	return domain.AssertionResult{
		Assertion: domain.Assertion{Type: t, Path: path, Operator: op, Expected: expected},
		Passed:    passed,
		Message:   msg,
	}
}

func passedResult(title string) *scenario.Result {
	return &scenario.Result{
		Scenario: makeScenario(title),
		Steps: []scenario.StepResult{
			makeStep("step one", true, 200, []domain.AssertionResult{
				assertion(domain.AssertionStatus, "", domain.OperatorEq, 200, true, ""),
			}),
		},
		Passed:   true,
		Duration: 55 * time.Millisecond,
	}
}

func failedResult(title string) *scenario.Result {
	return &scenario.Result{
		Scenario: makeScenario(title),
		Steps: []scenario.StepResult{
			makeStep("step one", false, 500, []domain.AssertionResult{
				assertion(domain.AssertionStatus, "", domain.OperatorEq, 200, false, "got 500, want 200"),
			}),
		},
		Passed:   false,
		Duration: 20 * time.Millisecond,
	}
}

func skippedResult(title string) *scenario.Result {
	sr := scenario.StepResult{Title: "skipped step", Skipped: true}
	return &scenario.Result{
		Scenario: makeScenario(title),
		Steps:    []scenario.StepResult{sr},
		Passed:   false,
		Duration: 1 * time.Millisecond,
	}
}

func errorResult(title string) *scenario.Result {
	sr := scenario.StepResult{Title: "error step", Error: "connection refused", Passed: false}
	return &scenario.Result{
		Scenario: makeScenario(title),
		Steps:    []scenario.StepResult{sr},
		Passed:   false,
		Duration: 5 * time.Millisecond,
	}
}

// ---- Text ----

func TestText_Passed(t *testing.T) {
	out := reporting.Text(passedResult("my-scenario"))
	if !strings.Contains(out, "my-scenario") {
		t.Error("missing scenario title")
	}
	if !strings.Contains(out, "PASS") {
		t.Error("missing PASS marker")
	}
	if !strings.Contains(out, "PASSED") {
		t.Error("missing PASSED verdict")
	}
	if !strings.Contains(out, "step one") {
		t.Error("missing step name")
	}
}

func TestText_Failed(t *testing.T) {
	out := reporting.Text(failedResult("my-scenario"))
	if !strings.Contains(out, "FAIL") {
		t.Error("missing FAIL marker")
	}
	if !strings.Contains(out, "FAILED") {
		t.Error("missing FAILED verdict")
	}
	if !strings.Contains(out, "got 500, want 200") {
		t.Error("missing assertion message")
	}
}

func TestText_Skipped(t *testing.T) {
	out := reporting.Text(skippedResult("skip-test"))
	if !strings.Contains(out, "SKIP") {
		t.Error("missing SKIP marker")
	}
}

func TestText_Error(t *testing.T) {
	out := reporting.Text(errorResult("err-test"))
	if !strings.Contains(out, "error: connection refused") {
		t.Error("missing error line")
	}
}

func TestText_Description(t *testing.T) {
	r := passedResult("title")
	r.Scenario.Description = "some description"
	out := reporting.Text(r)
	if !strings.Contains(out, "some description") {
		t.Error("missing description")
	}
}

func TestText_AssertionLabels(t *testing.T) {
	r := &scenario.Result{
		Scenario: makeScenario("labels"),
		Steps: []scenario.StepResult{
			{
				Title:  "s",
				Passed: true,
				Assertions: []domain.AssertionResult{
					assertion(domain.AssertionStatus, "", domain.OperatorEq, 200, true, ""),
					assertion(domain.AssertionHeader, "Content-Type", domain.OperatorContains, "json", true, ""),
					assertion(domain.AssertionJSONPath, "$.id", domain.OperatorEq, "1", true, ""),
					assertion(domain.AssertionSchema, "", "", nil, true, ""),
				},
			},
		},
		Passed:   true,
		Duration: 1 * time.Millisecond,
	}
	out := reporting.Text(r)
	if !strings.Contains(out, "status eq 200") {
		t.Error("missing status assertion label")
	}
	if !strings.Contains(out, "header Content-Type contains json") {
		t.Error("missing header assertion label")
	}
	if !strings.Contains(out, "jsonpath $.id eq 1") {
		t.Error("missing jsonpath assertion label")
	}
	if !strings.Contains(out, "schema") {
		t.Error("missing schema assertion label")
	}
}

func TestText_Extracted(t *testing.T) {
	r := &scenario.Result{
		Scenario: makeScenario("extract"),
		Steps: []scenario.StepResult{
			{Title: "s", Passed: true, Extracted: map[string]string{"token": "abc123"}},
		},
		Passed:   true,
		Duration: 1 * time.Millisecond,
	}
	out := reporting.Text(r)
	if !strings.Contains(out, `token`) || !strings.Contains(out, `abc123`) {
		t.Error("missing extracted variable")
	}
}

// ---- JSON ----

func TestJSON_Passed(t *testing.T) {
	raw, err := reporting.JSON(passedResult("json-test"))
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var rep reporting.JSONReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rep.Scenario != "json-test" {
		t.Errorf("scenario: got %q", rep.Scenario)
	}
	if !rep.Passed {
		t.Error("want passed=true")
	}
	if len(rep.Steps) != 1 {
		t.Fatalf("want 1 step, got %d", len(rep.Steps))
	}
	if rep.Steps[0].StatusCode != 200 {
		t.Errorf("statusCode: got %d", rep.Steps[0].StatusCode)
	}
}

func TestJSON_Failed(t *testing.T) {
	raw, err := reporting.JSON(failedResult("json-fail"))
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var rep reporting.JSONReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rep.Passed {
		t.Error("want passed=false")
	}
	if len(rep.Steps[0].Assertions) != 1 {
		t.Fatalf("want 1 assertion, got %d", len(rep.Steps[0].Assertions))
	}
	a := rep.Steps[0].Assertions[0]
	if a.Passed {
		t.Error("assertion should not be passed")
	}
	if a.Message == "" {
		t.Error("missing assertion message")
	}
}

func TestJSON_Skipped(t *testing.T) {
	raw, _ := reporting.JSON(skippedResult("sk"))
	var rep reporting.JSONReport
	json.Unmarshal(raw, &rep)
	if !rep.Steps[0].Skipped {
		t.Error("want skipped=true")
	}
}

func TestJSON_Error(t *testing.T) {
	raw, _ := reporting.JSON(errorResult("err"))
	var rep reporting.JSONReport
	json.Unmarshal(raw, &rep)
	if rep.Steps[0].Error != "connection refused" {
		t.Errorf("error field: got %q", rep.Steps[0].Error)
	}
}

// ---- JUnit ----

func TestJUnit_Passed(t *testing.T) {
	raw, err := reporting.JUnit(passedResult("suite"))
	if err != nil {
		t.Fatalf("JUnit: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "<testsuite") {
		t.Error("missing <testsuite>")
	}
	if !strings.Contains(s, "suite") {
		t.Error("missing suite name")
	}
}

func TestJUnit_Failed(t *testing.T) {
	raw, _ := reporting.JUnit(failedResult("suite"))
	s := string(raw)
	if !strings.Contains(s, "<failure") {
		t.Error("missing <failure> element")
	}
}

func TestJUnit_Skipped(t *testing.T) {
	raw, _ := reporting.JUnit(skippedResult("suite"))
	s := string(raw)
	if !strings.Contains(s, "<skipped") {
		t.Error("missing <skipped> element")
	}
}

func TestJUnit_ErrorStep(t *testing.T) {
	raw, _ := reporting.JUnit(errorResult("suite"))
	s := string(raw)
	if !strings.Contains(s, "<error") {
		t.Error("missing <error> element")
	}
}

func TestJUnit_ValidXML(t *testing.T) {
	raw, _ := reporting.JUnit(passedResult("suite"))
	var v interface{}
	if err := xml.Unmarshal(raw[len(xml.Header):], &v); err != nil {
		t.Errorf("JUnit output is not valid XML: %v", err)
	}
}

// ---- ConformText ----

func makeConformResult(passed bool) *conformance.ConformResult {
	ops := []conformance.OperationResult{
		{Method: "GET", Path: "/health", StatusCode: 200, OperationID: "getHealth"},
	}
	if !passed {
		ops[0].Violations = []openapi.ValidationError{{Message: "missing required field"}}
	}
	return &conformance.ConformResult{
		SpecTitle:   "test-api",
		SpecVersion: "1.0.0",
		BaseURL:     "http://localhost:8080",
		Operations:  ops,
		Duration:    30 * time.Millisecond,
	}
}

func TestConformText_Passed(t *testing.T) {
	out := reporting.ConformText(makeConformResult(true), nil)
	if !strings.Contains(out, "test-api") {
		t.Error("missing spec title")
	}
	if !strings.Contains(out, "PASS") {
		t.Error("missing PASS marker")
	}
	if !strings.Contains(out, "PASSED") {
		t.Error("missing PASSED verdict")
	}
}

func TestConformText_Failed(t *testing.T) {
	out := reporting.ConformText(makeConformResult(false), nil)
	if !strings.Contains(out, "FAIL") {
		t.Error("missing FAIL marker")
	}
	if !strings.Contains(out, "missing required field") {
		t.Error("missing violation message")
	}
}

func TestConformText_Skipped(t *testing.T) {
	r := &conformance.ConformResult{
		SpecTitle: "api", BaseURL: "http://x",
		Operations: []conformance.OperationResult{{Method: "DELETE", Path: "/x", Skipped: true}},
	}
	out := reporting.ConformText(r, nil)
	if !strings.Contains(out, "SKIP") {
		t.Error("missing SKIP marker")
	}
}

func TestConformText_WithCoverage(t *testing.T) {
	cov := &conformance.CoverageReport{
		Total: 2, Covered: 1, NotCovered: 1, Percent: 50,
		CoveredOps:    []conformance.OperationResult{{Method: "GET", Path: "/health"}},
		NotCoveredOps: []conformance.OperationResult{{Method: "POST", Path: "/users"}},
	}
	out := reporting.ConformText(makeConformResult(true), cov)
	if !strings.Contains(out, "Coverage:") {
		t.Error("missing Coverage section")
	}
	if !strings.Contains(out, "50%") {
		t.Error("missing coverage percentage")
	}
	if !strings.Contains(out, "Covered:") {
		t.Error("missing Covered section")
	}
	if !strings.Contains(out, "Not covered:") {
		t.Error("missing Not covered section")
	}
}

// ---- ConformJSON ----

func TestConformJSON_Passed(t *testing.T) {
	raw, err := reporting.ConformJSON(makeConformResult(true), nil)
	if err != nil {
		t.Fatalf("ConformJSON: %v", err)
	}
	var rep reporting.JSONConformReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rep.SpecTitle != "test-api" {
		t.Errorf("specTitle: got %q", rep.SpecTitle)
	}
	if !rep.Passed {
		t.Error("want passed=true")
	}
	if len(rep.Operations) != 1 {
		t.Fatalf("want 1 operation, got %d", len(rep.Operations))
	}
}

func TestConformJSON_WithCoverage(t *testing.T) {
	cov := &conformance.CoverageReport{
		Total: 1, Covered: 1, Percent: 100,
		CoveredOps: []conformance.OperationResult{{Method: "GET", Path: "/health"}},
	}
	raw, _ := reporting.ConformJSON(makeConformResult(true), cov)
	var rep reporting.JSONConformReport
	json.Unmarshal(raw, &rep)
	if rep.Coverage == nil {
		t.Fatal("want coverage section in JSON")
	}
	if rep.Coverage.Percent != 100 {
		t.Errorf("coverage percent: got %.0f", rep.Coverage.Percent)
	}
}

// ---- ConformJUnit ----

func TestConformJUnit_Passed(t *testing.T) {
	raw, err := reporting.ConformJUnit(makeConformResult(true))
	if err != nil {
		t.Fatalf("ConformJUnit: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "<testsuite") {
		t.Error("missing <testsuite>")
	}
	if !strings.Contains(s, "GET /health") {
		t.Error("missing operation name")
	}
}

func TestConformJUnit_Failed(t *testing.T) {
	raw, _ := reporting.ConformJUnit(makeConformResult(false))
	s := string(raw)
	if !strings.Contains(s, "<failure") {
		t.Error("missing <failure> element")
	}
}

func TestConformJUnit_SkippedOp(t *testing.T) {
	r := &conformance.ConformResult{
		SpecTitle:  "api",
		Operations: []conformance.OperationResult{{Method: "GET", Path: "/x", Skipped: true}},
	}
	raw, _ := reporting.ConformJUnit(r)
	s := string(raw)
	if !strings.Contains(s, "<skipped") {
		t.Error("missing <skipped> element")
	}
}

func TestConformJUnit_ErrorOp(t *testing.T) {
	r := &conformance.ConformResult{
		SpecTitle:  "api",
		Operations: []conformance.OperationResult{{Method: "GET", Path: "/x", Error: "timeout"}},
	}
	raw, _ := reporting.ConformJUnit(r)
	s := string(raw)
	if !strings.Contains(s, "<error") {
		t.Error("missing <error> element")
	}
}
