// Package application holds use cases shared by the CLI, the MCP server and the desktop app, so
// every interface runs the same code with the same defaults.
package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// DefaultConformRequestTimeoutMs is the per-request timeout for a conformance run unless the
// caller overrides it. Interfaces take their default from here instead of choosing their own.
const DefaultConformRequestTimeoutMs = 10000

// ConformInput is everything a conformance run needs. The caller owns reading the spec and the
// environment, building the executor and rendering the output.
type ConformInput struct {
	Exec    *executor.Executor
	Spec    *openapi.Spec
	BaseURL string
	Env     *domain.Environment

	Tags              []string
	Paths             []string
	FailFast          bool
	RequestTimeoutMs  int            // 0 means DefaultConformRequestTimeoutMs
	MaxResponseTimeMs int            // 0 means no response-time limit
	PathLimits        map[string]int // per path prefix; the longest prefix wins over MaxResponseTimeMs
	ReadOnly          bool           // send only GET and HEAD operations

	// Progress, when set, is called after each operation with how many are done out of how many
	// and the operation just handled.
	Progress func(done, total int, op conformance.OperationResult)

	Coverage bool // compute the coverage report
	Negative bool // also run negative probes
	Baseline *conformance.Baseline

	// FailOn is the lowest severity that fails the run: checks.Failure (also the default when
	// empty) or checks.Warning. Notes never fail a run.
	FailOn checks.Severity
}

// ConformOutput is the outcome of a run. Failed already accounts for the baseline: with one, the
// run fails only on violations that are not in it.
type ConformOutput struct {
	Result     *conformance.ConformResult
	Coverage   *conformance.CoverageReport // nil unless requested
	Negative   *conformance.NegativeResult // nil unless requested
	Violations []conformance.Violation
	Diff       *conformance.BaselineDiff // nil without a baseline
	Failed     bool
	// Warnings is how many operations passed with warnings (0 when FailOn is warning: they are
	// failures then).
	Warnings int
	// NothingSelected explains why no operation was checked when --tags or --paths matched none
	// (empty otherwise). A run like that proves nothing, so the CLI reports it as an error.
	NothingSelected string
}

// SplitList turns a comma-separated flag value into trimmed, non-empty items.
func SplitList(csv string) []string {
	var out []string
	for _, item := range strings.Split(csv, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// RunConformance calls the operations of the spec and validates every response (API Contract
// Validation), optionally with negative probes, coverage and a baseline comparison.
func RunConformance(ctx context.Context, in ConformInput) ConformOutput {
	timeout := in.RequestTimeoutMs
	if timeout == 0 {
		timeout = DefaultConformRequestTimeoutMs
	}
	opts := conformance.Options{
		Tags:              in.Tags,
		Paths:             in.Paths,
		FailFast:          in.FailFast,
		ReadOnly:          in.ReadOnly,
		RequestTimeoutMs:  timeout,
		MaxResponseTimeMs: in.MaxResponseTimeMs,
		PathLimits:        in.PathLimits,
		OnOperation:       in.Progress,
	}

	out := ConformOutput{}
	out.Result = conformance.New(in.Exec).Run(ctx, in.Spec, in.BaseURL, in.Env, opts)
	failOn := in.FailOn
	if failOn != checks.Warning {
		failOn = checks.Failure
	}
	out.Result.FailOn = failOn
	if in.Coverage {
		out.Coverage = conformance.ComputeCoverage(out.Result)
	}
	if in.Negative {
		out.Negative = conformance.NewNegative(in.Exec).Run(ctx, in.Spec, in.BaseURL, in.Env, opts)
	}

	_, failed, _ := out.Result.Counts()
	negFailed := 0
	if out.Negative != nil {
		_, negFailed, _, _ = out.Negative.Counts()
	}
	out.Failed = failed > 0 || negFailed > 0
	out.Violations = conformance.ViolationsFor(out.Result, out.Negative, failOn)
	out.Warnings = out.Result.WarningCount()
	if in.Baseline != nil {
		diff := conformance.Compare(out.Violations, in.Baseline)
		out.Diff = &diff
		out.Failed = len(diff.New) > 0
		out.Result.BaselineApplied = true
		out.Result.KnownViolations = len(diff.Known)
		out.Result.BaselineClean = len(diff.New) == 0
	}
	if len(out.Result.Operations) > 0 && out.Result.FilterMatched == 0 && (len(in.Tags) > 0 || len(in.Paths) > 0) {
		out.NothingSelected = noMatchMessage(in.Tags, in.Paths, len(out.Result.Operations))
	}
	return out
}

func noMatchMessage(tags, paths []string, total int) string {
	var by []string
	if len(tags) > 0 {
		by = append(by, "--tags "+strings.Join(tags, ","))
	}
	if len(paths) > 0 {
		by = append(by, "--paths "+strings.Join(paths, ","))
	}
	return fmt.Sprintf("no operation matches %s (the spec has %d operations; --paths matches the start of the path as the spec writes it, for example /cases/resp-005; on Windows Git Bash set MSYS_NO_PATHCONV=1 so a leading / is not rewritten)", strings.Join(by, " and "), total)
}
