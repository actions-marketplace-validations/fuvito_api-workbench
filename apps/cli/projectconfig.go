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

// configSetting ties one key of the project file to the flag (or the positional argument) it
// supplies on the commands that read it. The table is the one place that says which key means
// what; apply and `clrsig config show` both read it, and a test keeps it equal to the key list of
// the file format.
type configSetting struct {
	key      string
	flag     string   // the flag it sets; empty for a positional argument
	commands []string // the commands that read it
	values   func(c *projectconfig.Config, path func(string) string) []string
}

func one(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func boolValue(b *bool) []string {
	if b == nil {
		return nil
	}
	return []string{strconv.FormatBool(*b)}
}

func intValue(n *int) []string {
	if n == nil {
		return nil
	}
	return []string{strconv.Itoa(*n)}
}

func joined(vs []string) []string { return one(strings.Join(vs, ",")) }

func conformValues(c *projectconfig.Config, f func(*projectconfig.Conform) []string) []string {
	if c.Conform == nil {
		return nil
	}
	return f(c.Conform)
}

func runValues(c *projectconfig.Config, f func(*projectconfig.Run) []string) []string {
	if c.Run == nil {
		return nil
	}
	return f(c.Run)
}

var configSettings = []configSetting{
	{"spec", "", []string{"conform"}, func(c *projectconfig.Config, p func(string) string) []string { return one(p(c.Spec)) }},
	{"environment", "env", []string{"conform", "run", "validate"}, func(c *projectconfig.Config, p func(string) string) []string { return one(p(c.Environment)) }},
	{"baseUrl", "base-url", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string { return one(c.BaseURL) }},
	{"conform.readOnly", "read-only", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return boolValue(x.ReadOnly) })
	}},
	{"conform.negative", "negative", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return boolValue(x.Negative) })
	}},
	{"conform.failOn", "fail-on", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return one(x.FailOn) })
	}},
	{"conform.requestTimeoutMs", "request-timeout", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return intValue(x.RequestTimeoutMs) })
	}},
	{"conform.maxResponseTimeMs", "max-response-time", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return intValue(x.MaxResponseTimeMs) })
	}},
	{"conform.maxResponseTimeFor", "max-response-time-for", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return x.MaxResponseTimeFor })
	}},
	{"conform.tags", "tags", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return joined(x.Tags) })
	}},
	{"conform.paths", "paths", []string{"conform"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return joined(x.Paths) })
	}},
	{"conform.baseline", "baseline", []string{"conform"}, func(c *projectconfig.Config, p func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return one(p(x.Baseline)) })
	}},
	{"conform.setup", "setup", []string{"conform"}, func(c *projectconfig.Config, p func(string) string) []string {
		return conformValues(c, func(x *projectconfig.Conform) []string { return one(p(x.Setup)) })
	}},
	{"run.scenarios", "", []string{"run", "validate"}, func(c *projectconfig.Config, p func(string) string) []string {
		return runValues(c, func(x *projectconfig.Run) []string {
			out := make([]string, len(x.Scenarios))
			for i, v := range x.Scenarios {
				out[i] = p(v)
			}
			return out
		})
	}},
	{"run.failFast", "fail-fast", []string{"run"}, func(c *projectconfig.Config, _ func(string) string) []string {
		return runValues(c, func(x *projectconfig.Run) []string { return boolValue(x.FailFast) })
	}},
	{"reports", "report", []string{"conform", "run"}, func(c *projectconfig.Config, p func(string) string) []string {
		out := make([]string, len(c.Reports))
		for i, r := range c.Reports {
			if format, file, ok := strings.Cut(r, "="); ok {
				r = format + "=" + p(strings.TrimSpace(file))
			}
			out[i] = r
		}
		return out
	}},
}

func (s configSetting) appliesTo(command string) bool {
	for _, c := range s.commands {
		if c == command {
			return true
		}
	}
	return false
}

// loadedConfig is the project file a command found, and what it took from it.
type loadedConfig struct {
	cfg  *projectconfig.Config
	used []string
	cwd  string
}

// findProjectConfig reads --config and --no-config, then loads the file they point at or the one
// found from the current folder. It returns nil when there is none, or when it was switched off.
func findProjectConfig(cmd *cobra.Command) (cfg *projectconfig.Config, ignored bool, cwd string, err error) {
	cfgFlag, noFlag := cmd.Flag("config"), cmd.Flag("no-config")
	if cfgFlag == nil || noFlag == nil {
		return nil, true, "", nil
	}
	explicit := cfgFlag.Value.String()
	if noFlag.Value.String() == "true" {
		if explicit != "" {
			return nil, false, "", fmt.Errorf("--config and --no-config cannot be used together")
		}
		return nil, true, "", nil
	}
	if cwd, err = os.Getwd(); err != nil {
		return nil, false, "", err
	}
	path := explicit
	if path == "" {
		home, _ := os.UserHomeDir()
		if path, err = projectconfig.Discover(cwd, home); err != nil {
			return nil, false, cwd, err
		}
		if path == "" {
			return nil, false, cwd, nil
		}
	}
	cfg, err = projectconfig.Load(path)
	if err != nil {
		return nil, false, cwd, fmt.Errorf("project file: %w", err)
	}
	return cfg, false, cwd, nil
}

// loadAndApply fills in what the project file says for the flags and arguments the user did not
// give, and returns the arguments to use. A command built without the root flags (a test) is left
// alone.
func loadAndApply(cmd *cobra.Command, args []string) ([]string, *loadedConfig, error) {
	cfg, _, cwd, err := findProjectConfig(cmd)
	if err != nil || cfg == nil {
		return args, nil, err
	}
	lc := &loadedConfig{cfg: cfg, cwd: cwd}
	path := func(v string) string { return cfg.ForCommandLine(v, cwd) }
	name := cmd.Name()

	for _, s := range configSettings {
		if !s.appliesTo(name) {
			continue
		}
		values := s.values(cfg, path)
		if len(values) == 0 {
			continue
		}
		if s.flag == "" { // a positional argument: only when none was given
			if len(args) == 0 {
				args = values
				lc.used = append(lc.used, s.key)
			}
			continue
		}
		f := cmd.Flags().Lookup(s.flag)
		if f == nil || f.Changed {
			continue
		}
		for _, v := range values {
			if err := cmd.Flags().Set(s.flag, v); err != nil {
				return nil, nil, fmt.Errorf("project file: %s: %w", s.key, err)
			}
		}
		lc.used = append(lc.used, s.key)
	}
	return args, lc, nil
}

// applyProjectConfig is loadAndApply for a command that is about to run: it says on stderr which
// file it used and for what, so a value never comes from nowhere, and reports a missing argument
// with the place to set it.
func applyProjectConfig(cmd *cobra.Command, args []string) ([]string, error) {
	args, lc, err := loadAndApply(cmd, args)
	if err != nil {
		return nil, err
	}
	if lc == nil {
		return args, nil
	}
	switch cmd.Name() {
	case "conform":
		if len(args) == 0 {
			return nil, fmt.Errorf("conform needs a spec: pass one, or set spec: in %s", displayPath(lc.cfg.Path(), lc.cwd))
		}
	case "run", "validate":
		if len(args) == 0 {
			return nil, fmt.Errorf("%s needs a scenario file or folder: pass one, or set run.scenarios in %s", cmd.Name(), displayPath(lc.cfg.Path(), lc.cwd))
		}
	}
	if len(lc.used) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Using %s (%s from the file)\n", displayPath(lc.cfg.Path(), lc.cwd), strings.Join(lc.used, ", "))
	}
	return args, nil
}

func displayPath(path, cwd string) string {
	if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}
