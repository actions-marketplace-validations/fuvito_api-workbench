package scenario_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

func parse(t *testing.T, text string) *domain.Scenario {
	t.Helper()
	sc, err := serialization.ParseScenario([]byte(text), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func messages(issues []scenario.Issue) string {
	var out []string
	for _, i := range issues {
		out = append(out, i.String())
	}
	return strings.Join(out, "\n")
}

const healthy = `schemaVersion: "1"
title: Healthy
variables:
  - name: user
    value: ana
steps:
  - label: Login
    request: {method: POST, url: "{{baseUrl}}/login"}
    extractions:
      - {name: token, source: jsonpath, path: $.token}
  - label: Me
    request:
      method: GET
      url: "{{baseUrl}}/users/{{user}}"
      headers:
        - {name: Authorization, value: "Bearer {{token}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
`

func TestLint_AHealthyScenarioHasNothingToSay(t *testing.T) {
	sc := parse(t, healthy)
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "http://x"}}}
	if got := scenario.Lint(sc, scenario.LintOptions{Env: env}); len(got) != 0 {
		t.Fatalf("%s", messages(got))
	}
	if got := scenario.Lint(sc, scenario.LintOptions{}); len(got) != 0 {
		t.Fatalf("without an environment the same scenario is clean: %s", messages(got))
	}
}

func TestLint_StepsThatCheckNothing(t *testing.T) {
	sc := parse(t, `schemaVersion: "1"
title: Weak
steps:
  - label: Ping
    request: {method: GET, url: "http://x/ping"}
  - label: Login
    request: {method: POST, url: "http://x/login"}
    extractions:
      - {name: token, source: jsonpath, path: $.token}
  - label: Use
    request: {method: GET, url: "http://x/{{token}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
  - label: Off
    disabled: true
    request: {method: GET, url: "http://x/off"}
`)
	got := messages(scenario.Lint(sc, scenario.LintOptions{}))
	if !strings.Contains(got, "[step 1] Ping: has no assertions and extracts nothing") {
		t.Errorf("a step that checks nothing is named:\n%s", got)
	}
	if strings.Contains(got, "Login") || strings.Contains(got, "Off") {
		t.Errorf("a step whose extraction is its check, and a disabled step, are not flagged:\n%s", got)
	}
}

func TestLint_ExtractionsNobodyUses(t *testing.T) {
	sc := parse(t, `schemaVersion: "1"
title: Unused
steps:
  - label: Login
    request: {method: POST, url: "http://x/login"}
    extractions:
      - {name: token, source: jsonpath, path: $.token}
      - {name: spare, source: jsonpath, path: $.spare}
  - label: Use
    request:
      method: GET
      url: "http://x/me"
      headers:
        - {name: Authorization, value: "Bearer {{token}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
`)
	got := messages(scenario.Lint(sc, scenario.LintOptions{}))
	if !strings.Contains(got, `extracts "spare", but no later step uses {{spare}}`) || strings.Contains(got, `"token"`) {
		t.Errorf("only the unused extraction is reported:\n%s", got)
	}
}

func TestLint_NamesNothingDefinesNeedAnEnvironment(t *testing.T) {
	sc := parse(t, `schemaVersion: "1"
title: Undefined
steps:
  - label: One
    request: {method: GET, url: "{{baseUrl}}/a/{{missingId}}/{{missingId}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
  - label: Two
    request: {method: GET, url: "{{baseUrl}}/b/{{$guid}}/{{later}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
    extractions:
      - {name: later, source: jsonpath, path: $.id}
  - label: Three
    request: {method: GET, url: "{{baseUrl}}/c/{{later}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
`)
	if got := scenario.Lint(sc, scenario.LintOptions{}); len(got) != 0 {
		t.Fatalf("with no environment this rule is skipped: %s", messages(got))
	}
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "http://x"}}}
	got := messages(scenario.Lint(sc, scenario.LintOptions{Env: env}))
	if strings.Count(got, "{{missingId}}") != 1 {
		t.Errorf("a name nothing defines is reported once:\n%s", got)
	}
	if !strings.Contains(got, "[step 2] Two: uses {{later}}") {
		t.Errorf("a name extracted by the same step, or a later one, is not defined yet:\n%s", got)
	}
	for _, ok := range []string{"{{baseUrl}}", "{{$guid}}"} {
		if strings.Contains(got, ok+",") {
			t.Errorf("%s is defined or a function:\n%s", ok, got)
		}
	}
	if strings.Contains(got, "Three") {
		t.Errorf("a name extracted by an earlier step is defined:\n%s", got)
	}
}
