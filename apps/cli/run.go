package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/telemetry"
)

type runOptions struct {
	envFile       string
	format        string
	outputFile    string
	timeoutMs     int
	specFile      string
	showCoverage  bool
	workspacePath string
	failFast      bool
	noRecursive   bool
	vars          []string
}

// scenarioRun is one executed scenario.
type scenarioRun struct {
	path   string
	result *scenario.Result
	cov    *openapi.ScenarioCoverageResult
	// specRef is the spec as written (the --spec value or the scenario's specFile:). It is the
	// key coverage accumulates under.
	specRef string
}

func runCmd() *cobra.Command {
	var o runOptions

	cmd := &cobra.Command{
		Use:   "run <scenario-file-or-folder>...",
		Short: "Execute scenarios and report results",
		Long: `Run one scenario file, or several: pass any number of files and folders.

A folder runs every scenario in it and, unless --no-recursive is given, in its sub-folders.
Files in a folder that are not scenarios (environments, requests, specs) are skipped and listed.
All scenarios are read before any request is sent; if one cannot be read, nothing runs.

With several scenarios you get one combined report and one exit code.

Exit codes:
  0  every scenario passed
  1  at least one scenario failed
  2  a file could not be read or parsed, nothing was found, or a configuration error`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScenarios(cmd, args, o)
		},
	}

	cmd.Flags().StringVarP(&o.envFile, "env", "e", "", "Path to environment file (YAML or JSON)")
	cmd.Flags().StringVarP(&o.format, "format", "f", "text", "Output format: text, json, junit, or html")
	cmd.Flags().StringVarP(&o.outputFile, "output", "o", "", "Write report to file (parent directories created automatically)")
	cmd.Flags().IntVar(&o.timeoutMs, "timeout", 0, "Global step timeout in milliseconds (overrides per-step timeout, 0 = use per-step value)")
	cmd.Flags().StringVarP(&o.specFile, "spec", "s", "", "OpenAPI spec file or URL (overrides specFile: in scenario)")
	cmd.Flags().BoolVar(&o.showCoverage, "coverage", false, "Report which spec operations were exercised (requires --spec or specFile: in scenario)")
	cmd.Flags().StringVarP(&o.workspacePath, "workspace", "w", "", "Workspace directory to accumulate coverage into")
	cmd.Flags().StringArrayVar(&o.vars, "var", nil, "Set or override an environment variable for this run: name=value (repeatable)")
	cmd.Flags().BoolVar(&o.failFast, "fail-fast", false, "Stop after the first scenario that fails")
	cmd.Flags().BoolVar(&o.noRecursive, "no-recursive", false, "Only look at the top level of each folder, not its sub-folders")

	return cmd
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runScenarios(cmd *cobra.Command, args []string, o runOptions) error {
	start := time.Now()

	// One file keeps the exact behaviour and output it always had.
	singleFile := len(args) == 1 && !isDir(args[0])

	paths := args
	if !singleFile {
		found, err := scenario.Discover(args, !o.noRecursive)
		if err != nil {
			return err
		}
		paths = found.Files
		if n := len(found.Skipped); n > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "note: skipped %d file%s that %s not scenarios: %s\n",
				n, plural(n), map[bool]string{true: "is", false: "are"}[n == 1], listNames(found.Skipped, 5))
		}
	}

	// Read every scenario before sending a single request.
	scenarios := make([]*domain.Scenario, len(paths))
	var problems []string
	for i, p := range paths {
		s, err := loadScenario(p, o.timeoutMs)
		if err != nil {
			if singleFile {
				return err
			}
			problems = append(problems, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		scenarios[i] = s
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d scenario file%s could not be loaded, so nothing was run:\n  %s",
			len(problems), plural(len(problems)), strings.Join(problems, "\n  "))
	}

	env, err := loadRunEnvironment(o.envFile)
	if err != nil {
		return err
	}
	if env, err = finishEnvironment(cmd, env, o.vars); err != nil {
		return err
	}

	loader := newSpecLoader(cmd)
	var runs []*scenarioRun
	for i, p := range paths {
		r := executeScenario(cmd, p, scenarios[i], env, loader, o)
		runs = append(runs, r)
		if o.failFast && !r.result.Passed {
			if left := len(paths) - i - 1; left > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: stopped after the first failure (--fail-fast); %d scenario%s not run\n", left, plural(left))
			}
			break
		}
	}

	var report []byte
	if singleFile {
		report, err = singleReport(runs[0], o)
	} else {
		report, err = combinedReport(runTitle(args), runs, o, time.Since(start))
	}
	if err != nil {
		return err
	}
	if err := writeReport(cmd, o.outputFile, report); err != nil {
		return err
	}

	allPassed, steps, asserts := true, 0, 0
	for _, r := range runs {
		allPassed = allPassed && r.result.Passed
		p, f, _ := r.result.Counts()
		steps += p + f
		a, _ := r.result.AssertionCounts()
		asserts += a
	}
	telResult := "pass"
	if !allPassed {
		telResult = "fail"
	}
	newTelemetryClient().Record(telemetry.Event{
		Command:        "run",
		Result:         telResult,
		DurationMs:     time.Since(start).Milliseconds(),
		StepCount:      steps,
		AssertionCount: asserts,
	})
	if !allPassed {
		return &exitError{1}
	}
	return nil
}

