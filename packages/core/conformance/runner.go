package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
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
	// SpecPath is the path as the spec writes it (for example /products/{id}); Path is the path of
	// the request that was sent, with its placeholders. Findings point at SpecPath.
	SpecPath          string
	StatusCode        int
	DurationMs        int64
	RequestViolations []openapi.ValidationError
	Violations        []openapi.ValidationError
	Error             string
	Skipped           bool
	// Blocked is set when the request was not sent because a value it needs is missing: a
	// {{placeholder}} nothing provided, or a dynamic value that failed. Missing lists the
	// placeholder names. A blocked operation is neither passed nor skipped: it counts as failed, so
	// a check that could not run never looks like one that passed.
	Blocked string
	Missing []string
}

// Passed reports whether the operation executed and conformed to the spec.
func (r OperationResult) Passed() bool {
	return !r.Skipped && r.Blocked == "" && r.Error == "" && len(r.RequestViolations) == 0 && len(r.Violations) == 0
}

// BlockedSummary reports how many operations were not sent because a value was missing, and the
// placeholder names behind that (each once, in the order they were first needed), so a caller can
// say "set these once" instead of repeating the same names for every operation.
func (r *ConformResult) BlockedSummary() (operations int, missing []string) {
	seen := map[string]bool{}
	for _, op := range r.Operations {
		if op.Blocked == "" {
			continue
		}
		operations++
		for _, name := range op.Missing {
			if !seen[name] {
				seen[name] = true
				missing = append(missing, name)
			}
		}
	}
	return operations, missing
}

// MissingValuesAdvice is the one-line hint shown under a report that has blocked operations.
func MissingValuesAdvice(operations int, missing []string) string {
	if operations == 0 {
		return ""
	}
	plural := "operations were"
	if operations == 1 {
		plural = "operation was"
	}
	if len(missing) == 0 {
		return fmt.Sprintf("%d %s not sent because a dynamic value failed (see above).", operations, plural)
	}
	return fmt.Sprintf("%d %s not sent because nothing provides: %s. Add them to the environment, or pass them as overrides (name=value).",
		operations, plural, strings.Join(missing, ", "))
}

// ConformResult holds the aggregate outcome of a conformance run.
type ConformResult struct {
	SpecTitle   string
	SpecVersion string
	BaseURL     string
	Operations  []OperationResult
	StartedAt   time.Time
	Duration    time.Duration

	// Secrets are the resolved values of the environment variables marked secret; reports
	// replace them wherever they would otherwise be printed (for example inside an error that
	// quotes the request URL).
	Secrets []string
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
	// ReadOnly runs only GET and HEAD operations; every other operation is reported as skipped
	// and never sent.
	ReadOnly bool
	// OnOperation, when set, is called after each operation has been handled (run, skipped or
	// failed to send) with how many are done out of how many, so a caller can show progress.
	OnOperation func(done, total int, op OperationResult)
}

