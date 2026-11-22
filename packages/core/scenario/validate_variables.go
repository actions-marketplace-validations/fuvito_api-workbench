package scenario

import (
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// validateVariables checks the scenario's own variables: names, duplicates, every {{$function}}
// call, and that a variable only refers to scenario variables defined before it (they are
// evaluated in order).
func validateVariables(sc *domain.Scenario, add func(step int, name, msg string)) {
	seen := map[string]bool{}
	defined := map[string]int{}
	for i, v := range sc.Variables {
		defined[v.Name] = i
	}
	for i, v := range sc.Variables {
		if strings.TrimSpace(v.Name) == "" {
			add(0, "", fmt.Sprintf("variables[%d]: name is required", i))
			continue
		}
		if strings.ContainsAny(v.Name, "{}$ \t") {
			add(0, "", fmt.Sprintf("variable %q: a name must not contain spaces, braces or $", v.Name))
		}
		if seen[v.Name] {
			add(0, "", fmt.Sprintf("variable %q is defined more than once", v.Name))
		}
		seen[v.Name] = true

		for _, err := range variables.Check(v.Value) {
			add(0, "", fmt.Sprintf("variable %q: %v", v.Name, err))
		}
		for _, ref := range variables.References(v.Value) {
			if ref == v.Name {
				add(0, "", fmt.Sprintf("variable %q refers to itself", v.Name))
			} else if j, ok := defined[ref]; ok && j > i {
				add(0, "", fmt.Sprintf("variable %q uses {{%s}}, which is defined after it; define %s first", v.Name, ref, ref))
			}
		}
	}
}

// stepStrings returns every text in a step that may contain {{variables}} or {{$functions}}.
// StepTexts returns every piece of a step that is sent or compared and may hold a {{placeholder}}:
// the request's URL, headers, query, body and auth, and the assertions. Notes and labels are not
// included, because they are never sent.
func StepTexts(step *domain.ScenarioStep) []string { return stepStrings(step) }

func stepStrings(step *domain.ScenarioStep) []string {
	var out []string
	if r := step.Request; r != nil {
		out = append(out, r.URL)
		for _, h := range r.Headers {
			out = append(out, h.Name, h.Value)
		}
		for _, q := range r.QueryParams {
			out = append(out, q.Name, q.Value)
		}
		if r.Body != nil {
			out = append(out, r.Body.Content)
		}
		if a := r.Auth; a != nil {
			if a.Bearer != nil {
				out = append(out, a.Bearer.Token)
			}
			if a.Basic != nil {
				out = append(out, a.Basic.Username, a.Basic.Password)
			}
			if a.APIKey != nil {
				out = append(out, a.APIKey.Key, a.APIKey.Value)
			}
		}
	}
	for _, a := range step.Assertions {
		out = append(out, a.Name, a.Path)
		if s, ok := a.Expected.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// CheckDynamic reports only the problems with variables and {{$function}} calls: the checks that
// can be done before a run starts and that would otherwise make a scenario fail or send a wrong
// request. Issues with Step 0 are about the scenario's own variables (a duplicate, a cycle, a bad
// function in a value): the scenario cannot start. Issues with a step number are about one step;
// that step fails when it is reached, and clrsig run does not stop the others for it.
func CheckDynamic(sc *domain.Scenario) []Issue {
	var issues []Issue
	add := func(step int, name, msg string) {
		issues = append(issues, Issue{Step: step, Name: name, Message: msg})
	}
	validateVariables(sc, add)
	for i := range sc.Steps {
		for _, text := range stepStrings(&sc.Steps[i]) {
			for _, err := range variables.Check(text) {
				add(i+1, sc.Steps[i].Label, err.Error())
			}
		}
	}
	return issues
}

// ScenarioLevel returns the issues that stop a scenario from starting: the ones about its own
// variables, not about one step.
func ScenarioLevel(issues []Issue) []Issue {
	var out []Issue
	for _, i := range issues {
		if i.Step == 0 {
			out = append(out, i)
		}
	}
	return out
}
