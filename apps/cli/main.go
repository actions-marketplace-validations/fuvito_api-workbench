package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "endpointdesk",
		Short: "EndpointDesk — API Contract Validation and Spec Conformance Testing",
	}

	root.AddCommand(runCmd())

	if err := root.Execute(); err != nil {
		os.Exit(2)
	}
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
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")

	return cmd
}
