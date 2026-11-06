package conformance

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// OperationResult holds the outcome of running and validating one spec operation.
type OperationResult struct {
	OperationID string
	Summary     string
	Tags        []string
	Method      string
	Path        string
	StatusCode  int
	DurationMs  int64
	Violations  []openapi.ValidationError
	Error       string
	Skipped     bool
}

// Passed reports whether the operation executed and conformed to the spec.
func (r OperationResult) Passed() bool {
	return !r.Skipped && r.Error == "" && len(r.Violations) == 0
}

// ConformResult holds the aggregate outcome of a conformance run.
type ConformResult struct {
	SpecTitle   string
	SpecVersion string
	BaseURL     string
	Operations  []OperationResult
	StartedAt   time.Time
	Duration    time.Duration
}

// Counts returns (passed, failed, skipped) operation counts.
func (r *ConformResult) Counts() (passed, failed, skipped int) {
	for _, op := range r.Operations {
		switch {
		case op.Skipped:
			skipped++
		case op.Passed():
			passed++
		default:
			failed++
		}
	}
	return
}

// CoverageReport summarises how many spec operations were exercised in a run.
// An operation is "covered" if it received an HTTP response (StatusCode > 0).
// Skipped operations and execution errors that produced no response are "not covered".
type CoverageReport struct {
	Total         int
	Covered       int
	NotCovered    int
	Percent       float64
	CoveredOps    []OperationResult
	NotCoveredOps []OperationResult
}

// ComputeCoverage derives operation coverage from a conformance result.
func ComputeCoverage(result *ConformResult) *CoverageReport {
	cr := &CoverageReport{Total: len(result.Operations)}
	for _, op := range result.Operations {
		if op.StatusCode > 0 {
			cr.Covered++
			cr.CoveredOps = append(cr.CoveredOps, op)
		} else {
			cr.NotCovered++
			cr.NotCoveredOps = append(cr.NotCoveredOps, op)
		}
	}
	if cr.Total > 0 {
		cr.Percent = float64(cr.Covered) / float64(cr.Total) * 100
	}
	return cr
}

// Options controls which operations are run and how failures are handled.
type Options struct {
	// Tags filters operations to those tagged with at least one of the given tags.
	// Empty means all tags are included.
	Tags []string
	// Paths filters operations to those whose path starts with one of the given prefixes.
	// Empty means all paths are included.
	Paths []string
	// FailFast stops after the first failing operation.
	FailFast bool
	// RequestTimeoutMs overrides the per-request timeout. 0 = use executor default.
	RequestTimeoutMs int
}

// Runner executes all operations in an OpenAPI spec against a live server
// and validates each response against the spec's declared response schema.
type Runner struct {
	exec *executor.Executor
}

// New creates a Runner using the given executor.
func New(exec *executor.Executor) *Runner {
	return &Runner{exec: exec}
}

// Run executes matching operations and returns the aggregate conformance result.
func (r *Runner) Run(ctx context.Context, spec *openapi.Spec, baseURL string, env *domain.Environment, opts Options) *ConformResult {
	title, version := spec.Info()
	result := &ConformResult{
		SpecTitle:   title,
		SpecVersion: version,
		BaseURL:     baseURL,
		StartedAt:   time.Now(),
	}

	resolver := variables.NewResolver(env)
	reqs, err := openapi.GenerateRequests(spec, baseURL)
	if err != nil {
		result.Duration = time.Since(result.StartedAt)
		return result
	}

	for _, gr := range reqs {
		opPath := pathOf(gr.Request.URL, baseURL)
		or := OperationResult{
			OperationID: gr.OperationID,
			Summary:     gr.Summary,
			Tags:        gr.Tags,
			Method:      string(gr.Request.Method),
			Path:        opPath,
		}

		if !matchesTags(gr.Tags, opts.Tags) || !matchesPathPrefix(opPath, opts.Paths) {
			or.Skipped = true
			result.Operations = append(result.Operations, or)
			continue
		}

		resolved := resolver.InterpolateRequest(gr.Request)
		if opts.RequestTimeoutMs > 0 {
			resolved.Timeout = time.Duration(opts.RequestTimeoutMs) * time.Millisecond
		}

		resp, execErr := r.exec.Execute(ctx, resolved)
		if execErr != nil {
			or.Error = execErr.Error()
			result.Operations = append(result.Operations, or)
			if opts.FailFast {
				break
			}
			continue
		}

		or.StatusCode = resp.StatusCode
		or.DurationMs = resp.Duration.Milliseconds()
		or.Violations = openapi.ValidateResponseByPath(spec, string(gr.Request.Method), opPath, resp.StatusCode, resp)

		result.Operations = append(result.Operations, or)

		if !or.Passed() && opts.FailFast {
			break
		}
	}

	result.Duration = time.Since(result.StartedAt)
	return result
}

func pathOf(fullURL, baseURL string) string {
	if baseURL != "" {
		trimmed := strings.TrimRight(baseURL, "/")
		if strings.HasPrefix(fullURL, trimmed) {
			p := fullURL[len(trimmed):]
			if p == "" {
				return "/"
			}
			// strip query string
			if idx := strings.Index(p, "?"); idx >= 0 {
				p = p[:idx]
			}
			return p
		}
	}
	u, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}
	return u.Path
}

func matchesTags(opTags, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	for _, ft := range filterTags {
		for _, ot := range opTags {
			if strings.EqualFold(ot, ft) {
				return true
			}
		}
	}
	return false
}

func matchesPathPrefix(opPath string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if strings.HasPrefix(opPath, p) {
			return true
		}
	}
	return false
}
