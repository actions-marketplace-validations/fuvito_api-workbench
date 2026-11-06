package scenario

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// Issue is a single validation finding for a scenario file.
// Step is 1-based; 0 means the issue is at file level (not tied to a step).
type Issue struct {
	Step    int
	Name    string // step name, empty for file-level issues
	Message string
}

func (iss Issue) String() string {
	if iss.Step == 0 {
		return iss.Message
	}
	if iss.Name != "" {
		return fmt.Sprintf("[step %d] %s: %s", iss.Step, iss.Name, iss.Message)
	}
	return fmt.Sprintf("[step %d]: %s", iss.Step, iss.Message)
}

// Validate checks a parsed Scenario for structural problems without executing it.
// scenarioDir is the directory containing the scenario file; pass "" to skip
// requestRef file-existence checks.
func Validate(sc *domain.Scenario, scenarioDir string) []Issue {
	var issues []Issue
	add := func(step int, name, msg string) {
		issues = append(issues, Issue{Step: step, Name: name, Message: msg})
	}

	if sc.Name == "" {
		add(0, "", "scenario name is required")
	}
	if len(sc.Steps) == 0 {
		add(0, "", "scenario has no steps")
		return issues
	}

	seenNames := make(map[string]int)

	for i, step := range sc.Steps {
		num := i + 1
		name := step.Name

		if name == "" {
			add(num, "", "step name is required")
		} else if prev, ok := seenNames[name]; ok {
			add(num, name, fmt.Sprintf("duplicate step name (also step %d)", prev))
		} else {
			seenNames[name] = num
		}

		if step.Request == nil && step.RequestRef == "" {
			add(num, name, "step must have either request or requestRef")
		}
		if step.Request != nil && step.RequestRef != "" {
			add(num, name, "step must not have both request and requestRef")
		}

		if step.RequestRef != "" && scenarioDir != "" {
			refPath := filepath.Join(scenarioDir, step.RequestRef)
			if _, err := os.Stat(refPath); err != nil {
				add(num, name, fmt.Sprintf("requestRef %q not found", step.RequestRef))
			}
		}

		if step.FailPolicy != "" &&
			step.FailPolicy != domain.FailPolicyStop &&
			step.FailPolicy != domain.FailPolicyContinue &&
			step.FailPolicy != domain.FailPolicySkip {
			add(num, name, fmt.Sprintf("unknown failPolicy %q (want: stop, continue, skip)", step.FailPolicy))
		}

		validTypes := map[domain.AssertionType]bool{
			domain.AssertionStatus: true, domain.AssertionHeader: true,
			domain.AssertionJSONPath: true, domain.AssertionSchema: true,
		}
		validOps := map[domain.Operator]bool{
			domain.OperatorEq: true, domain.OperatorNe: true,
			domain.OperatorContains: true, domain.OperatorMatches: true,
			domain.OperatorGt: true, domain.OperatorLt: true,
			"": true, // defaults to eq
		}

		for _, a := range step.Assertions {
			if !validTypes[a.Type] {
				add(num, name, fmt.Sprintf("unknown assertion type %q", a.Type))
			}
			if !validOps[a.Operator] {
				add(num, name, fmt.Sprintf("unknown assertion operator %q", a.Operator))
			}
			if (a.Type == domain.AssertionHeader || a.Type == domain.AssertionJSONPath) && a.Path == "" {
				add(num, name, fmt.Sprintf("%s assertion requires a path", a.Type))
			}
		}

		validSources := map[domain.ExtractionSource]bool{
			domain.ExtractJSONPath: true, domain.ExtractHeader: true, domain.ExtractStatus: true,
		}
		for _, e := range step.Extractions {
			if e.Name == "" {
				add(num, name, "extraction must have a name")
			}
			if !validSources[e.Source] {
				add(num, name, fmt.Sprintf("unknown extraction source %q", e.Source))
			}
			if (e.Source == domain.ExtractJSONPath || e.Source == domain.ExtractHeader) && e.Path == "" {
				add(num, name, fmt.Sprintf("%s extraction requires a path", e.Source))
			}
		}
	}

	return issues
}