func loadScenario(path string, timeoutMs int) (*domain.Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading scenario: %w", err)
	}
	s, err := serialization.ParseScenarioFile(data, path)
	if err != nil {
		return nil, fmt.Errorf("parsing scenario: %w", err)
	}
	if timeoutMs > 0 {
		for i := range s.Steps {
			s.Steps[i].TimeoutMs = timeoutMs
		}
	}
	return s, nil
}

func loadRunEnvironment(envFile string) (*domain.Environment, error) {
	if envFile == "" {
		return nil, nil
	}
	envData, err := os.ReadFile(envFile)
	if err != nil {
		return nil, fmt.Errorf("reading environment: %w", err)
	}
	env, err := serialization.ParseEnvironmentFile(envData, envFile)
	if err != nil {
		return nil, fmt.Errorf("parsing environment: %w", err)
	}
	return env, nil
}

// executeScenario runs one scenario, computes its spec coverage and, with --workspace,
// accumulates that coverage.
func executeScenario(cmd *cobra.Command, scenarioPath string, s *domain.Scenario, env *domain.Environment, loader *specLoader, o runOptions) *scenarioRun {
	// --spec overrides specFile: from the scenario. A --spec path is relative to the current
	// directory; a specFile: inside the scenario is relative to the scenario file.
	// specRef stays as written because it is also the key coverage is accumulated under.
	specRef := s.SpecFile
	loadSrc := s.SpecFile
	if o.specFile != "" {
		specRef = o.specFile
		loadSrc = cwdSpecPath(o.specFile)
	}

	loadedSpec := loader.load(loadSrc, filepath.Dir(scenarioPath))
	runner := newScenarioRunner(loadedSpec)

	// Steps that use requestRef need their request files loaded first. Look next to the
	// scenario, then in the requests/ folder beside its scenarios/ folder, then in the
	// workspace's requests/ folder when --workspace is given.
	refDirs := []string{filepath.Dir(scenarioPath), filepath.Join(filepath.Dir(scenarioPath), "..", "requests")}
	if o.workspacePath != "" {
		refDirs = append(refDirs, filepath.Join(o.workspacePath, "requests"))
	}
	for _, ref := range scenario.ResolveRequestRefs(s, refDirs...) {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: requestRef %q could not be loaded\n", ref)
	}
	result := runner.Run(context.Background(), s, env)

	var cov *openapi.ScenarioCoverageResult
	if loadedSpec != nil && (o.showCoverage || o.workspacePath != "") {
		var executed []openapi.ExecutedRequest
		for _, sr := range result.Steps {
			if sr.Request != nil && sr.Response != nil {
				executed = append(executed, openapi.ExecutedRequest{
					Method:     string(sr.Request.Method),
					URL:        sr.Request.URL,
					StatusCode: sr.Response.StatusCode,
					HadAuth:    sr.Request.Auth != nil,
					Passed:     sr.Passed,
				})
			}
		}
		cov = loadedSpec.ScenarioCoverage(executed)
	} else if o.showCoverage && loadedSpec == nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: --coverage requires a spec (use --spec or add specFile: to the scenario)")
	}

	if o.workspacePath != "" && loadedSpec != nil && cov != nil {
		accumulateCoverage(cmd, o.workspacePath, specRef, result, cov)
	}
	return &scenarioRun{path: scenarioPath, result: result, cov: cov, specRef: specRef}
}

