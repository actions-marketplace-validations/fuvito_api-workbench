package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/bundle"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/telemetry"
	"github.com/spf13/cobra"
)

// specLoader loads OpenAPI specs (file paths or http(s) URLs) once per source and
// reports problems as warnings, so a bad spec never stops a run.
type specLoader struct {
	cmd   *cobra.Command
	cache map[string]*openapi.Spec
}

func newSpecLoader(cmd *cobra.Command) *specLoader {
	return &specLoader{cmd: cmd, cache: map[string]*openapi.Spec{}}
}

// load returns the parsed spec for src, or nil when src is empty or cannot be used.
// Relative file paths are resolved against baseDir.
func (l *specLoader) load(src, baseDir string) *openapi.Spec {
	if src == "" {
		return nil
	}
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") && !filepath.IsAbs(src) {
		src = filepath.Join(baseDir, src)
	}
	if spec, seen := l.cache[src]; seen {
		return spec
	}
	var spec *openapi.Spec
	data, format, err := fetchSpec(src)
	if err != nil {
		fmt.Fprintf(l.cmd.ErrOrStderr(), "warning: could not load spec %q: %v\n", src, err)
	} else if spec, err = openapi.ParseBytes(data, format); err != nil {
		fmt.Fprintf(l.cmd.ErrOrStderr(), "warning: could not parse spec %q: %v\n", src, err)
		spec = nil
	}
	l.cache[src] = spec
	return spec
}

// cwdSpecPath makes a spec path given on the command line relative to the current
// directory, like every other CLI path. URLs and absolute paths are returned unchanged.
// (A specFile written inside a scenario or bundle is relative to that file instead.)
func cwdSpecPath(src string) string {
	if src == "" || strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") || filepath.IsAbs(src) {
		return src
	}
	if abs, err := filepath.Abs(src); err == nil {
		return abs
	}
	return src
}

// newScenarioRunner builds a runner that validates responses against spec when one is given.
func newScenarioRunner(spec *openapi.Spec) *scenario.Runner {
	exec := executor.New(executor.Options{})
	if spec != nil {
		return scenario.NewWithSpec(exec, spec)
	}
	return scenario.New(exec)
}

func bundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Run, validate, import and export bundles",
		Long: `A bundle is one file holding environments, reusable requests and scenarios —
for example the reply of an AI assistant, or an export of a workspace.

  clrsig bundle run bundle.yaml --env staging     run every scenario in it
  clrsig bundle validate bundle.yaml              check it without running anything
  clrsig bundle import bundle.yaml -w ./ws        save its files into a workspace
  clrsig bundle export -w ./ws -o bundle.yaml     write a workspace as one bundle`,
	}
	cmd.AddCommand(bundleRunCmd(), bundleValidateCmd(), bundleImportCmd(), bundleExportCmd())
	return cmd
}

func loadBundle(path string) (*serialization.Bundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading bundle: %w", err)
	}
	b, err := bundle.ParseText(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing bundle: %w", err)
	}
	return b, nil
}

