package scenario

import (
	"fmt"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// LintOptions say what a lint may rely on.
type LintOptions struct {
	// Env is the environment the scenario will run against. When set, a {{name}} that nothing in
	// the scenario or the environment defines is reported; without it that check is skipped, since
	// any name might come from an environment chosen later.
	Env *domain.Environment
}

// Lint finds scenarios that are valid but weak or misleading: things Validate accepts because the
// scenario runs, but that make a green run mean less than it seems. Each finding is a warning.
// Disabled steps are ignored.
func Lint(sc *domain.Scenario, o LintOptions) []Issue {
	var out []Issue
	add := func(step int, name, msg string) { out = append(out, Issue{Step: step, Name: name, Message: msg}) }

	defined := map[string]bool{}
	for _, v := range sc.Variables {
		defined[v.Name] = true
	}
	if o.Env != nil {
		for _, v := range o.Env.Variables {
			defined[v.Name] = true
		}
	}

	// where each name is used, by step index, so an extraction can be matched with a later use
	uses := make([]map[string]bool, len(sc.Steps))
	for i := range sc.Steps {
		uses[i] = map[string]bool{}
		for _, text := range StepTexts(&sc.Steps[i]) {
			for _, ref := range variables.References(text) {
				uses[i][ref] = true
			}
		}
	}

	for i := range sc.Steps {
		step := &sc.Steps[i]
		if step.Disabled {
			continue
		}
		num, label := i+1, step.Label

		if len(step.Assertions) == 0 && len(step.Extractions) == 0 {
			add(num, label, "has no assertions and extracts nothing: it passes whenever the request is answered, so a wrong status or body would go unnoticed")
		}

		if o.Env != nil {
			for _, text := range StepTexts(step) {
				for _, ref := range variables.References(text) {
					if !defined[ref] && !extractedBefore(sc, i, ref) {
						add(num, label, fmt.Sprintf("uses {{%s}}, which nothing defines: not a scenario variable, not extracted by an earlier step and not in the environment", ref))
						defined[ref] = true // say it once per scenario
					}
				}
			}
		}

		for _, e := range step.Extractions {
			if !usedLater(sc, uses, i, e.Name) {
				add(num, label, fmt.Sprintf("extracts %q, but no later step uses {{%s}}", e.Name, e.Name))
			}
		}
	}
	return out
}

func extractedBefore(sc *domain.Scenario, step int, name string) bool {
	for i := 0; i < step; i++ {
		if sc.Steps[i].Disabled {
			continue
		}
		for _, e := range sc.Steps[i].Extractions {
			if e.Name == name {
				return true
			}
		}
	}
	return false
}

func usedLater(sc *domain.Scenario, uses []map[string]bool, step int, name string) bool {
	for j := step + 1; j < len(sc.Steps); j++ {
		if !sc.Steps[j].Disabled && uses[j][name] {
			return true
		}
	}
	return false
}
