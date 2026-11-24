package application

import (
	"context"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// SetupInput describes the scenario that runs before a conformance run, usually a login: whatever
// its steps extract becomes available to the generated requests as {{name}}.
type SetupInput struct {
	Exec     *executor.Executor
	Scenario *domain.Scenario
	// Env is the environment of the conformance run; the setup scenario runs against it.
	Env *domain.Environment
	// RefDirs are searched, in order, for the request files of steps that use requestRef.
	RefDirs []string
	// Keep names values the user gave explicitly (--var): an extracted value of the same name does
	// not replace them.
	Keep []string
}

// SetupOutput is what the setup scenario produced.
type SetupOutput struct {
	Result *scenario.Result
	// Passed is true when every step passed and the scenario could start.
	Passed bool
	// Env is the environment the conformance run should use: Env plus the extracted values. A
	// value whose name looks like a credential (token, password, api key, ...) is marked secret,
	// so reports mask it.
	Env *domain.Environment
	// Names lists the extracted values added to the environment, in the order they first appeared.
	// The values themselves are never listed.
	Names []string
	// Unresolved are request files the scenario names that could not be found.
	Unresolved []string
}

// RunSetup runs the setup scenario and collects what it extracted. When it fails, Env is the
// input environment unchanged: a conformance run must not start on half a login.
func RunSetup(ctx context.Context, in SetupInput) SetupOutput {
	out := SetupOutput{Env: in.Env}
	out.Unresolved = scenario.ResolveRequestRefs(in.Scenario, in.RefDirs...)

	runner := scenario.New(in.Exec).WithEngine(variables.NewEngine(variables.RandomSeed()), "setup")
	out.Result = runner.Run(ctx, in.Scenario, in.Env)
	out.Passed = out.Result.Passed && out.Result.SetupError == ""
	if !out.Passed {
		return out
	}

	keep := map[string]bool{}
	for _, n := range in.Keep {
		keep[strings.TrimSpace(n)] = true
	}
	values := map[string]string{}
	var order []string
	for _, step := range out.Result.Steps {
		for name, value := range step.Extracted {
			if keep[name] {
				continue
			}
			if _, seen := values[name]; !seen {
				order = append(order, name)
			}
			values[name] = value
		}
	}
	if len(order) == 0 {
		return out
	}
	sort.Strings(order)
	pairs := make([]string, len(order))
	for i, name := range order {
		pairs[i] = name + "=" + values[name]
	}
	merged, err := variables.WithOverrides(in.Env, pairs)
	if err != nil {
		out.Passed = false
		return out
	}
	for i := range merged.Variables {
		v := &merged.Variables[i]
		if _, added := values[v.Name]; added && reporting.IsSecretName(v.Name) {
			v.Secret = true
		}
	}
	out.Env, out.Names = merged, order
	return out
}
