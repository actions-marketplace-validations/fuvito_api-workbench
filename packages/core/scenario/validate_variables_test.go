package scenario

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func scenarioWith(vars []domain.EnvironmentVariable, req *domain.Request, asserts ...domain.Assertion) *domain.Scenario {
	if req == nil {
		req = &domain.Request{Method: domain.MethodGET, URL: "http://x/ping"}
	}
	return &domain.Scenario{
		Title:     "S",
		Variables: vars,
		Steps:     []domain.ScenarioStep{{Label: "call", Request: req, Assertions: asserts}},
	}
}

func messages(issues []Issue) string {
	var out []string
	for _, i := range issues {
		out = append(out, i.String())
	}
	return strings.Join(out, "\n")
}

func TestValidate_ValidVariablesAndFunctions(t *testing.T) {
	sc := scenarioWith(
		[]domain.EnvironmentVariable{
			{Name: "email", Value: "u-{{$randomString 8}}@x.com"},
			{Name: "label", Value: "mail-{{email}}-{{baseUrl}}"}, // earlier scenario variable and an environment variable
			{Name: "due", Value: "{{$date +7d format=yyyy-MM-dd}}"},
		},
		&domain.Request{Method: domain.MethodPOST, URL: "{{baseUrl}}/orders/{{$guid}}",
			Headers: []domain.Header{{Name: "X-Id", Value: "{{$timestamp}}"}},
			Body:    &domain.RequestBody{Type: domain.BodyJSON, Content: `{"n":{{$randomInt 1 5}}}`}},
		domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.email", Operator: domain.OperatorEq, Expected: "{{email}}"},
	)
	if issues := Validate(sc, ""); len(issues) != 0 {
		t.Errorf("a valid scenario has no issues:\n%s", messages(issues))
	}
}

func TestValidate_VariableProblems(t *testing.T) {
	cases := []struct {
		name string
		vars []domain.EnvironmentVariable
		want string
	}{
		{"missing name", []domain.EnvironmentVariable{{Name: "", Value: "x"}}, "name is required"},
		{"bad name", []domain.EnvironmentVariable{{Name: "my var", Value: "x"}}, "must not contain spaces, braces or $"},
		{"dollar in name", []domain.EnvironmentVariable{{Name: "$x", Value: "x"}}, "must not contain spaces, braces or $"},
		{"duplicate", []domain.EnvironmentVariable{{Name: "a", Value: "1"}, {Name: "a", Value: "2"}}, `"a" is defined more than once`},
		{"unknown function", []domain.EnvironmentVariable{{Name: "a", Value: "{{$randomIntt 1 2}}"}}, "unknown function $randomIntt"},
		{"bad arguments", []domain.EnvironmentVariable{{Name: "a", Value: "{{$randomInt 5}}"}}, "give both a minimum and a maximum"},
		{"bad offset", []domain.EnvironmentVariable{{Name: "a", Value: "{{$date +7x}}"}}, "is not an offset"},
		{"self reference", []domain.EnvironmentVariable{{Name: "a", Value: "{{a}}"}}, `"a" refers to itself`},
		{"used before defined", []domain.EnvironmentVariable{{Name: "a", Value: "x-{{b}}"}, {Name: "b", Value: "1"}}, "defined after it; define b first"},
	}
	for _, c := range cases {
		issues := Validate(scenarioWith(c.vars, nil), "")
		if !strings.Contains(messages(issues), c.want) {
			t.Errorf("%s: want an issue containing %q, got:\n%s", c.name, c.want, messages(issues))
		}
	}
}

func TestValidate_FunctionErrorsInStepsAreLocated(t *testing.T) {
	sc := scenarioWith(nil,
		&domain.Request{Method: domain.MethodGET, URL: "http://x/{{$guidd}}",
			Headers:     []domain.Header{{Name: "X", Value: "{{$randomString}}"}},
			QueryParams: []domain.QueryParam{{Name: "d", Value: "{{$date tz=Mars/Base}}"}},
			Body:        &domain.RequestBody{Type: domain.BodyRaw, Content: "{{$randomFloat 5 1}}"},
			Auth:        &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "{{$nope}}"}}},
		domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.x", Operator: domain.OperatorEq, Expected: "{{$randomInt 9 1}}"},
	)
	issues := Validate(sc, "")
	text := messages(issues)
	for _, want := range []string{"unknown function $guidd", "wrong number of arguments", "unknown time zone", "must not be larger", "unknown function $nope", "larger than the maximum"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	for _, i := range issues {
		if i.Step != 1 || i.Name != "call" {
			t.Errorf("an error in a step should point at the step: %+v", i)
		}
	}
}

func TestValidate_PlainUnknownReferencesAreStillAllowed(t *testing.T) {
	sc := scenarioWith(nil, &domain.Request{Method: domain.MethodGET, URL: "{{baseUrl}}/users/{{userId}}"})
	if issues := Validate(sc, ""); len(issues) != 0 {
		t.Errorf("plain references may come from an environment or an extraction: %s", messages(issues))
	}
}
