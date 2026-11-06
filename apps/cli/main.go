package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/variables"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "apidesk",
		Short: "ApiDesk — API Contract Validation and Spec Conformance Testing",
	}

	root.AddCommand(runCmd())
	root.AddCommand(requestCmd())
	root.AddCommand(validateCmd())

	if err := root.Execute(); err != nil {
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
				os.Exit(1)
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
				fmt.Fprintln(cmd.OutOrStdout(), string(b))

			default:
				for _, r := range results {
					if r.err != nil {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\n  PARSE ERROR: %s\n\n", r.path, r.err)
						continue
					}
					if len(r.issues) == 0 {
						fmt.Fprintf(cmd.OutOrStdout(), "%s  OK\n", r.path)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\n", r.path)
						for _, iss := range r.issues {
							fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", iss)
						}
						fmt.Fprintln(cmd.OutOrStdout())
					}
				}
				if len(results) > 1 {
					valid := 0
					for _, r := range results {
						if r.err == nil && len(r.issues) == 0 {
							valid++
						}
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%d file(s) checked  %d valid  %d invalid\n",
						len(results), valid, len(results)-valid)
				}
			}

			if !allValid {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
	return cmd
}

func runCmd() *cobra.Command {
	var envFile string
	var format string

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
			runner := scenario.New(exec)
			result := runner.Run(context.Background(), s, env)

			switch strings.ToLower(format) {
			case "json":
				out, err := reporting.JSON(result)
				if err != nil {
					return fmt.Errorf("generating JSON report: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
			case "junit":
				out, err := reporting.JUnit(result)
				if err != nil {
					return fmt.Errorf("generating JUnit report: %w", err)
				}
				fmt.Fprint(cmd.OutOrStdout(), string(out))
			default:
				fmt.Fprint(cmd.OutOrStdout(), reporting.Text(result))
			}

			if !result.Passed {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&envFile, "env", "e", "", "Path to environment file (YAML or JSON)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, or junit")

	return cmd
}