// IsReadOnlyMethod reports whether an HTTP method cannot change server state (GET, HEAD).
func IsReadOnlyMethod(method string) bool {
	m := strings.ToUpper(method)
	return m == "GET" || m == "HEAD"
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
	if baseURL == "" {
		baseURL = spec.FirstServerURL()
	}
	result := &ConformResult{
		SpecTitle:   title,
		SpecVersion: version,
		BaseURL:     baseURL,
		StartedAt:   time.Now(),
		Secrets:     variables.SecretValues(env),
	}

	resolver := variables.NewResolver(env)
	reqs, err := openapi.GenerateRequests(spec, baseURL)
	if err != nil {
		result.Duration = time.Since(result.StartedAt)
		return result
	}

	handled := func() {
		if opts.OnOperation != nil {
			opts.OnOperation(len(result.Operations), len(reqs), result.Operations[len(result.Operations)-1])
		}
	}

	for _, gr := range reqs {
		opPath := pathOf(gr.Request.URL, baseURL)
		or := OperationResult{
			OperationID: gr.OperationID,
			Summary:     gr.Summary,
			Tags:        gr.Tags,
			Method:      string(gr.Request.Method),
			Path:        opPath,
			SpecPath:    gr.Path,
		}

		if !matchesTags(gr.Tags, opts.Tags) || !matchesPathPrefix(opPath, opts.Paths) ||
			(opts.ReadOnly && !IsReadOnlyMethod(or.Method)) {
			or.Skipped = true
			result.Operations = append(result.Operations, or)
			handled()
			continue
		}

		resolved := resolver.InterpolateRequest(gr.Request)
		if missing, dynamicErrors := UnresolvedPlaceholders(resolved), resolver.TakeErrors(); len(missing) > 0 || len(dynamicErrors) > 0 {
			or.Missing = missing
			or.Blocked = blockedReason(missing, dynamicErrors)
			result.Operations = append(result.Operations, or)
			handled()
			if opts.FailFast {
				break
			}
			continue
		}
		if opts.RequestTimeoutMs > 0 {
			resolved.Timeout = time.Duration(opts.RequestTimeoutMs) * time.Millisecond
		}

		// Skip request validation when path parameters are still unresolved.
		// An unresolved {{placeholder}} in the URL means the request will fail
		// for the wrong reason; the server error should not be attributed to us.
		if !strings.Contains(resolved.URL, "{{") {
			if bad := invalidBodyValues(gr.Request, resolved, resolver, spec, gr.Path); len(bad) > 0 {
				or.RequestViolations = bad
			} else {
				or.RequestViolations = openapi.ValidateRequest(spec, resolved)
			}
		}

		resp, execErr := r.exec.Execute(ctx, resolved)
		if execErr != nil {
			or.Error = execErr.Error()
			result.Operations = append(result.Operations, or)
			handled()
			if opts.FailFast {
				break
			}
			continue
		}

		or.StatusCode = resp.StatusCode
		or.DurationMs = resp.Duration.Milliseconds()
		or.Violations = openapi.ValidateResponseByPath(spec, string(gr.Request.Method), opPath, resp.StatusCode, resp)

		result.Operations = append(result.Operations, or)
		handled()

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

var bareBodyPlaceholder = regexp.MustCompile(`"([^"]+)"\s*:\s*\{\{\s*([A-Za-z_][\w.-]*)\s*\}\}`)

// invalidBodyValues explains a JSON body that stopped being JSON because an environment value was
// put in unquoted (a number or boolean field written as text, "cheap" for a price). The library
// only says "invalid character 'c'"; this names the field, the variable, the value and the type the
// spec declares. It returns nothing when the body is valid or no placeholder explains the problem.
func invalidBodyValues(template, resolved *domain.Request, resolver *variables.Resolver, spec *openapi.Spec, specPath string) []openapi.ValidationError {
	if template.Body == nil || resolved.Body == nil || template.Body.Type != domain.BodyJSON || json.Valid([]byte(resolved.Body.Content)) {
		return nil
	}
	var out []openapi.ValidationError
	for _, m := range bareBodyPlaceholder.FindAllStringSubmatch(template.Body.Content, -1) {
		field, name := m[1], m[2]
		if strings.HasPrefix(name, "$") {
			continue
		}
		val := resolver.Interpolate("{{" + name + "}}")
		if json.Valid([]byte(val)) {
			continue
		}
		declared := spec.BodyFieldType(string(template.Method), specPath, field)
		if declared == "" {
			declared = "a valid JSON value"
		} else {
			declared = "a " + declared
			if declared == "a integer" {
				declared = "an integer"
			}
		}
		out = append(out, openapi.ValidationError{
			Kind: openapi.KindRequestBody, Subject: field, In: "body", Expected: declared, Got: val, Variable: name,
			Message: fmt.Sprintf("%s %s: the request body field %q takes its value from {{%s}}, which is %q; that is not a valid JSON value (the spec declares %s)", template.Method, specPath, field, name, val, declared),
		})
	}
	return out
}
