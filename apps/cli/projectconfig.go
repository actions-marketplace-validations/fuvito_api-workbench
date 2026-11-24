package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/projectconfig"
)

// addConfigFlags registers --config and --no-config on the root command.
func addConfigFlags(root *cobra.Command) {
	f := root.PersistentFlags()
	f.String("config", "", "Project file to use instead of the clrsig.yaml found in this folder or above it")
	f.Bool("no-config", false, "Ignore any project file (clrsig.yaml)")
}

// applyProjectConfig fills in what the project file says for the flags and arguments the user did
// not give, and returns the arguments to use. Flags and arguments given on the command line always
// win. It says on stderr which file it used and for what, so a value never comes from nowhere.
// A command that was built without the root flags (a test) is left alone.
func applyProjectConfig(cmd *cobra.Command, args []string) ([]string, error) {
	cfgFlag, noFlag := cmd.Flag("config"), cmd.Flag("no-config")
	if cfgFlag == nil || noFlag == nil {
		return args, nil
	}
	explicit := cfgFlag.Value.String()
	if noFlag.Value.String() == "true" {
		if explicit != "" {
			return nil, fmt.Errorf("--config and --no-config cannot be used together")
		}
		return args, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := explicit
	if path == "" {
		home, _ := os.UserHomeDir()
		if path, err = projectconfig.Discover(cwd, home); err != nil {
			return nil, err
		}
		if path == "" {
			return args, nil
		}
	}
	cfg, err := projectconfig.Load(path)
	if err != nil {
		return nil, fmt.Errorf("project file: %w", err)
	}

	var used []string
	setFlag := func(key, flag string, values ...string) error {
		f := cmd.Flags().Lookup(flag)
		if f == nil || f.Changed || len(values) == 0 {
			return nil
		}
		for _, v := range values {
			if err := cmd.Flags().Set(flag, v); err != nil {
				return fmt.Errorf("project file: %s: %w", key, err)
			}
		}
		used = append(used, key)
		return nil
	}
	pathOf := func(v string) string { return cfg.ForCommandLine(v, cwd) }
	pathsOf := func(vs []string) []string {
		out := make([]string, len(vs))
		for i, v := range vs {
			out[i] = pathOf(v)
		}
		return out
	}
	reports := make([]string, len(cfg.Reports))
	for i, r := range cfg.Reports {
		if format, file, ok := strings.Cut(r, "="); ok {
			r = format + "=" + pathOf(strings.TrimSpace(file))
		}
		reports[i] = r
	}
	boolStr := func(b *bool) []string {
		if b == nil {
			return nil
		}
		return []string{strconv.FormatBool(*b)}
	}
	intStr := func(n *int) []string {
		if n == nil {
			return nil
		}
		return []string{strconv.Itoa(*n)}
	}
	orNone := func(v string) []string {
		if v == "" {
			return nil
		}
		return []string{v}
	}

	switch cmd.Name() {
	case "conform":
		if len(args) == 0 && cfg.Spec != "" {
			args = []string{pathOf(cfg.Spec)}
			used = append(used, "spec")
		}
		steps := []struct {
			key, flag string
			values    []string
		}{
			{"environment", "env", orNone(pathOf(cfg.Environment))},
			{"baseUrl", "base-url", orNone(cfg.BaseURL)},
			{"reports", "report", reports},
		}
		if c := cfg.Conform; c != nil {
			steps = append(steps, []struct {
				key, flag string
				values    []string
			}{
				{"conform.readOnly", "read-only", boolStr(c.ReadOnly)},
				{"conform.negative", "negative", boolStr(c.Negative)},
				{"conform.failOn", "fail-on", orNone(c.FailOn)},
				{"conform.requestTimeoutMs", "request-timeout", intStr(c.RequestTimeoutMs)},
				{"conform.maxResponseTimeMs", "max-response-time", intStr(c.MaxResponseTimeMs)},
				{"conform.maxResponseTimeFor", "max-response-time-for", c.MaxResponseTimeFor},
				{"conform.tags", "tags", orNone(strings.Join(c.Tags, ","))},
				{"conform.paths", "paths", orNone(strings.Join(c.Paths, ","))},
				{"conform.baseline", "baseline", orNone(pathOf(c.Baseline))},
			}...)
		}
		for _, s := range steps {
			if err := setFlag(s.key, s.flag, s.values...); err != nil {
				return nil, err
			}
		}
		if len(args) == 0 {
			return nil, fmt.Errorf("conform needs a spec: pass one, or set spec: in %s", displayPath(cfg.Path(), cwd))
		}
	case "run", "validate":
		if len(args) == 0 && cfg.Run != nil && len(cfg.Run.Scenarios) > 0 {
			args = pathsOf(cfg.Run.Scenarios)
			used = append(used, "run.scenarios")
		}
		if cmd.Name() == "run" {
			if err := setFlag("environment", "env", orNone(pathOf(cfg.Environment))...); err != nil {
				return nil, err
			}
			if cfg.Run != nil {
				if err := setFlag("run.failFast", "fail-fast", boolStr(cfg.Run.FailFast)...); err != nil {
					return nil, err
				}
			}
			if err := setFlag("reports", "report", reports...); err != nil {
				return nil, err
			}
		}
		if len(args) == 0 {
			return nil, fmt.Errorf("%s needs a scenario file or folder: pass one, or set run.scenarios in %s", cmd.Name(), displayPath(cfg.Path(), cwd))
		}
	default:
		return args, nil
	}

	if len(used) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Using %s (%s from the file)\n", displayPath(cfg.Path(), cwd), strings.Join(used, ", "))
	}
	return args, nil
}

func displayPath(path, cwd string) string {
	if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}
