package main

import (
	"fmt"
	"os"
	"strconv"
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

// resolveSeed decides the seed for dynamic values: --seed, else the CLRSIG_SEED environment
// variable, else a fresh random one (printed in the report whenever random values were used, so
// the run can be replayed).
func resolveSeed(cmd *cobra.Command, flag int64) (int64, error) {
	if cmd.Flags().Changed("seed") {
		return flag, nil
	}
	if v := strings.TrimSpace(os.Getenv("CLRSIG_SEED")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("CLRSIG_SEED must be a whole number, got %q", v)
		}
		return n, nil
	}
	return variables.RandomSeed(), nil
}

// prepareDynamic creates the run's engine and generates the environment's dynamic values once, so
// every scenario in the command sees the same generated environment values.
func prepareDynamic(cmd *cobra.Command, env *domain.Environment, seedFlag int64) (*variables.Engine, *domain.Environment, error) {
	seed, err := resolveSeed(cmd, seedFlag)
	if err != nil {
		return nil, nil, err
	}
	eng := variables.NewEngine(seed)
	resolved, errs := variables.ResolveEnvironment(env, eng.Derive("environment"))
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, nil, fmt.Errorf("environment: %s", strings.Join(msgs, "; "))
	}
	return eng, resolved, nil
}
