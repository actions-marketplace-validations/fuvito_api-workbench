package conformance

import (
	"context"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// NegativeProbeResult holds the outcome of a single negative probe execution.
type NegativeProbeResult struct {
	OperationID string
	Method      string
	Path        string
	Kind        openapi.ProbeKind
	Description string
	StatusCode  int
	DurationMs  int64
	Error       string
	Skipped     bool
}

// Passed reports whether the server correctly rejected the malformed/unauthorized
// request with a 4xx status code.
func (r NegativeProbeResult) Passed() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// NegativeResult holds the aggregate outcome of a negative conformance run.
type NegativeResult struct {
	SpecTitle   string
	SpecVersion string
	BaseURL     string
	Probes      []NegativeProbeResult
	StartedAt   time.Time
	Duration    time.Duration

	// Secrets: see ConformResult.Secrets.
	Secrets []string
}

// Counts returns (passed, failed, skipped, errored) probe counts.
// failed = server returned 2xx/3xx/5xx when it should have returned 4xx.
func (r *NegativeResult) Counts() (passed, failed, skipped, errored int) {
	for _, p := range r.Probes {
		switch {
		case p.Skipped:
			skipped++
		case p.Error != "":
			errored++
		case p.Passed():
			passed++
		default:
			failed++
		}
	}
	return
}

// NegativeRunner executes negative probes against a live server and reports
// which malformed or unauthorized requests the server incorrectly accepts.
type NegativeRunner struct {
	exec *executor.Executor
}

// NewNegative creates a NegativeRunner using the given executor.
func NewNegative(exec *executor.Executor) *NegativeRunner {
	return &NegativeRunner{exec: exec}
}

// Run generates and executes all negative probes derived from the spec.
func (r *NegativeRunner) Run(ctx context.Context, spec *openapi.Spec, baseURL string, env *domain.Environment, opts Options) *NegativeResult {
	title, version := spec.Info()
	result := &NegativeResult{
		SpecTitle:   title,
		SpecVersion: version,
		BaseURL:     baseURL,
		StartedAt:   time.Now(),
		Secrets:     variables.SecretValues(env),
	}

	probes, err := openapi.GenerateNegativeProbes(spec, baseURL)
	if err != nil {
		result.Duration = time.Since(result.StartedAt)
		return result
	}

	resolver := variables.NewResolver(env)

	for _, probe := range probes {
		pr := NegativeProbeResult{
			OperationID: probe.OperationID,
			Method:      probe.Method,
			Path:        probe.Path,
			Kind:        probe.Kind,
			Description: probe.Description,
		}

		resolved := resolver.InterpolateRequest(probe.Request)

		// Skip probes where path parameters are still unresolved. An unresolved
		// placeholder in the URL causes a network error that masks the actual
		// rejection behaviour we are trying to observe.
		if strings.Contains(resolved.URL, "{{") {
			pr.Skipped = true
			result.Probes = append(result.Probes, pr)
			continue
		}

		if opts.RequestTimeoutMs > 0 {
			resolved.Timeout = time.Duration(opts.RequestTimeoutMs) * time.Millisecond
		}

		resp, execErr := r.exec.Execute(ctx, resolved)
		if execErr != nil {
			pr.Error = execErr.Error()
			result.Probes = append(result.Probes, pr)
			if opts.FailFast {
				break
			}
			continue
		}

		pr.StatusCode = resp.StatusCode
		pr.DurationMs = resp.Duration.Milliseconds()
		result.Probes = append(result.Probes, pr)

		if !pr.Passed() && opts.FailFast {
			break
		}
	}

	result.Duration = time.Since(result.StartedAt)
	return result
}
