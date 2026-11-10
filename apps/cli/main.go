package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/history"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/variables"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags="-X main.version=v0.1.0".
var version = "dev"

// exitError is returned from cobra RunE to signal a non-zero exit code.
// Using a sentinel avoids calling os.Exit inside a handler, which would
// bypass any future deferred cleanup.
type exitError struct{ code int }

func (e *exitError) Error() string { return "" }

func main() {
	root := &cobra.Command{
		Use:     "clrsig",
		Short:   "Clear Signal — API Contract Validation and Spec Conformance Testing",
		Version: version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(runCmd())
	root.AddCommand(requestCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(generateCmd())
	root.AddCommand(conformCmd())
	root.AddCommand(diffCmd())

	if err := root.Execute(); err != nil {
		var xe *exitError
		if errors.As(err, &xe) {
			os.Exit(xe.code)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}

func requestCmd() *cobra.Command {
	var envFile string
	var format string
	var timeoutMs int

	cmd := &cobra.Command{
		Use:   "request <request-file>",
		Short: "Execute a saved request file and print the response",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqPath := args[0]

			data, err := os.ReadFile(reqPath)
			if err != nil {
				return fmt.Errorf("reading request: %w", err)
			}

			req, err := serialization.ParseRequestFile(data, reqPath)
			if err != nil {
				return fmt.Errorf("parsing request: %w", err)
			}

			var env *domain.Environment
			if envFile != "" {
				envData, err := os.ReadFile(envFile)
				if err != nil {
					return fmt.Errorf("reading environment: %w", err)
				}
				env, err = serialization.ParseEnvironmentFile(envData, envFile)
				if err != nil {
					return fmt.Errorf("parsing environment: %w", err)
				}
			}

			resolver := variables.NewResolver(env)
			resolved := resolver.InterpolateRequest(req)

			if timeoutMs > 0 {
				resolved.Timeout = time.Duration(timeoutMs) * time.Millisecond
			}

			exec := executor.New(executor.Options{})
			resp, execErr := exec.Execute(context.Background(), resolved)

			method := string(resolved.Method)
			switch strings.ToLower(format) {
			case "json":
				out, err := reporting.JSONRequest(method, resolved.URL, resp, execErr)
				if err != nil {
					return fmt.Errorf("generating JSON: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
			default:
				fmt.Fprint(cmd.OutOrStdout(), reporting.TextRequest(method, resolved.URL, resp, execErr))
			}

			if execErr != nil {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&envFile, "env", "e", "", "Path to environment file (YAML or JSON)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
	cmd.Flags().IntVar(&timeoutMs, "timeout", 0, "Request timeout in milliseconds (overrides file setting, 0 = use file value)")

	return cmd
}

func validateCmd() *cobra.Command {
	var format string
	var outputFile string

	cmd := &cobra.Command{
		Use:   "validate <scenario-file> [file2...]",
		Short: "Check scenario files for errors without executing them",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			type result struct {
				path   string
				issues []scenario.Issue
				err    error
			}

			results := make([]result, 0, len(args))
			for _, path := range args {
				data, err := os.ReadFile(path)
				if err != nil {
					results = append(results, result{path: path, err: fmt.Errorf("reading file: %w", err)})
					continue
				}
				sc, err := serialization.ParseScenarioFile(data, path)
				if err != nil {
					results = append(results, result{path: path, err: fmt.Errorf("parse error: %w", err)})
					continue
				}
				results = append(results, result{path: path, issues: scenario.Validate(sc, filepath.Dir(path))})
			}

			allValid := true
			for _, r := range results {
				if r.err != nil || len(r.issues) > 0 {
					allValid = false
					break
				}
			}

			var reportBytes []byte
			switch strings.ToLower(format) {
			case "json":
				type jsonIssue struct {
					Step    int    `json:"step"`
					Name    string `json:"stepName,omitempty"`
					Message string `json:"message"`
				}
				type jsonResult struct {
					Path   string      `json:"path"`
					Valid  bool        `json:"valid"`
					Error  string      `json:"error,omitempty"`
					Issues []jsonIssue `json:"issues"`
				}
				out := make([]jsonResult, 0, len(results))
				for _, r := range results {
					jr := jsonResult{Path: r.path, Valid: r.err == nil && len(r.issues) == 0, Issues: []jsonIssue{}}
					if r.err != nil {
						jr.Error = r.err.Error()
					}
					for _, iss := range r.issues {
						jr.Issues = append(jr.Issues, jsonIssue{Step: iss.Step, Name: iss.Name, Message: iss.Message})
					}
					out = append(out, jr)
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				reportBytes = append(b, '\n')

			default:
				var sb strings.Builder
				for _, r := range results {
					if r.err != nil {
						fmt.Fprintf(&sb, "%s\n  PARSE ERROR: %s\n\n", r.path, r.err)
						continue
					}
					if len(r.issues) == 0 {
						fmt.Fprintf(&sb, "%s  OK\n", r.path)
					} else {
						fmt.Fprintf(&sb, "%s\n", r.path)
						for _, iss := range r.issues {
							fmt.Fprintf(&sb, "  %s\n", iss)
						}
						sb.WriteByte('\n')
					}
				}
				if len(results) > 1 {
					valid := 0
					for _, r := range results {
						if r.err == nil && len(r.issues) == 0 {
							valid++
						}
					}
					fmt.Fprintf(&sb, "%d file(s) checked  %d valid  %d invalid\n",
						len(results), valid, len(results)-valid)
				}
				reportBytes = []byte(sb.String())
			}

			if outputFile != "" {
				if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				if err := os.WriteFile(outputFile, reportBytes, 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), string(reportBytes))
			}

			if !allValid {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report to file (parent directories created automatically)")
	return cmd
}

func runCmd() *cobra.Command {
	var envFile string
	var format string
	var outputFile string
	var timeoutMs int

	cmd := &cobra.Command{
		Use:   "run <scenario-file>",
		Short: "Execute a scenario and report results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarioPath := args[0]

			data, err := os.ReadFile(scenarioPath)
			if err != nil {
				return fmt.Errorf("reading scenario: %w", err)
			}

			s, err := serialization.ParseScenarioFile(data, scenarioPath)
			if err != nil {
				return fmt.Errorf("parsing scenario: %w", err)
			}

			if timeoutMs > 0 {
				for i := range s.Steps {
					s.Steps[i].TimeoutMs = timeoutMs
				}
			}

			var env *domain.Environment
			if envFile != "" {
				envData, err := os.ReadFile(envFile)
				if err != nil {
					return fmt.Errorf("reading environment: %w", err)
				}
				env, err = serialization.ParseEnvironmentFile(envData, envFile)
				if err != nil {
					return fmt.Errorf("parsing environment: %w", err)
				}
			}

			exec := executor.New(executor.Options{})
			var runner *scenario.Runner
			if s.SpecFile != "" {
				specPath := s.SpecFile
				if !filepath.IsAbs(specPath) {
					specPath = filepath.Join(filepath.Dir(scenarioPath), specPath)
				}
				specData, err := os.ReadFile(specPath)
				if err != nil {
					return fmt.Errorf("reading spec file %q: %w", specPath, err)
				}
				ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(specPath)), ".")
				if ext == "yml" {
					ext = "yaml"
				}
				spec, err := openapi.ParseBytes(specData, ext)
				if err != nil {
					return fmt.Errorf("parsing spec file %q: %w", specPath, err)
				}
				runner = scenario.NewWithSpec(exec, spec)
			} else {
				runner = scenario.New(exec)
			}
			result := runner.Run(context.Background(), s, env)

			var reportBytes []byte
			switch strings.ToLower(format) {
			case "json":
				reportBytes, err = reporting.JSON(result)
				if err != nil {
					return fmt.Errorf("generating JSON report: %w", err)
				}
				reportBytes = append(reportBytes, '\n')
			case "junit":
				reportBytes, err = reporting.JUnit(result)
				if err != nil {
					return fmt.Errorf("generating JUnit report: %w", err)
				}
			default:
				reportBytes = []byte(reporting.Text(result))
			}

			if outputFile != "" {
				if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				if err := os.WriteFile(outputFile, reportBytes, 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), string(reportBytes))
			}

			if !result.Passed {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&envFile, "env", "e", "", "Path to environment file (YAML or JSON)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, or junit")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report to file (parent directories created automatically)")
	cmd.Flags().IntVar(&timeoutMs, "timeout", 0, "Global step timeout in milliseconds (overrides per-step timeout, 0 = use per-step value)")

	return cmd
}

func generateCmd() *cobra.Command {
	var baseURL string
	var outputDir string
	var format string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "generate <spec-file-or-url>",
		Short: "Generate request files from an OpenAPI specification",
		Long: `Parse an OpenAPI 3.0/3.1 spec and write one YAML request file per operation.

The argument may be a local file path or an http/https URL.

Each generated file contains the HTTP method, URL (with path parameters as
{{paramName}} placeholders), required headers and query params, an empty JSON
body stub for POST/PUT/PATCH operations, and a Bearer auth hint when the
operation declares a security requirement.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]

			data, specFormat, err := fetchSpec(src)
			if err != nil {
				return err
			}

			spec, err := openapi.ParseBytes(data, specFormat)
			if err != nil {
				return fmt.Errorf("parsing spec: %w", err)
			}

			title, version := spec.Info()
			reqs, err := openapi.GenerateRequests(spec, baseURL)
			if err != nil {
				return fmt.Errorf("generating requests: %w", err)
			}

			// Resolve output directory.
			destDir := outputDir
			if destDir == "" {
				dirName := genSanitizeDirName(title)
				if dirName == "" {
					dirName = "requests"
				}
				destDir = dirName
			}

			type genResult struct {
				OperationID string   `json:"operationId,omitempty"`
				Method      string   `json:"method"`
				URL         string   `json:"url"`
				Path        string   `json:"path"`
				Tags        []string `json:"tags,omitempty"`
			}
			var results []genResult
			var skipped int

			for _, gr := range reqs {
				filename := genSanitizeFilename(gr.OperationID)
				if filename == "" {
					filename = genSanitizeFilename(gr.Request.Title)
				}
				if filename == "" {
					filename = strings.ToLower(string(gr.Request.Method)) + "-request"
				}
				savePath := filepath.Join(destDir, filename+".yaml")

				if !dryRun {
					yamlData, err := serialization.WriteRequest(gr.Request, "yaml")
					if err != nil {
						skipped++
						continue
					}
					if err := os.MkdirAll(destDir, 0o755); err != nil {
						return fmt.Errorf("creating output directory: %w", err)
					}
					if err := os.WriteFile(savePath, yamlData, 0o644); err != nil {
						skipped++
						continue
					}
				}

				results = append(results, genResult{
					OperationID: gr.OperationID,
					Method:      string(gr.Request.Method),
					URL:         gr.Request.URL,
					Path:        savePath,
					Tags:        gr.Tags,
				})
			}

			switch strings.ToLower(format) {
			case "json":
				type jsonOut struct {
					SpecTitle   string      `json:"specTitle"`
					SpecVersion string      `json:"specVersion"`
					OutputDir   string      `json:"outputDir"`
					DryRun      bool        `json:"dryRun"`
					Generated   []genResult `json:"generated"`
					Skipped     int         `json:"skipped"`
				}
				b, _ := json.MarshalIndent(jsonOut{
					SpecTitle:   title,
					SpecVersion: version,
					OutputDir:   destDir,
					DryRun:      dryRun,
					Generated:   results,
					Skipped:     skipped,
				}, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
			default:
				dryTag := ""
				if dryRun {
					dryTag = " (dry run)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s%s → %s\n\n", title, version, dryTag, destDir)
				for _, r := range results {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-7s %s\n", r.Method, r.URL)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n%d request(s) generated", len(results))
				if skipped > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "  %d skipped (write error)", skipped)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&baseURL, "base-url", "b", "", "Override base URL from spec (e.g. https://api.example.com/v1)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for generated files (default: ./<spec-title>)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be generated without writing any files")

	return cmd
}

func conformCmd() *cobra.Command {
	var baseURL string
	var envFile string
	var tagsCSV string
	var pathsCSV string
	var format string
	var outputFile string
	var requestTimeout int
	var failFast bool
	var showCoverage bool

	cmd := &cobra.Command{
		Use:   "conform <spec-file-or-url>",
		Short: "Run all spec operations against a live server and validate responses",
		Long: `Parse an OpenAPI 3.0/3.1 spec, execute each declared operation against
a running server, and validate every response against the spec's declared
response schema.

Exit codes:
  0  all operations conformed
  1  one or more violations or execution failures
  2  spec parse or configuration error`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]

			data, specFormat, err := fetchSpec(src)
			if err != nil {
				return err
			}

			spec, err := openapi.ParseBytes(data, specFormat)
			if err != nil {
				return fmt.Errorf("parsing spec: %w", err)
			}

			var env *domain.Environment
			if envFile != "" {
				envData, err := os.ReadFile(envFile)
				if err != nil {
					return fmt.Errorf("reading environment: %w", err)
				}
				env, err = serialization.ParseEnvironmentFile(envData, envFile)
				if err != nil {
					return fmt.Errorf("parsing environment: %w", err)
				}
			}

			opts := conformance.Options{
				FailFast:         failFast,
				RequestTimeoutMs: requestTimeout,
			}
			if tagsCSV != "" {
				for _, t := range strings.Split(tagsCSV, ",") {
					if t = strings.TrimSpace(t); t != "" {
						opts.Tags = append(opts.Tags, t)
					}
				}
			}
			if pathsCSV != "" {
				for _, p := range strings.Split(pathsCSV, ",") {
					if p = strings.TrimSpace(p); p != "" {
						opts.Paths = append(opts.Paths, p)
					}
				}
			}

			exec := executor.New(executor.Options{})
			runner := conformance.New(exec)
			result := runner.Run(context.Background(), spec, baseURL, env, opts)

			// Hash the spec bytes to group runs of the same spec in history.
			rawHash := sha256.Sum256(data)
			specHash := hex.EncodeToString(rawHash[:])

			// Record this run to history (silently — a broken db never blocks conformance).
			recordConformRun(src, specHash, result)

			var cov *conformance.CoverageReport
			if showCoverage {
				cov = conformance.ComputeCoverage(result)
			}

			var reportBytes []byte
			switch strings.ToLower(format) {
			case "json":
				reportBytes, err = reporting.ConformJSON(result, cov)
				if err != nil {
					return fmt.Errorf("generating JSON report: %w", err)
				}
				reportBytes = append(reportBytes, '\n')
			case "junit":
				reportBytes, err = reporting.ConformJUnit(result)
				if err != nil {
					return fmt.Errorf("generating JUnit report: %w", err)
				}
			default:
				reportBytes = []byte(reporting.ConformText(result, cov))
			}

			if outputFile != "" {
				if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				if err := os.WriteFile(outputFile, reportBytes, 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), string(reportBytes))
			}

			// Print cumulative coverage in text mode when --coverage is set and
			// there is more than one run recorded for this spec.
			if showCoverage && strings.ToLower(format) == "text" {
				printCumulativeCoverage(cmd, specHash)
			}

			_, failed, _ := result.Counts()
			if failed > 0 {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&baseURL, "base-url", "b", "", "Base URL to send requests to (overrides spec servers[0])")
	cmd.Flags().StringVarP(&envFile, "env", "e", "", "Path to environment file for variable resolution")
	cmd.Flags().StringVar(&tagsCSV, "tags", "", "Comma-separated list of OpenAPI tags to include")
	cmd.Flags().StringVar(&pathsCSV, "paths", "", "Comma-separated path prefixes to include")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, or junit")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report to file (parent directories created automatically)")
	cmd.Flags().IntVar(&requestTimeout, "request-timeout", 10000, "Per-request timeout in milliseconds")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop after the first violation or execution error")
	cmd.Flags().BoolVar(&showCoverage, "coverage", false, "Append a coverage summary showing which operations received a response")
	return cmd
}

// recordConformRun persists per-operation coverage to history.db.
// Errors are silently discarded so a broken database never blocks conformance output.
func recordConformRun(specPath, specHash string, result *conformance.ConformResult) {
	dbPath, err := history.DefaultPath()
	if err != nil {
		return
	}
	store, err := history.Open(dbPath)
	if err != nil {
		return
	}
	defer store.Close()

	ops := make([]history.ConformOp, len(result.Operations))
	for i, op := range result.Operations {
		ops[i] = history.ConformOp{
			OperationID: op.OperationID,
			Method:      op.Method,
			Path:        op.Path,
			Covered:     op.StatusCode > 0,
		}
	}
	_ = store.RecordConformRun(specPath, specHash, ops)
}

// printCumulativeCoverage writes the cross-run coverage summary to cmd's stdout
// when more than one run has been recorded for the given spec hash.
func printCumulativeCoverage(cmd *cobra.Command, specHash string) {
	dbPath, err := history.DefaultPath()
	if err != nil {
		return
	}
	store, err := history.Open(dbPath)
	if err != nil {
		return
	}
	defer store.Close()

	rep, err := store.CumulativeCoverage(specHash)
	if err != nil || rep == nil || rep.RunCount < 2 {
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nCumulative coverage (%d runs):  %d/%d (%.1f%%)\n",
		rep.RunCount, len(rep.EverCoveredOps), rep.TotalOperations, rep.Percent)
	if len(rep.NeverCoveredOps) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Never covered: %s\n", strings.Join(rep.NeverCoveredOps, ", "))
	}
}

// fetchSpec retrieves spec bytes and detected format from a file path or URL.
func fetchSpec(src string) (data []byte, format string, err error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, fetchErr := client.Get(src)
		if fetchErr != nil {
			return nil, "", fmt.Errorf("fetching spec: %w", fetchErr)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("server returned %d", resp.StatusCode)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", fmt.Errorf("reading response: %w", err)
		}
		ct := resp.Header.Get("Content-Type")
		if strings.Contains(ct, "json") || strings.HasSuffix(strings.ToLower(src), ".json") {
			format = "json"
		} else {
			format = "yaml"
		}
		return data, format, nil
	}
	data, err = os.ReadFile(src)
	if err != nil {
		return nil, "", fmt.Errorf("reading spec: %w", err)
	}
	format = strings.TrimPrefix(strings.ToLower(filepath.Ext(src)), ".")
	if format == "yml" {
		format = "yaml"
	}
	return data, format, nil
}

func diffCmd() *cobra.Command {
	var format string
	var outputFile string
	var breakingOnly bool

	cmd := &cobra.Command{
		Use:   "diff <base-spec> <revision-spec>",
		Short: "Compare two OpenAPI specs and report breaking and non-breaking changes",
		Long: `Parse two OpenAPI 3.0/3.1 specs and classify every change as breaking
(ERR), potentially breaking (WARN), or informational (INFO).

Exit codes:
  0  no breaking changes found
  1  one or more breaking changes found
  2  spec parse or configuration error`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseData, baseFmt, err := fetchSpec(args[0])
			if err != nil {
				return err
			}
			revData, revFmt, err := fetchSpec(args[1])
			if err != nil {
				return err
			}

			baseSpec, err := openapi.ParseBytes(baseData, baseFmt)
			if err != nil {
				return fmt.Errorf("parsing base spec: %w", err)
			}
			revSpec, err := openapi.ParseBytes(revData, revFmt)
			if err != nil {
				return fmt.Errorf("parsing revision spec: %w", err)
			}

			result, err := openapi.Diff(baseSpec, revSpec)
			if err != nil {
				return fmt.Errorf("diffing specs: %w", err)
			}

			changes := result.Changes
			if breakingOnly {
				filtered := changes[:0]
				for _, c := range changes {
					if c.Breaking {
						filtered = append(filtered, c)
					}
				}
				changes = filtered
			}

			var reportBytes []byte
			switch strings.ToLower(format) {
			case "junit":
				reportBytes, err = reporting.DiffJUnit(result)
				if err != nil {
					return fmt.Errorf("generating JUnit report: %w", err)
				}
			case "json":
				type jsonChange struct {
					ID       string `json:"id"`
					Level    string `json:"level"`
					Breaking bool   `json:"breaking"`
					Text     string `json:"text"`
				}
				type jsonOut struct {
					HasBreaking bool         `json:"hasBreaking"`
					Changes     []jsonChange `json:"changes"`
				}
				out := jsonOut{HasBreaking: result.HasBreaking}
				for _, c := range changes {
					out.Changes = append(out.Changes, jsonChange{
						ID:       c.ID,
						Level:    c.Level,
						Breaking: c.Breaking,
						Text:     c.Text,
					})
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				reportBytes = append(b, '\n')
			default:
				var sb strings.Builder
				if len(changes) == 0 {
					if breakingOnly {
						fmt.Fprintln(&sb, "No breaking changes.")
					} else {
						fmt.Fprintln(&sb, "No changes.")
					}
				} else {
					for _, c := range changes {
						mark := " "
						if c.Breaking {
							mark = "!"
						}
						fmt.Fprintf(&sb, "  [%s] %s  %s\n", c.Level, mark, c.Text)
					}
					fmt.Fprintln(&sb)
					breaking := 0
					for _, c := range changes {
						if c.Breaking {
							breaking++
						}
					}
					if result.HasBreaking {
						fmt.Fprintf(&sb, "%d change(s)  ·  %d breaking\n", len(changes), breaking)
					} else {
						fmt.Fprintf(&sb, "%d change(s)  ·  no breaking changes\n", len(changes))
					}
				}
				reportBytes = []byte(sb.String())
			}

			if outputFile != "" {
				if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				if err := os.WriteFile(outputFile, reportBytes, 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), string(reportBytes))
			}

			if result.HasBreaking {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, or junit")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report to file (parent directories created automatically)")
	cmd.Flags().BoolVar(&breakingOnly, "breaking-only", false, "Show only breaking changes")
	return cmd
}

func genSanitizeDirName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func genSanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	return b.String()
}