func accumulateCoverage(cmd *cobra.Command, workspacePath, specRef string, result *scenario.Result, cov *openapi.ScenarioCoverageResult) {
	stored := coverage.Load(workspacePath, specRef)
	stored.TotalRuns++
	stored.LastRunAt = time.Now()
	if result.Passed {
		stored.PassedRuns++
	} else {
		stored.FailedRuns++
	}
	var exercised []string
	for _, op := range cov.Exercised {
		exercised = append(exercised, op.String())
	}
	stored.Exercised = coverage.MergeExercised(stored.Exercised, exercised)
	stored.TotalOps = cov.Total
	// Merge per-operation quality signals (status codes seen, auth presence).
	now := time.Now()
	if stored.OperationResults == nil {
		stored.OperationResults = make(map[string]coverage.OperationResult)
	}
	for _, er := range cov.ExercisedResults {
		prev := stored.OperationResults[er.String()]
		stored.OperationResults[er.String()] = coverage.OperationResult{
			LastPassed:  er.Passed,
			LastRunAt:   now,
			StatusCodes: coverage.MergeStatusCodes(prev.StatusCodes, er.StatusCodes),
			HadAuth:     prev.HadAuth || er.HadAuth,
		}
	}
	if err := coverage.Save(workspacePath, stored); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not save coverage: %v\n", err)
	}
}

func singleReport(r *scenarioRun, o runOptions) ([]byte, error) {
	switch strings.ToLower(o.format) {
	case "json":
		var b []byte
		var err error
		if r.cov != nil && o.showCoverage {
			b, err = reporting.JSONWithCoverage(r.result, r.cov)
		} else {
			b, err = reporting.JSON(r.result)
		}
		if err != nil {
			return nil, fmt.Errorf("generating JSON report: %w", err)
		}
		return append(b, '\n'), nil
	case "junit":
		b, err := reporting.JUnit(r.result)
		if err != nil {
			return nil, fmt.Errorf("generating JUnit report: %w", err)
		}
		return b, nil
	case "html":
		var htmlCov *openapi.ScenarioCoverageResult
		if o.showCoverage {
			htmlCov = r.cov
		}
		b, err := reporting.HTML(r.result, htmlCov)
		if err != nil {
			return nil, fmt.Errorf("generating HTML report: %w", err)
		}
		return b, nil
	}
	if r.cov != nil && o.showCoverage {
		return []byte(reporting.TextWithCoverage(r.result, r.cov)), nil
	}
	return []byte(reporting.Text(r.result)), nil
}

func combinedReport(title string, runs []*scenarioRun, o runOptions, total time.Duration) ([]byte, error) {
	results := make([]*scenario.Result, len(runs))
	for i, r := range runs {
		results[i] = r.result
	}
	var cov []reporting.SpecCoverage
	if o.showCoverage {
		cov = groupCoverage(runs)
	}

	switch strings.ToLower(o.format) {
	case "json":
		b, err := reporting.JSONMulti(title, results, cov, total)
		if err != nil {
			return nil, fmt.Errorf("generating JSON report: %w", err)
		}
		return b, nil
	case "junit":
		b, err := reporting.JUnitMulti(title, results)
		if err != nil {
			return nil, fmt.Errorf("generating JUnit report: %w", err)
		}
		return b, nil
	case "html":
		b, err := reporting.HTMLMulti(title, results, total, cov...)
		if err != nil {
			return nil, fmt.Errorf("generating HTML report: %w", err)
		}
		return b, nil
	}
	return []byte(reporting.TextMulti(title, results, cov, total)), nil
}

// groupCoverage merges the coverage of the scenarios that used the same spec, in the order the
// specs first appear.
func groupCoverage(runs []*scenarioRun) []reporting.SpecCoverage {
	var order []string
	groups := map[string][]*openapi.ScenarioCoverageResult{}
	for _, r := range runs {
		if r.cov == nil {
			continue
		}
		if _, seen := groups[r.specRef]; !seen {
			order = append(order, r.specRef)
		}
		groups[r.specRef] = append(groups[r.specRef], r.cov)
	}
	out := make([]reporting.SpecCoverage, 0, len(order))
	for _, ref := range order {
		out = append(out, reporting.SpecCoverage{Spec: ref, Result: reporting.MergeCoverage(groups[ref])})
	}
	return out
}

// runTitle names a combined report: the folder name when one folder was given, else "scenarios".
func runTitle(args []string) string {
	if len(args) == 1 && isDir(args[0]) {
		if abs, err := filepath.Abs(args[0]); err == nil {
			return filepath.Base(abs)
		}
	}
	return "scenarios"
}

func listNames(paths []string, max int) string {
	names := make([]string, 0, len(paths))
	for _, p := range paths {
		names = append(names, filepath.ToSlash(p))
	}
	if len(names) > max {
		return strings.Join(names[:max], ", ") + fmt.Sprintf(", and %d more", len(names)-max)
	}
	return strings.Join(names, ", ")
}
