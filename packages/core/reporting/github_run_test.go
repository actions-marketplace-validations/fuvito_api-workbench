package reporting

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func githubRunFixture() *scenario.Result {
	return &scenario.Result{
		Scenario: &domain.Scenario{Title: "Users"},
		Secrets:  []string{"sk-live-ABCDEFGH12345"},
		Steps: []scenario.StepResult{
			{Label: "Skipped", Skipped: true},
			{Label: "Fetch", Error: "dial tcp: refused, token sk-live-ABCDEFGH12345"},
			{
				Label:    "Check",
				Request:  &domain.Request{Method: domain.MethodGET, URL: "http://x/users/1"},
				Response: &domain.Response{StatusCode: 200},
				Assertions: []domain.AssertionResult{
					{Assertion: domain.Assertion{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 201}, Passed: false, Message: "expected 201, got 200"},
					{Assertion: domain.Assertion{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 200}, Passed: true},
				},
				Spec: scenario.SpecCheck{State: scenario.SpecViolations, Violations: []openapi.ValidationError{{Message: `Error at "/email": property "email" is missing`}}},
			},
		},
	}
}

func TestRunGitHub_ErrorsOnTheLineOfTheStepWithTheCheckOfASpecProblem(t *testing.T) {
	out := RunGitHub([]GitHubRun{{Result: githubRunFixture(), File: "scenarios/users.yaml", StepLines: []int{4, 9, 14}}})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var errs []string
	for _, l := range lines {
		if strings.HasPrefix(l, "::error ") {
			errs = append(errs, l)
		}
	}
	if len(errs) != 3 {
		t.Fatalf("an unsent request, a failed assertion and a spec violation:\n%s", out)
	}
	if !strings.HasPrefix(errs[0], "::error file=scenarios/users.yaml,line=9,title=Users%3A Fetch::") {
		t.Errorf("the step that errored: %s", errs[0])
	}
	if !strings.HasPrefix(errs[1], "::error file=scenarios/users.yaml,line=14,title=Users%3A Check::") || !strings.Contains(errs[1], "expected 201, got 200") {
		t.Errorf("the failed assertion: %s", errs[1])
	}
	if !strings.Contains(errs[2], "title=CS-RESP-005 Users%3A Check::") {
		t.Errorf("a spec violation carries its check ID: %s", errs[2])
	}
	if strings.Contains(out, "sk-live-ABCDEFGH12345") {
		t.Errorf("a secret in an error text is masked:\n%s", out)
	}
	if !strings.Contains(out, "::notice title=Clear Signal run::1 scenarios: 0 passed, 1 failed.") {
		t.Errorf("%s", out)
	}
}

func TestRunGitHub_WithoutAFileTheAnnotationStaysOnTheRun(t *testing.T) {
	r := githubRunFixture()
	r.SetupError = "unknown function $nope"
	out := RunGitHub([]GitHubRun{{Result: r}})
	if strings.Contains(out, "file=") || !strings.Contains(out, "::error title=Users::The scenario could not start: unknown function $nope") {
		t.Errorf("%s", out)
	}
	pass := RunGitHub([]GitHubRun{{Result: &scenario.Result{Scenario: &domain.Scenario{Title: "T"}, Passed: true}}})
	if strings.Contains(pass, "::error") || !strings.Contains(pass, "1 scenarios: 1 passed, 0 failed.") {
		t.Errorf("%s", pass)
	}
}