// printIssues writes validation findings to stderr and reports whether any is an error.
func printIssues(cmd *cobra.Command, issues []bundle.Issue) (errors int) {
	for _, i := range issues {
		prefix := "warning"
		if i.Severity == bundle.SeverityError {
			prefix = "error"
			errors++
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", prefix, i.String())
	}
	return errors
}

func writeReport(cmd *cobra.Command, outputFile string, data []byte) error {
	if outputFile == "" {
		fmt.Fprint(cmd.OutOrStdout(), string(data))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(outputFile, data, 0o644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	return nil
}

func bundleRunCmd() *cobra.Command {
	var envName, envFile, specFile, format, outputFile string
	var vars []string
	var reports []string
	var showExtracted bool
	var seed int64
	var only []string

	cmd := &cobra.Command{
		Use:   "run <bundle-file>",
		Short: "Run every scenario in a bundle",
		Long: `Run the scenarios of a bundle directly, without importing it into a workspace.

The bundle is validated first; any error stops the run (exit 2). Requests the scenarios
refer to are taken from the bundle. The environment comes from the bundle: pass --env
when it has more than one, or --env-file to use an environment from a file instead.

Exit codes:
  0  every scenario passed
  1  at least one scenario failed
  2  the bundle is invalid or could not be read`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			bundlePath := args[0]

			plan, err := planReports(format, reports, outputFile)
			if err != nil {
				return err
			}

			b, err := loadBundle(bundlePath)
			if err != nil {
				return err
			}
			opts := bundle.Options{SpecFile: cwdSpecPath(specFile)}
			if n := printIssues(cmd, bundle.Validate(b, opts)); n > 0 {
				return fmt.Errorf("the bundle has %d error(s) and was not run", n)
			}

			var env *domain.Environment
			if envFile != "" {
				data, err := os.ReadFile(envFile)
				if err != nil {
					return fmt.Errorf("reading environment: %w", err)
				}
				if env, err = serialization.ParseEnvironmentFile(data, envFile); err != nil {
					return fmt.Errorf("parsing environment: %w", err)
				}
			} else if env, err = bundle.SelectEnvironment(b, envName); err != nil {
				return err
			}

			if env, err = finishEnvironment(cmd, env, vars); err != nil {
				return err
			}
			engine, env, err := prepareDynamic(cmd, env, seed)
			if err != nil {
				return err
			}

			runs, unmatched := bundle.FilterRunnables(bundle.Materialize(b, opts), only)
			if len(unmatched) > 0 {
				return fmt.Errorf("no scenario matches %s (use a scenario's title or file name)", strings.Join(unmatched, ", "))
			}
			if len(runs) == 0 {
				return fmt.Errorf("the bundle has no scenarios to run")
			}

			baseDir := filepath.Dir(bundlePath)
			loader := newSpecLoader(cmd)
			results := make([]*scenario.Result, 0, len(runs))
			for _, r := range runs {
				for _, ref := range scenario.ResolveRequestRefs(r.Scenario, baseDir) {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: scenario %q: requestRef %q could not be loaded\n", r.Scenario.Title, ref)
				}
				key := r.File
				if key == "" {
					key = r.Scenario.Title
				}
				runner := newScenarioRunner(loader.load(r.Scenario.SpecFile, baseDir)).WithEngine(engine, key)
				res := runner.Run(context.Background(), r.Scenario, env)
				res.RevealExtracted = showExtracted
				results = append(results, res)
			}

			total := time.Since(start)
			err = plan.emit(cmd, outputFile, func(f string) ([]byte, error) {
				return bundleReport(f, b.Title, results, total)
			})
			if err != nil {
				return err
			}

			passedAll, steps, asserts := true, 0, 0
			for _, r := range results {
				passedAll = passedAll && r.Passed
				p, f, _ := r.Counts()
				steps += p + f
				a, _ := r.AssertionCounts()
				asserts += a
			}
			telResult := "pass"
			if !passedAll {
				telResult = "fail"
			}
			newTelemetryClient().Record(telemetry.Event{
				Command: "bundle", Result: telResult, DurationMs: time.Since(start).Milliseconds(),
				StepCount: steps, AssertionCount: asserts,
			})
			if !passedAll {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "", "Name of the bundle's environment to use (needed when it has several)")
	cmd.Flags().StringArrayVar(&reports, "report", nil, "Also write a report to a file: format=file, for example junit=reports/bundle.xml (repeatable)")
	cmd.Flags().Int64Var(&seed, "seed", 0, "Seed for random and dynamic values; default: random, printed in the report (or set CLRSIG_SEED)")
	cmd.Flags().BoolVar(&showExtracted, "show-extracted", false, "Show extracted values whose names look secret (tokens, passwords, keys); secret environment values are always hidden")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "Set or override an environment variable for this run: name=value (repeatable)")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Use an environment file instead of the bundle's own (YAML or JSON)")
	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "OpenAPI spec file or URL; replaces the spec of every scenario")
	cmd.Flags().StringSliceVar(&only, "scenario", nil, "Run only the scenario with this title or file name (repeatable)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", formatUsage)
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write the report to a file (parent directories created automatically)")
	return cmd
}

func bundleReport(format, title string, results []*scenario.Result, total time.Duration) ([]byte, error) {
	if title == "" {
		title = "Bundle"
	}
	switch strings.ToLower(format) {
	case "markdown":
		return reporting.MarkdownMulti(title, results, total), nil

	case "junit":
		out, err := reporting.JUnitMulti(title, results)
		if err != nil {
			return nil, fmt.Errorf("generating JUnit report: %w", err)
		}
		return out, nil

	case "html":
		out, err := reporting.HTMLMulti(title, results, total)
		if err != nil {
			return nil, fmt.Errorf("generating HTML report: %w", err)
		}
		return out, nil

	case "json":
		type jsonBundle struct {
			Bundle     string            `json:"bundle"`
			Passed     bool              `json:"passed"`
			DurationMs int64             `json:"durationMs"`
			Scenarios  []json.RawMessage `json:"scenarios"`
		}
		out := jsonBundle{Bundle: title, Passed: true, DurationMs: total.Milliseconds(), Scenarios: []json.RawMessage{}}
		for _, r := range results {
			raw, err := reporting.JSON(r)
			if err != nil {
				return nil, fmt.Errorf("generating JSON report: %w", err)
			}
			out.Passed = out.Passed && r.Passed
			out.Scenarios = append(out.Scenarios, raw)
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Bundle: %s  (%d scenario%s)\n\n", title, len(results), plural(len(results)))
	passedScenarios, steps, assertsPassed, assertsTotal := 0, 0, 0, 0
	for _, r := range results {
		b.WriteString(reporting.Text(r))
		b.WriteString("\n")
		if r.Passed {
			passedScenarios++
		}
		p, f, _ := r.Counts()
		steps += p + f
		ap, af := r.AssertionCounts()
		assertsPassed += ap
		assertsTotal += ap + af
	}
	verdict := "PASSED"
	if passedScenarios != len(results) {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "Bundle %s  %d/%d scenarios passed  (%d steps, %d/%d assertions passed)  %dms\n",
		verdict, passedScenarios, len(results), steps, assertsPassed, assertsTotal, total.Milliseconds())
	return []byte(b.String()), nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func bundleValidateCmd() *cobra.Command {
	var specFile string
	cmd := &cobra.Command{
		Use:   "validate <bundle-file>",
		Short: "Check a bundle for errors without running or importing it",
		Long: `Exit codes:
  0  the bundle is valid (warnings are printed but do not fail)
  1  the bundle has errors
  2  the file could not be read or parsed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := loadBundle(args[0])
			if err != nil {
				return err
			}
			issues := bundle.Validate(b, bundle.Options{SpecFile: cwdSpecPath(specFile)})
			errs := printIssues(cmd, issues)
			warns := len(issues) - errs
			if errs > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Bundle is invalid: %d error(s), %d warning(s)\n", errs, warns)
				return &exitError{1}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Bundle is valid: %d environment(s), %d request(s), %d scenario(s), %d warning(s)\n",
				len(b.Environments), len(b.Requests), len(b.Scenarios), warns)
			return nil
		},
	}
	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "OpenAPI spec file or URL, as it would be given to run or import")
	return cmd
}

func bundleImportCmd() *cobra.Command {
	var workspacePath, specFile, onConflict string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "import <bundle-file>",
		Short: "Save a bundle's environments, requests and scenarios into a workspace",
		Long: `Writes ordinary request, scenario and environment files under requests/, scenarios/ and
environments/ of the workspace. Nothing is written if the bundle has any error.

Exit codes: 0 success, 2 the bundle is invalid or could not be read.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if workspacePath == "" {
				return fmt.Errorf("--workspace is required")
			}
			b, err := loadBundle(args[0])
			if err != nil {
				return err
			}
			plan, err := bundle.Prepare(b, workspacePath, bundle.Options{SpecFile: specFile}, bundle.Strategy(onConflict))
			if err != nil {
				return err
			}
			if n := printIssues(cmd, plan.Issues); n > 0 {
				return fmt.Errorf("the bundle has %d error(s) and was not imported", n)
			}
			for _, e := range plan.Entries {
				line := fmt.Sprintf("  %-9s %s", e.Action, e.FinalPath)
				if e.Action == bundle.ActionRename {
					line += fmt.Sprintf("  (wanted %s)", e.RelPath)
				}
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %d file(s) would be written, nothing changed\n", len(plan.Entries))
				return nil
			}
			res, err := plan.Write(workspacePath)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported: %d created, %d renamed, %d overwritten, %d skipped\n",
				res.Created, res.Renamed, res.Overwritten, res.Skipped)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspacePath, "workspace", "w", "", "Workspace directory to import into (required)")
	cmd.Flags().StringVarP(&specFile, "spec-file", "s", "", "OpenAPI spec file or URL; replaces the spec of every imported scenario")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "rename", "When a file already exists: rename, skip or overwrite")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be written without writing it")
	return cmd
}

func bundleExportCmd() *cobra.Command {
	var workspacePath, outputFile string
	var keepSecrets bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Write a workspace's environments, requests and scenarios as one bundle",
		Long: `Secret environment values are replaced with CHANGE_ME so the bundle is safe to share;
pass --keep-secrets to write them as they are. The workspace is never modified.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if workspacePath == "" {
				return fmt.Errorf("--workspace is required")
			}
			res, err := bundle.Export(workspacePath, bundle.ExportOptions{KeepSecrets: keepSecrets})
			if res != nil {
				for _, s := range res.Skipped {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipped %s: %s\n", s.Path, s.Reason)
				}
				for _, w := range res.Warnings {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
				}
			}
			if err != nil {
				return err
			}
			data, err := res.Marshal()
			if err != nil {
				return err
			}
			if res.RedactedSecrets > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: %d secret value(s) replaced with %s (use --keep-secrets to keep them)\n", res.RedactedSecrets, bundle.SecretPlaceholder)
			}
			if err := writeReport(cmd, outputFile, data); err != nil {
				return err
			}
			if outputFile != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "Exported %d environment(s), %d request(s), %d scenario(s) to %s\n",
					len(res.Bundle.Environments), len(res.Bundle.Requests), len(res.Bundle.Scenarios), outputFile)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspacePath, "workspace", "w", "", "Workspace directory to export (required)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write the bundle to a file instead of standard output")
	cmd.Flags().BoolVar(&keepSecrets, "keep-secrets", false, "Write secret values as they are instead of CHANGE_ME")
	return cmd
}
