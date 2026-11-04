package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/fuvito/api-workbench/packages/core/assertions"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// StepResult holds the outcome of a single scenario step.
type StepResult struct {
	Name        string
	Request     *domain.Request
	Response    *domain.Response
	Assertions  []domain.AssertionResult
	Extracted   map[string]string // variables extracted from this step's response
	Passed      bool
	Skipped     bool
	Error       string
}

// Result holds the full outcome of running a scenario.
type Result struct {
	Scenario   *domain.Scenario
	Steps      []StepResult
	Passed     bool
	StartedAt  time.Time
	Duration   time.Duration
}

// Runner executes scenarios against the HTTP executor.
type Runner struct {
	exec     *executor.Executor
	asserter *assertions.Evaluator
}

func New(exec *executor.Executor) *Runner {
	return &Runner{exec: exec, asserter: assertions.New()}
}

// Run executes all steps of the scenario sequentially.
// env provides the initial variable set; variables extracted in each step
// are made available to all subsequent steps.
func (r *Runner) Run(ctx context.Context, s *domain.Scenario, env *domain.Environment) *Result {
	result := &Result{
		Scenario:  s,
		StartedAt: time.Now(),
		Passed:    true,
	}

	resolver := variables.NewResolver(env)
	prevFailed := false

	for i := range s.Steps {
		step := &s.Steps[i]
		sr := r.runStep(ctx, step, resolver, prevFailed)
		result.Steps = append(result.Steps, sr)

		if !sr.Passed && !sr.Skipped {
			result.Passed = false
			prevFailed = true
			if step.FailPolicy == domain.FailPolicyStop || step.FailPolicy == "" {
				// mark remaining steps as skipped
				for j := i + 1; j < len(s.Steps); j++ {
					result.Steps = append(result.Steps, StepResult{
						Name:    s.Steps[j].Name,
						Skipped: true,
					})
				}
				break
			}
		} else {
			prevFailed = false
		}
	}

	result.Duration = time.Since(result.StartedAt)
	return result
}

func (r *Runner) runStep(ctx context.Context, step *domain.ScenarioStep, resolver *variables.Resolver, prevFailed bool) StepResult {
	sr := StepResult{Name: step.Name}

	if prevFailed && step.FailPolicy == domain.FailPolicySkip {
		sr.Skipped = true
		return sr
	}

	if step.Request == nil {
		sr.Error = "step has no request (use inline request or resolve requestRef before running)"
		return sr
	}

	resolved := resolver.InterpolateRequest(step.Request)
	sr.Request = resolved

	resp, err := r.exec.Execute(ctx, resolved)
	if err != nil {
		sr.Error = err.Error()
		return sr
	}
	sr.Response = resp

	// Evaluate assertions
	sr.Assertions = r.asserter.EvaluateAll(step.Assertions, resp)
	sr.Passed = true
	for _, ar := range sr.Assertions {
		if !ar.Passed {
			sr.Passed = false
		}
	}

	// Extract variables (regardless of assertion outcome)
	extracted := make(map[string]string)
	for _, ex := range step.Extractions {
		val, err := extractValue(ex, resp)
		if err != nil {
			sr.Error = fmt.Sprintf("extraction %q: %v", ex.Name, err)
			continue
		}
		extracted[ex.Name] = val
		resolver.Set(ex.Name, val)
	}
	sr.Extracted = extracted

	return sr
}

func extractValue(ex domain.Extraction, resp *domain.Response) (string, error) {
	switch ex.Source {
	case domain.ExtractStatus:
		return fmt.Sprintf("%d", resp.StatusCode), nil

	case domain.ExtractHeader:
		name := strings.ToLower(ex.Path)
		for _, h := range resp.Headers {
			if strings.ToLower(h.Name) == name {
				return h.Value, nil
			}
		}
		return "", fmt.Errorf("header %q not found in response", ex.Path)

	case domain.ExtractJSONPath:
		body := resp.BodyString()
		if body == "" {
			return "", fmt.Errorf("response body is empty")
		}
		var data interface{}
		if err := json.Unmarshal([]byte(body), &data); err != nil {
			return "", fmt.Errorf("response body is not valid JSON: %w", err)
		}
		val, err := jsonpath.Get(ex.Path, data)
		if err != nil {
			return "", fmt.Errorf("jsonpath %q: %w", ex.Path, err)
		}
		return fmt.Sprintf("%v", val), nil

	default:
		return "", fmt.Errorf("unknown extraction source %q", ex.Source)
	}
}

// AllPassed reports whether every non-skipped step passed all assertions.
func (res *Result) AllPassed() bool {
	return res.Passed
}

// Counts returns (passed, failed, skipped) step counts.
func (res *Result) Counts() (passed, failed, skipped int) {
	for _, s := range res.Steps {
		switch {
		case s.Skipped:
			skipped++
		case s.Passed && s.Error == "":
			passed++
		default:
			failed++
		}
	}
	return
}

// AssertionCounts returns total (passed, failed) assertion counts across all steps.
func (res *Result) AssertionCounts() (passed, failed int) {
	for _, s := range res.Steps {
		for _, a := range s.Assertions {
			if a.Passed {
				passed++
			} else {
				failed++
			}
		}
	}
	return
}
