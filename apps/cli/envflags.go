package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// finishEnvironment applies --var overrides to env (without modifying it) and warns, once per
// variable, about ${NAME} references whose process environment variable is not set. A missing
// variable is a warning, not an error: an unused one should not fail a run.
func finishEnvironment(cmd *cobra.Command, env *domain.Environment, vars []string) (*domain.Environment, error) {
	if len(vars) > 0 {
		overridden, err := applyVars(env, vars)
		if err != nil {
			return nil, err
		}
		env = overridden
	}
	for _, ref := range variables.MissingEnv(env) {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: environment variable %s is not set (the environment variable %q refers to it as ${%s})\n", ref.Name, ref.Var, ref.Name)
	}
	return env, nil
}

// applyVars returns a copy of env with each name=value overriding or adding a variable. An
// override keeps the variable's secret flag, so a secret stays redacted. A nil env is allowed.
func applyVars(env *domain.Environment, vars []string) (*domain.Environment, error) {
	out := &domain.Environment{SchemaVersion: domain.CurrentSchemaVersion, Name: "command line"}
	if env != nil {
		copied := *env
		copied.Variables = append([]domain.EnvironmentVariable(nil), env.Variables...)
		out = &copied
	}
	for _, raw := range vars {
		name, value, ok := strings.Cut(raw, "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			return nil, fmt.Errorf("--var %q must look like name=value", raw)
		}
		replaced := false
		for i := range out.Variables {
			if out.Variables[i].Name == name {
				out.Variables[i].Value = value
				replaced = true
				break
			}
		}
		if !replaced {
			out.Variables = append(out.Variables, domain.EnvironmentVariable{Name: name, Value: value})
		}
	}
	return out, nil
}
