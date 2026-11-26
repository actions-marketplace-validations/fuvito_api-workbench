package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/redact"
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
	// BodySnippet is the start of the response body (at most 200 characters) when the server
	// answered 4xx or 5xx: the reason for a refusal is usually written there.
	BodySnippet string
	// Blocked is set when the request was not sent because a value it needs is missing: a
	// {{placeholder}} nothing provided, or a dynamic value that failed. Missing lists the
	// placeholder names. A blocked operation is neither passed nor skipped: it counts as failed, so
	// a check that could not run never looks like one that passed.
	Blocked string
	Missing []string
}

// Passed reports whether the operation executed and conformed to the spec. Findings whose check
// is only a warning or a note do not stop an operation from passing (see Warnings).
func (r OperationResult) Passed() bool {
	if r.Skipped || r.Blocked != "" || r.Error != "" {
		return false
	}
	if len(r.RequestViolations) == 0 && len(r.Violations) == 0 {
		return true
	}
	for _, f := range Diagnose(r) {
		if f.Severity == checks.Failure {
			return false
		}
	}
	return true
}

// Warnings returns the findings of a passing operation that are only warnings.
func (r OperationResult) Warnings() []Finding {
	if r.Skipped || len(r.RequestViolations) == 0 && len(r.Violations) == 0 {
		return nil
	}
	var out []Finding
	for _, f := range Diagnose(r) {
		if f.Severity == checks.Warning {
			out = append(out, f)
		}
	}
	return out
}

// OpPassed is Passed for this run: with FailOn set to warning, an operation with warnings has
// not passed. Reports and counts use it so they agree with the exit code.
func (r *ConformResult) OpPassed(op OperationResult) bool {
	if !op.Passed() {
		return false
	}
	return !(r.FailOn == checks.Warning && len(op.Warnings()) > 0)
}

// WarningCount is the number of operations that passed but have warnings (zero when warnings
// fail the run, since those operations are then counted as failed).
func (r *ConformResult) WarningCount() int {
	n := 0
	for _, op := range r.Operations {
		if r.OpPassed(op) && len(op.Warnings()) > 0 {
			n++
		}
	}
	return n
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

	// FilterMatched is how many operations the --tags and --paths filters selected (every
	// operation when there is no filter); it is 0 when a filter matches nothing, which a caller
	// must not report as a pass.
	FilterMatched int
	// Scope is set when the run was limited to the operations a change touched.
	Scope *Scope
	// BaselineApplied is set when the run was compared with a baseline; KnownViolations is how
	// many of the failures it accepted. A run with only known failures passes.
	BaselineApplied bool
	KnownViolations int
	BaselineClean   bool

	// FailOn is the lowest severity that fails the run: failure (the default) or warning.
	FailOn checks.Severity
}

