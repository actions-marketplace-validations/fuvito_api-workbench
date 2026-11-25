package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// runConformSetup runs the --setup scenario and returns the environment conform should use: the
// given one plus what the scenario extracted. The scenario runs against the same environment; a
// --base-url fills {{baseUrl}} for it when the environment does not define one. A failed setup
// stops the command before the check starts, with the setup's own report.
func runConformSetup(cmd *cobra.Command, exec *executor.Executor, file string, env *domain.Environment, baseURL string, vars []string) (*domain.Environment, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading the --setup scenario: %w", err)
	}
	sc, err := serialization.ParseScenarioFile(data, file)
	if err != nil {
		return nil, fmt.Errorf("parsing the --setup scenario %s: %w", file, err)
	}

	if issues := scenario.Validate(sc, filepath.Dir(file)); len(issues) > 0 {
		msgs := make([]string, len(issues))
		for i, iss := range issues {
			msgs[i] = iss.String()
		}
		return nil, fmt.Errorf("the --setup scenario %s is not valid: %s", file, strings.Join(msgs, "; "))
	}

	setupEnv := env
	if baseURL != "" && !hasVariable(env, "baseUrl") {
		withBase := &domain.Environment{SchemaVersion: domain.CurrentSchemaVersion, Name: "command line"}
		if env != nil {
			c := *env
			c.Variables = append([]domain.EnvironmentVariable(nil), env.Variables...)
			withBase = &c
		}
		withBase.Variables = append(withBase.Variables, domain.EnvironmentVariable{Name: "baseUrl", Value: baseURL})
		setupEnv = withBase
	}

	keep := make([]string, 0, len(vars))
	for _, v := range vars {
		name, _, _ := strings.Cut(v, "=")
		keep = append(keep, name)
	}
	dir := filepath.Dir(file)
	out := application.RunSetup(context.Background(), application.SetupInput{
		Exec: exec, Scenario: sc, Env: setupEnv, Keep: keep,
		RefDirs: []string{dir, filepath.Join(dir, "..", "requests")},
	})
	for _, ref := range out.Unresolved {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: requestRef %q in the --setup scenario could not be loaded\n", ref)
	}
	if !out.Passed {
		fmt.Fprint(cmd.ErrOrStderr(), reporting.Text(out.Result))
		return nil, fmt.Errorf("the --setup scenario %s did not pass, so the check was not started: fix the scenario (the report above says which step and why) or run without --setup", file)
	}
	if len(out.Names) == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Setup scenario %s passed; it extracted nothing, so the check has no new values.\n", filepath.ToSlash(file))
		return env, nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Setup scenario %s passed; values available to the check: %s\n", filepath.ToSlash(file), strings.Join(out.Names, ", "))
	// the extracted values go into the environment conform uses; the {{baseUrl}} added for the
	// setup alone stays out of it
	if setupEnv != env {
		return withoutAddedBaseURL(out.Env, env), nil
	}
	return out.Env, nil
}

func hasVariable(env *domain.Environment, name string) bool {
	if env == nil {
		return false
	}
	for _, v := range env.Variables {
		if v.Name == name {
			return true
		}
	}
	return false
}

// withoutAddedBaseURL removes the baseUrl variable that was added only for the setup run.
func withoutAddedBaseURL(merged, original *domain.Environment) *domain.Environment {
	out := *merged
	out.Variables = nil
	for _, v := range merged.Variables {
		if v.Name == "baseUrl" && !hasVariable(original, "baseUrl") {
			continue
		}
		out.Variables = append(out.Variables, v)
	}
	return &out
}