// Counts returns (passed, failed, skipped) operation counts.
func (r *ConformResult) Counts() (passed, failed, skipped int) {
	for _, op := range r.Operations {
		switch {
		case op.Skipped:
			skipped++
		case r.OpPassed(op):
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
	// OnlyOperations restricts the run to these operations, each written METHOD path with the path as
	// the spec writes it (see openapi.OperationKey). Nil means no restriction; an empty map selects
	// nothing. It applies to requests and to negative probes, and combines with Tags and Paths.
	OnlyOperations map[string]bool
	// FailFast stops after the first failing operation.
	FailFast bool
	// RequestTimeoutMs overrides the per-request timeout. 0 = use executor default.
	RequestTimeoutMs int
	// MaxResponseTimeMs fails an operation whose response took longer than this many milliseconds.
	// The response is kept and compared with the spec as usual. 0 = no limit.
	MaxResponseTimeMs int
	// PathLimits sets a response-time limit for the operations under a path prefix (the longest
	// matching prefix wins over MaxResponseTimeMs). A limit of 0 under a prefix switches the limit
	// off for those operations.
	PathLimits map[string]int
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
		or := operationFor(gr, baseURL)

		selected, send := opts.selects(or, gr)
		if selected {
			result.FilterMatched++
		}
		if !send {
			or.Skipped = true
			result.Operations = append(result.Operations, or)
			handled()
			continue
		}

		resolved, missing, blocked := prepare(resolver, gr)
		if blocked != "" {
			or.Missing = missing
			or.Blocked = blocked
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
		if resp.StatusCode >= 400 {
			or.BodySnippet = redact.Excerpt(resp.BodyString(), 200)
		}
		or.Violations = openapi.ValidateResponseByPath(spec, string(gr.Request.Method), or.Path, resp.StatusCode, resp)
		if limit := opts.limitFor(or.SpecPath, or.Path); limit > 0 && resp.Duration.Milliseconds() > int64(limit) {
			// The message holds the limit, not the measured time, so a baseline stays stable
			// between runs; the time is in DurationMs and in the finding's Got.
			or.Violations = append(or.Violations, openapi.ValidationError{
				Kind: openapi.KindSlowResponse, Subject: strconv.Itoa(limit),
				Expected: fmt.Sprintf("within %d ms", limit), Got: fmt.Sprintf("%d ms", resp.Duration.Milliseconds()),
				Message: fmt.Sprintf("%s %s %d: the response was slower than the %d ms limit", or.Method, or.ReportPath(), resp.StatusCode, limit),
			})
		}

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

func matchesOnly(only map[string]bool, method, path string) bool {
	return only == nil || only[openapi.OperationKey(method, path)]
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

// limitFor returns the response-time limit for an operation path: the longest matching
// PathLimits prefix, otherwise MaxResponseTimeMs.
func (o Options) limitFor(specPath, sentPath string) int {
	best, limit := "", o.MaxResponseTimeMs
	for prefix, ms := range o.PathLimits {
		if hasPathPrefix(prefix, specPath, sentPath) && (best == "" || longerPrefix(prefix, best)) {
			best, limit = prefix, ms
		}
	}
	return limit
}

// longerPrefix orders the prefixes that match one operation: the one naming more of the path wins,
// however its parameters are written. Between two spellings of the same path the {{name}} one
// wins, which is the only one that used to match; the last rule just keeps the answer stable.
func longerPrefix(a, b string) bool {
	if x, y := len(SpecStyle(a)), len(SpecStyle(b)); x != y {
		return x > y
	}
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	return a > b
}

// hasPathPrefix says whether a prefix the user typed starts an operation's path. Both spellings of
// a parameter count: the spec's ({id}) and the generated request's ({{id}}).
func hasPathPrefix(prefix, specPath, sentPath string) bool {
	return strings.HasPrefix(specPath, prefix) || strings.HasPrefix(sentPath, prefix) || strings.HasPrefix(specPath, SpecStyle(prefix))
}

// ParsePathLimits reads repeated /prefix=ms values (for --max-response-time-for).
func ParsePathLimits(values []string) (map[string]int, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := map[string]int{}
	for _, v := range values {
		i := strings.LastIndex(v, "=")
		if i <= 0 || !strings.HasPrefix(v, "/") {
			return nil, fmt.Errorf("invalid time limit %q: use /path-prefix=milliseconds, for example /reports=2000", v)
		}
		ms, err := strconv.Atoi(strings.TrimSpace(v[i+1:]))
		if err != nil || ms < 0 {
			return nil, fmt.Errorf("invalid time limit %q: the milliseconds must be a whole number, 0 or more", v)
		}
		out[strings.TrimSpace(v[:i])] = ms
	}
	return out, nil
}

func matchesPathPrefix(specPath, sentPath string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if hasPathPrefix(p, specPath, sentPath) {
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
			Kind: openapi.KindRequestBody, Subject: field, In: "body", Expected: declared, Got: quotable(field, name, val), Variable: name,
			Message: fmt.Sprintf("%s %s: the request body field %q takes its value from {{%s}}, which is %q; that is not a valid JSON value (the spec declares %s)", template.Method, specPath, field, name, quotable(field, name, val), declared),
		})
	}
	return out
}

// quotable is an environment value as a message may quote it: masked when the field or the
// variable that holds it is named like a credential, or it looks like a token.
func quotable(field, variable, value string) string {
	if redact.IsSensitiveName(field) || redact.IsSensitiveName(variable) {
		return redact.Mask
	}
	return redact.Tokens(value)
}

// ReportPath is the path to show to a person: the one the spec writes (/users/{id}), so that every
// report names an operation the same way. Path (/users/{{id}}, as the request was generated) stays
// the identity that baselines, filters and stored history use.
func (or OperationResult) ReportPath() string {
	if or.SpecPath != "" {
		return or.SpecPath
	}
	return or.Path
}

var rePlaceholderSegment = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// SpecStyle writes a stored operation name such as "GET /users/{{id}}" the way the spec does,
// "GET /users/{id}", where only the name is left (a baseline entry, a history row). It is for
// display only.
func SpecStyle(operation string) string {
	return rePlaceholderSegment.ReplaceAllString(operation, "{$1}")
}

// Scope describes a run that was limited to the operations that changed since a git ref.
type Scope struct {
	ChangedAgainst string   `json:"changedAgainst"`
	Checked        []string `json:"checked"`
	Added          int      `json:"added"`
	Modified       int      `json:"modified"`
	Unchanged      int      `json:"unchanged"`
	Removed        []string `json:"removed,omitempty"`
	// Unattributed counts spec-level changes no operation owns; they do not widen the check.
	Unattributed int `json:"unattributed,omitempty"`
}

// Line is the one-line description shown in reports and on the terminal.
func (s *Scope) Line() string {
	if s == nil {
		return ""
	}
	if len(s.Checked) == 0 {
		return fmt.Sprintf("Changed against %s: no operation changed, so nothing was checked (%d unchanged%s).", s.ChangedAgainst, s.Unchanged, s.removedText())
	}
	line := fmt.Sprintf("Changed against %s: %d operation%s (%d added, %d modified) checked, %d unchanged and not checked%s.",
		s.ChangedAgainst, len(s.Checked), map[bool]string{true: "", false: "s"}[len(s.Checked) == 1], s.Added, s.Modified, s.Unchanged, s.removedText())
	if s.Unattributed > 0 {
		line += fmt.Sprintf(" %d spec-level change%s (for example servers or security) belong to no operation and do not widen the check.", s.Unattributed, map[bool]string{true: "", false: "s"}[s.Unattributed == 1])
	}
	return line
}

func (s *Scope) removedText() string {
	if len(s.Removed) == 0 {
		return ""
	}
	return fmt.Sprintf(", %d removed", len(s.Removed))
}
