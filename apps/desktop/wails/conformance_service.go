package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/projectconfig"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// ConformanceService backs the Check conformance page. It is a thin wrapper over
// application.RunConformance, the same function the CLI uses.
type ConformanceService struct {
	ctx       context.Context
	ws        *WorkspaceService
	settings  *SettingsService
	mu        sync.Mutex
	cancelRun context.CancelFunc
	last      *application.ConformOutput // the latest run, for Export
	// lastFiltered is set when the latest run was limited by tags or path prefixes: such a run cannot
	// become a baseline, because it would drop the entries of the operations it left out.
	lastFiltered bool
	emit         func(event string, data ...interface{}) // nil outside the running app (tests)
}

func NewConformanceService(ws *WorkspaceService, ss *SettingsService) *ConformanceService {
	return &ConformanceService{ctx: context.Background(), ws: ws, settings: ss}
}

func (s *ConformanceService) setContext(ctx context.Context) {
	s.ctx = ctx
	s.emit = func(event string, data ...interface{}) { runtime.EventsEmit(ctx, event, data...) }
}

// ConformProgressEvent is emitted as "conformance:progress" after each operation of a run.
type ConformProgressEvent struct {
	Done      int    `json:"done"`
	Total     int    `json:"total"`
	Operation string `json:"operation"` // for example "GET /products"
}

type ConformContextIO struct {
	Workspace    string      `json:"workspace"`
	Specs        []CliFileIO `json:"specs"`
	Environments []CliFileIO `json:"environments"`
	// ActiveEnvironment is the file "Active environment" refers to ("" when none is selected), so
	// the page can put it in a command line.
	ActiveEnvironment string `json:"activeEnvironment"`
	// ProjectDefaults are the choices the workspace's clrsig.yaml makes for this page.
	ProjectDefaults ConformDefaultsIO `json:"projectDefaults"`
	// Scenarios are the workspace scenarios the page offers to run first (a login, for example).
	Scenarios []CliFileIO `json:"scenarios"`
	// Baselines are the workspace files that look like conformance baselines (their name says baseline).
	Baselines []CliFileIO `json:"baselines"`
}

// ConformDefaultsIO is what the page preselects from the workspace's clrsig.yaml. The file is read
// only from the workspace folder, and a path it names is used only when it is inside the workspace.
// conform.readOnly is deliberately not applied: the page sends only reads until the user ticks
// "Allow writes", whatever the file says.
type ConformDefaultsIO struct {
	Found bool   `json:"found"`
	File  string `json:"file,omitempty"`
	// Note says what was not applied and why (an unusable file, a path outside the workspace).
	Note string `json:"note,omitempty"`
	// Applied lists the keys that gave a value, as written in the file.
	Applied            []string `json:"applied"`
	SpecPath           string   `json:"specPath,omitempty"`
	EnvironmentPath    string   `json:"environmentPath,omitempty"`
	BaseURL            string   `json:"baseUrl,omitempty"`
	SetupPath          string   `json:"setupPath,omitempty"`
	BaselinePath       string   `json:"baselinePath,omitempty"`
	Tags               []string `json:"tags"`
	Paths              []string `json:"paths"`
	Negative           bool     `json:"negative"`
	FailOnWarning      bool     `json:"failOnWarning"`
	RequestTimeoutMs   int      `json:"requestTimeoutMs"`
	MaxResponseTimeMs  int      `json:"maxResponseTimeMs"`
	MaxResponseTimeFor []string `json:"maxResponseTimeFor"`
}

// projectDefaults reads the clrsig.yaml in the workspace folder for the page.
func projectDefaults(ws string) ConformDefaultsIO {
	out := ConformDefaultsIO{Applied: []string{}, Tags: []string{}, Paths: []string{}, MaxResponseTimeFor: []string{}}
	if ws == "" {
		return out
	}
	path, err := projectconfig.InFolder(ws)
	if err != nil {
		out.Found, out.Note = true, err.Error()
		return out
	}
	if path == "" {
		return out
	}
	out.Found, out.File = true, filepath.Base(path)
	cfg, err := projectconfig.Load(path)
	if err != nil {
		out.Note = err.Error()
		return out
	}
	var notes []string
	inside := func(key, value string) string {
		if value == "" || strings.Contains(value, "://") {
			return ""
		}
		full := value
		if !filepath.IsAbs(full) {
			full = filepath.Join(cfg.Dir(), filepath.FromSlash(value))
		}
		if !pathInside(ws, full) {
			notes = append(notes, key+" is outside the workspace, so it was not preselected")
			return ""
		}
		if _, err := os.Stat(full); err != nil {
			notes = append(notes, key+" does not exist, so it was not preselected")
			return ""
		}
		return full
	}
	apply := func(key string, set bool) {
		if set {
			out.Applied = append(out.Applied, key)
		}
	}
	if out.SpecPath = inside("spec", cfg.Spec); out.SpecPath != "" {
		apply("spec", true)
	}
	if out.EnvironmentPath = inside("environment", cfg.Environment); out.EnvironmentPath != "" {
		apply("environment", true)
	}
	if cfg.BaseURL != "" {
		out.BaseURL = cfg.BaseURL
		apply("baseUrl", true)
	}
	if c := cfg.Conform; c != nil {
		if len(c.Tags) > 0 {
			out.Tags = c.Tags
			apply("conform.tags", true)
		}
		if len(c.Paths) > 0 {
			out.Paths = c.Paths
			apply("conform.paths", true)
		}
		if c.Negative != nil {
			out.Negative = *c.Negative
			apply("conform.negative", *c.Negative)
		}
		if c.FailOn == "warning" {
			out.FailOnWarning = true
			apply("conform.failOn", true)
		}
		if c.RequestTimeoutMs != nil && *c.RequestTimeoutMs > 0 {
			out.RequestTimeoutMs = *c.RequestTimeoutMs
			apply("conform.requestTimeoutMs", true)
		}
		if c.MaxResponseTimeMs != nil && *c.MaxResponseTimeMs > 0 {
			out.MaxResponseTimeMs = *c.MaxResponseTimeMs
			apply("conform.maxResponseTimeMs", true)
		}
		if len(c.MaxResponseTimeFor) > 0 {
			out.MaxResponseTimeFor = c.MaxResponseTimeFor
			apply("conform.maxResponseTimeFor", true)
		}
		if out.BaselinePath = inside("conform.baseline", c.Baseline); out.BaselinePath != "" {
			apply("conform.baseline", true)
		}
		if out.SetupPath = inside("conform.setup", c.Setup); out.SetupPath != "" {
			apply("conform.setup", true)
		}
		if c.ReadOnly != nil && !*c.ReadOnly {
			notes = append(notes, "conform.readOnly: false is not applied; this page sends only reads until you tick Allow writes")
		}
	}
	out.Note = strings.Join(notes, "; ")
	return out
}

// GetConformContext lists the workspace files the page offers.
func (s *ConformanceService) GetConformContext() ConformContextIO {
	ws := s.ws.GetWorkspacePath()
	out := ConformContextIO{Workspace: ws, Specs: []CliFileIO{}, Environments: []CliFileIO{}, Scenarios: []CliFileIO{}, Baselines: []CliFileIO{}, ActiveEnvironment: s.ws.activeEnvPath, ProjectDefaults: projectDefaults(ws)}
	if ws != "" {
		out.Specs = findSpecs(ws)
		out.Environments = listFiles(ws, "environments")
		out.Scenarios = listFiles(ws, "scenarios")
		out.Baselines = findBaselines(ws)
	}
	return out
}

type ConformSpecInfoIO struct {
	Title     string `json:"title"`
	Version   string `json:"version"`
	ServerURL string `json:"serverUrl"`
	Reads     int    `json:"reads"`
	Writes    int    `json:"writes"`
	Error     string `json:"error,omitempty"`
	// Lint lists what is wrong with the spec document itself (the same findings as clrsig validate
	// --spec): worth fixing before reading response violations, which blame the server.
	Lint []SpecIssueIO `json:"lint"`
}

// SpecIssueIO is one finding about the spec document.
type SpecIssueIO struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	Where    string `json:"where"`
	What     string `json:"what"`
	Why      string `json:"why"`
	Fix      string `json:"fix"`
}

// InspectConformSpec reads a spec so the page can show its default base URL and how many
// operations would send a write request.
func (s *ConformanceService) InspectConformSpec(specPath string) ConformSpecInfoIO {
	if err := s.insideWorkspace("spec", specPath); err != nil {
		return ConformSpecInfoIO{Error: err.Error()}
	}
	spec, err := loadSpecFile(specPath)
	if err != nil {
		return ConformSpecInfoIO{Error: err.Error()}
	}
	title, version := spec.Info()
	info := ConformSpecInfoIO{Title: title, Version: version, ServerURL: spec.FirstServerURL(), Lint: []SpecIssueIO{}}
	for _, i := range spec.Lint() {
		info.Lint = append(info.Lint, SpecIssueIO{Check: i.Check, Severity: i.Severity, Where: i.Where, What: i.What, Why: i.Why, Fix: i.Fix})
	}
	for _, op := range spec.AllOperations() {
		if conformance.IsReadOnlyMethod(op.Method) {
			info.Reads++
		} else {
			info.Writes++
		}
	}
	return info
}

// insideWorkspace refuses a file the page was asked to read when it is not in the open workspace: the page
// only offers workspace files, so anything else is a request the interface did not make (CLAUDE.md
// section 7: file operations stay inside the project).
func (s *ConformanceService) insideWorkspace(what, path string) error {
	if path == "" {
		return nil
	}
	ws := s.ws.GetWorkspacePath()
	if ws == "" {
		return fmt.Errorf("open a workspace first: Clear Signal reads the %s only from the open workspace", what)
	}
	if !pathInside(ws, path) {
		return fmt.Errorf("the %s %q is outside the open workspace %q, and Clear Signal reads only files inside it; copy the file into the workspace (for example into specs/) and choose it there", what, path, ws)
	}
	return nil
}

func loadSpecFile(path string) (*openapi.Spec, error) {
	if path == "" {
		return nil, fmt.Errorf("choose a spec")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if format == "yml" {
		format = "yaml"
	}
	spec, err := openapi.ParseBytes(data, format)
	if err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	return spec, nil
}

type ConformRunInput struct {
	SpecPath string `json:"specPath"`
	BaseURL  string `json:"baseUrl"`
	EnvPath  string `json:"envPath"`
	// AllowWrites sends POST, PUT, PATCH and DELETE operations too. Off by default.
	AllowWrites bool `json:"allowWrites"`
	// ConfirmedRemote must be true to run against a base URL that is not on this computer.
	ConfirmedRemote bool `json:"confirmedRemote"`

	// Filters: operations outside them are skipped. Comma lists are split by the caller.
	Tags  []string `json:"tags"`
	Paths []string `json:"paths"`
	// Negative adds negative probes (malformed or unauthorized requests that should be rejected).
	Negative bool `json:"negative"`
	FailFast bool `json:"failFast"`
	// FailOnWarning makes a warning fail the run (clrsig conform --fail-on warning).
	FailOnWarning bool `json:"failOnWarning"`
	// TimeoutMs is the per-request timeout; 0 uses the shared default.
	TimeoutMs int `json:"timeoutMs"`
	// MaxResponseTimeMs fails an operation that answers more slowly than this; 0 = no limit.
	MaxResponseTimeMs int `json:"maxResponseTimeMs"`
	// MaxResponseTimeFor sets other limits for path prefixes, each as /prefix=ms.
	MaxResponseTimeFor []string `json:"maxResponseTimeFor"`
	// RecordCoverage adds the run to the Contracts coverage store. Only unfiltered runs are
	// recorded, so a run limited by tags or paths cannot skew the accumulated totals.
	RecordCoverage bool `json:"recordCoverage"`
	// Vars override or add environment variables for this run only, as name=value (the same as
	// `clrsig conform --var`). They are not saved and not part of the Get CLI command.
	Vars []string `json:"vars"`
	// BaselinePath is a baseline file in the workspace: violations recorded in it are accepted and
	// only new ones fail the run, as clrsig conform --baseline does.
	BaselinePath string `json:"baselinePath"`
	// SetupScenario is a workspace scenario that runs first (a login): what its steps extract is
	// available to the generated requests as {{name}}, like clrsig conform --setup.
	SetupScenario string `json:"setupScenario"`
}

type ConformProbeIO struct {
	OperationID string `json:"operationId"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Status      string `json:"status"` // rejected, accepted, error, skipped
	StatusCode  int    `json:"statusCode"`
	Error       string `json:"error,omitempty"`
}

// ConformFindingIO explains one violation: where, what the spec wanted, what the server sent, why
// that fails and what to change. Detail is the validator's own message.
type ConformFindingIO struct {
	Kind string `json:"kind"`
	// CheckID is the check's ID in the catalog (Help, What We Check), for example CS-RESP-005.
	CheckID string `json:"checkId,omitempty"`
	// Severity is failure, warning or note.
	Severity string `json:"severity,omitempty"`
	// Part and Where say where in the response (or request) the problem is: the header name, the
	// status, a body field. Spec says where the expectation comes from in the spec.
	Part     string `json:"part,omitempty"`
	Where    string `json:"where,omitempty"`
	Spec     string `json:"spec,omitempty"`
	Summary  string `json:"summary"`
	Expected string `json:"expected,omitempty"`
	Got      string `json:"got,omitempty"`
	Why      string `json:"why"`
	Fix      string `json:"fix"`
	Detail   string `json:"detail,omitempty"`
}

// findingsIO converts diagnosed findings for the frontend.
func findingsIO(fs []conformance.Finding) []ConformFindingIO {
	out := make([]ConformFindingIO, 0, len(fs))
	for _, f := range fs {
		out = append(out, ConformFindingIO{
			Kind: f.Kind, CheckID: f.CheckID, Severity: string(f.Severity), Part: f.Part, Where: f.Where, Spec: f.Spec, Summary: f.Summary, Expected: f.Expected,
			Got: f.Got, Why: f.Why, Fix: f.Fix, Detail: f.Detail,
		})
	}
	return out
}

type ConformOperationIO struct {
	OperationID string `json:"operationId"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	// SpecPath is the path as the spec writes it, which the page shows; Path is the generated
	// request's, as in the JSON report.
	SpecPath   string   `json:"specPath"`
	Status     string   `json:"status"` // passed, failed, error, blocked (not sent: a value is missing), skipped
	StatusCode int      `json:"statusCode"`
	DurationMs int64    `json:"durationMs"`
	Violations []string `json:"violations"`
	Error      string   `json:"error,omitempty"`
	// Findings explain the failure in plain language; empty for a passing or skipped operation.
	Findings []ConformFindingIO `json:"findings"`
	// Warnings counts the findings that are only warnings; a passing operation can have some.
	Warnings int `json:"warnings"`
}

type ConformRunOutputIO struct {
	Error      string               `json:"error,omitempty"`
	SpecTitle  string               `json:"specTitle"`
	BaseURL    string               `json:"baseUrl"`
	Writes     bool                 `json:"writes"`
	DurationMs int64                `json:"durationMs"`
	Operations []ConformOperationIO `json:"operations"`
	// The four totals: Covered + NotCovered = Total; Passing + Failing = Covered.
	Total      int `json:"total"`
	Covered    int `json:"covered"`
	NotCovered int `json:"notCovered"`
	Passing    int `json:"passing"`
	Failing    int `json:"failing"`
	// WithWarnings is how many passing operations have warnings.
	WithWarnings int `json:"withWarnings"`
	// Probes is empty unless negative probes were requested. An accepted probe means the server
	// took a request it should have rejected.
	Probes []ConformProbeIO `json:"probes"`
	// MissingValues are the placeholders that kept operations from being sent, each once, and
	// BlockedOperations how many operations that affected.
	MissingValues     []string `json:"missingValues"`
	BlockedOperations int      `json:"blockedOperations"`
	// Warnings are unset ${NAME} references in the environment: those values stay as written in
	// the requests, which usually makes them fail for the wrong reason.
	Warnings []string `json:"warnings"`
	// CoverageNote says what happened to RecordCoverage: recorded under which spec name, or why not.
	CoverageNote string `json:"coverageNote,omitempty"`
	// Failed is the run's verdict, the one the CLI turns into exit code 1: a violation, a failed
	// probe, an operation that was not sent or got no answer, or (with a baseline) a new violation.
	Failed bool `json:"failed"`
	// SetupNote says what the setup scenario did: which values it made available (never the values).
	SetupNote string `json:"setupNote,omitempty"`
	// Baseline is how the run compares with the baseline file, when one was chosen.
	Baseline *ConformBaselineIO `json:"baseline,omitempty"`
}

// IsLocalBaseURL reports whether the URL points at this computer.
func IsLocalBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// RunConformance runs the check. It never sends write requests unless AllowWrites is set, and
// refuses a non-local base URL that the user has not confirmed, including a URL that only came
// from the spec's servers.
func (s *ConformanceService) RunConformance(in ConformRunInput) ConformRunOutputIO {
	if err := s.insideWorkspace("spec", in.SpecPath); err != nil {
		return ConformRunOutputIO{Error: err.Error()}
	}
	if err := s.insideWorkspace("environment", in.EnvPath); err != nil {
		return ConformRunOutputIO{Error: err.Error()}
	}
	spec, err := loadSpecFile(in.SpecPath)
	if err != nil {
		return ConformRunOutputIO{Error: err.Error()}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if baseURL == "" {
		baseURL = spec.FirstServerURL()
	}
	if baseURL == "" {
		return ConformRunOutputIO{Error: "enter a base URL: the spec declares no server"}
	}
	if !IsLocalBaseURL(baseURL) && !in.ConfirmedRemote {
		return ConformRunOutputIO{Error: "confirm the base URL " + baseURL + " before running: it is not on this computer"}
	}

	env := s.ws.ActiveEnvironment()
	if in.EnvPath != "" {
		data, err := os.ReadFile(in.EnvPath)
		if err != nil {
			return ConformRunOutputIO{Error: "reading environment: " + err.Error()}
		}
		if env, err = serialization.ParseEnvironmentFile(data, filepath.Base(in.EnvPath)); err != nil {
			return ConformRunOutputIO{Error: "parsing environment: " + err.Error()}
		}
	}

	if len(in.Vars) > 0 {
		var err error
		if env, err = variables.WithOverrides(env, in.Vars); err != nil {
			return ConformRunOutputIO{Error: "variable overrides: " + err.Error()}
		}
	}

	pathLimits, err := conformance.ParsePathLimits(in.MaxResponseTimeFor)
	if err != nil {
		return ConformRunOutputIO{Error: "response-time limits: " + err.Error()}
	}

	var baseline *conformance.Baseline
	if in.BaselinePath != "" {
		if err := s.insideWorkspace("baseline", in.BaselinePath); err != nil {
			return ConformRunOutputIO{Error: err.Error()}
		}
		data, err := os.ReadFile(in.BaselinePath)
		if err != nil {
			return ConformRunOutputIO{Error: "reading the baseline: " + err.Error() + " (record one with Record as baseline after a run)"}
		}
		if baseline, err = conformance.ParseBaseline(data, in.BaselinePath); err != nil {
			return ConformRunOutputIO{Error: err.Error()}
		}
	}

	setupNote := ""
	if in.SetupScenario != "" {
		if err := s.insideWorkspace("setup scenario", in.SetupScenario); err != nil {
			return ConformRunOutputIO{Error: err.Error()}
		}
		var errText string
		if env, setupNote, errText = s.runSetup(in, env, baseURL); errText != "" {
			return ConformRunOutputIO{Error: errText}
		}
	}

	ctx, cancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	if s.cancelRun != nil {
		s.mu.Unlock()
		cancel()
		return ConformRunOutputIO{Error: "a conformance run is already in progress: cancel it or wait for it to finish"}
	}
	s.cancelRun = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		s.cancelRun = nil
		s.mu.Unlock()
	}()

	out := application.RunConformance(ctx, application.ConformInput{
		Exec:              buildExecutor(s.settings),
		Spec:              spec,
		BaseURL:           baseURL,
		Env:               env,
		Tags:              in.Tags,
		Paths:             in.Paths,
		FailFast:          in.FailFast,
		RequestTimeoutMs:  in.TimeoutMs,
		MaxResponseTimeMs: in.MaxResponseTimeMs,
		PathLimits:        pathLimits,
		Coverage:          true,
		Negative:          in.Negative,
		ReadOnly:          !in.AllowWrites,
		FailOn:            failOnFor(in.FailOnWarning),
		Baseline:          baseline,
		Progress: func(done, total int, op conformance.OperationResult) {
			if s.emit != nil {
				s.emit("conformance:progress", ConformProgressEvent{Done: done, Total: total, Operation: op.Method + " " + op.ReportPath()})
			}
		},
	})
	s.mu.Lock()
	s.last = &out
	s.lastFiltered = len(in.Tags) > 0 || len(in.Paths) > 0
	s.mu.Unlock()

	res := conformOutputIO(out, baseURL, in.AllowWrites)
	res.SetupNote = setupNote
	res.Baseline = baselineIO(in.BaselinePath, out.Diff)
	for _, ref := range variables.MissingEnv(env) {
		res.Warnings = append(res.Warnings, ref.Message())
	}
	if out.NothingSelected != "" {
		res.Warnings = append(res.Warnings, "Nothing was checked: "+out.NothingSelected)
	}
	if in.RecordCoverage {
		res.CoverageNote = s.recordCoverage(in, spec, baseURL, out)
	}
	return res
}

// recordCoverage adds an unfiltered run to the Contracts coverage store and says what it did. The
// store is keyed by the specFile string a scenario writes, so a scenario that already names this
// spec file decides the name (one Contracts card); otherwise the workspace-relative path is used.
func (s *ConformanceService) recordCoverage(in ConformRunInput, spec *openapi.Spec, baseURL string, out application.ConformOutput) string {
	if len(in.Tags) > 0 || len(in.Paths) > 0 {
		return "Not added to coverage: a run limited by tags or paths would skew the totals."
	}
	ws := s.ws.GetWorkspacePath()
	if ws == "" {
		return "Not added to coverage: no workspace is open."
	}
	specName := scenarioSpecRef(ws, in.SpecPath)
	if specName == "" {
		specName = in.SpecPath
		if rel, err := filepath.Rel(ws, in.SpecPath); err == nil && !outsideRel(rel) {
			specName = filepath.ToSlash(rel)
		}
	}
	var executed []openapi.ExecutedRequest
	for _, op := range out.Result.Operations {
		if op.Skipped || op.StatusCode == 0 {
			continue
		}
		executed = append(executed, openapi.ExecutedRequest{
			Method: op.Method, URL: baseURL + op.Path, StatusCode: op.StatusCode, Passed: out.Result.OpPassed(op), DurationMs: op.DurationMs,
		})
	}
	cr := spec.ScenarioCoverage(executed)
	if cr == nil {
		return "Not added to coverage: the spec could not be matched."
	}
	if _, ok := accumulateCoverage(ws, specName, spec, cr, !out.Failed); !ok {
		return "Not added to coverage: the coverage store could not be written."
	}
	return "Added to coverage as " + specName + ". Operations that were skipped or got no response were not counted."
}

// ExportConformReport renders the latest run with the same renderers as the CLI.
func (s *ConformanceService) ExportConformReport(format string) RunReportIO {
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if last == nil {
		return RunReportIO{Error: "there is no result to export; run the check first"}
	}
	switch format {
	case "html":
		b, err := reporting.ConformHTML(last.Result, last.Coverage, last.Negative)
		if err != nil {
			return RunReportIO{Error: fmt.Sprintf("generating the report: %v", err)}
		}
		return RunReportIO{Content: string(b)}
	case "junit":
		b, err := reporting.ConformJUnit(last.Result, last.Negative)
		if err != nil {
			return RunReportIO{Error: fmt.Sprintf("generating the report: %v", err)}
		}
		return RunReportIO{Content: string(b)}
	case "json":
		b, err := reporting.ConformJSON(last.Result, last.Coverage, last.Negative)
		if err != nil {
			return RunReportIO{Error: fmt.Sprintf("generating the report: %v", err)}
		}
		return RunReportIO{Content: string(b) + "\n"}
	case "markdown":
		return RunReportIO{Content: string(reporting.ConformMarkdown(last.Result, last.Coverage, last.Negative))}
	}
	return RunReportIO{Error: fmt.Sprintf("unknown report format %q (use html, junit, json or markdown)", format)}
}

// CancelConformance stops a run in progress.
func (s *ConformanceService) CancelConformance() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelRun != nil {
		s.cancelRun()
	}
}

func failOnFor(warning bool) checks.Severity {
	if warning {
		return checks.Warning
	}
	return checks.Failure
}

func conformOutputIO(out application.ConformOutput, baseURL string, writes bool) ConformRunOutputIO {
	r := out.Result
	res := ConformRunOutputIO{
		SpecTitle:     r.SpecTitle,
		BaseURL:       baseURL,
		Writes:        writes,
		DurationMs:    r.Duration.Milliseconds(),
		Operations:    []ConformOperationIO{},
		Probes:        []ConformProbeIO{},
		Warnings:      []string{},
		MissingValues: []string{},
		Total:         len(r.Operations),
		Failed:        out.Failed,
	}
	for _, op := range r.Operations {
		item := ConformOperationIO{
			OperationID: op.OperationID, Method: op.Method, Path: op.Path, SpecPath: op.ReportPath(),
			StatusCode: op.StatusCode, DurationMs: op.DurationMs, Error: op.Error,
			Violations: []string{},
			Findings:   []ConformFindingIO{},
		}
		item.Findings = append(item.Findings, findingsIO(reporting.Explain(op, r.Secrets))...)
		item.Warnings = len(op.Warnings())
		for _, v := range op.RequestViolations {
			item.Violations = append(item.Violations, "request: "+v.Error())
		}
		for _, v := range op.Violations {
			item.Violations = append(item.Violations, v.Error())
		}
		switch {
		case op.Skipped:
			item.Status = "skipped"
		case op.Blocked != "":
			item.Status = "blocked"
		case op.Error != "":
			item.Status = "error"
		case r.OpPassed(op):
			item.Status = "passed"
		default:
			item.Status = "failed"
		}
		// An operation is covered when it got a response; one that never got a response (skipped
		// or a network error) is not covered, as in the CLI's coverage report.
		if item.Status == "passed" || item.Status == "failed" {
			res.Covered++
			if item.Status == "passed" {
				res.Passing++
				if item.Warnings > 0 {
					res.WithWarnings++
				}
			} else {
				res.Failing++
			}
		} else {
			res.NotCovered++
		}
		res.Operations = append(res.Operations, item)
	}
	res.BlockedOperations, res.MissingValues = r.BlockedSummary()
	if res.MissingValues == nil {
		res.MissingValues = []string{}
	}
	if out.Negative != nil {
		for _, p := range out.Negative.Probes {
			item := ConformProbeIO{
				OperationID: p.OperationID, Method: p.Method, Path: p.Path,
				Kind: string(p.Kind), Description: p.Description, StatusCode: p.StatusCode, Error: p.Error,
			}
			switch {
			case p.Skipped:
				item.Status = "skipped"
			case p.Error != "":
				item.Status = "error"
			case p.Passed():
				item.Status = "rejected"
			default:
				item.Status = "accepted"
			}
			res.Probes = append(res.Probes, item)
		}
	}
	return res
}

// scenarioSpecRef returns the specFile string of the first scenario in the workspace whose
// (file path) specFile resolves to specPath, or "" when no scenario uses that file.
func scenarioSpecRef(ws, specPath string) string {
	target := filepath.Clean(specPath)
	found := ""
	_ = filepath.WalkDir(filepath.Join(ws, "scenarios"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || found != "" || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml", ".json":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		sc, err := serialization.ParseScenarioFile(data, filepath.Base(path))
		if err != nil || sc.SpecFile == "" || strings.Contains(sc.SpecFile, "://") {
			return nil
		}
		ref := sc.SpecFile
		if !filepath.IsAbs(ref) {
			ref = filepath.Join(filepath.Dir(path), ref)
		}
		if strings.EqualFold(filepath.Clean(ref), target) {
			found = sc.SpecFile
		}
		return nil
	})
	return found
}

// loadWorkspaceSpec loads the spec a Contracts entry is keyed by. The key is the specFile string
// exactly as the scenarios wrote it, and a relative one is relative to the scenario's own folder,
// so it is tried from the folder of each scenario that names it, then from the workspace root.
func loadWorkspaceSpec(ws, specRef string) (data []byte, format string, err error) {
	if strings.Contains(specRef, "://") || filepath.IsAbs(specRef) {
		return loadSpecBytes(specRef, "")
	}
	dirs := scenarioDirsUsing(ws, specRef)
	dirs = append(dirs, ws, filepath.Join(ws, "scenarios"))
	var firstErr error
	for _, dir := range dirs {
		data, format, err = loadSpecBytes(specRef, dir)
		if err == nil {
			return data, format, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, "", fmt.Errorf("spec file %q was not found relative to the scenarios that use it or to the workspace: %w", specRef, firstErr)
}

// scenarioDirsUsing lists the folders of the scenarios whose specFile is exactly specRef.
func scenarioDirsUsing(ws, specRef string) []string {
	var dirs []string
	seen := map[string]bool{}
	_ = filepath.WalkDir(filepath.Join(ws, "scenarios"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml", ".json":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		sc, err := serialization.ParseScenarioFile(data, filepath.Base(path))
		if err != nil || sc.SpecFile != specRef {
			return nil
		}
		if dir := filepath.Dir(path); !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
		return nil
	})
	return dirs
}

// runSetup runs the scenario chosen to run first and returns the environment the check should use.
// A setup that fails stops the run with its own report, as clrsig conform --setup does.
func (s *ConformanceService) runSetup(in ConformRunInput, env *domain.Environment, baseURL string) (*domain.Environment, string, string) {
	data, err := os.ReadFile(in.SetupScenario)
	if err != nil {
		return env, "", "reading the setup scenario: " + err.Error()
	}
	sc, err := serialization.ParseScenarioFile(data, filepath.Base(in.SetupScenario))
	if err != nil {
		return env, "", "parsing the setup scenario: " + err.Error()
	}
	if issues := scenario.Validate(sc, filepath.Dir(in.SetupScenario)); len(issues) > 0 {
		msgs := make([]string, len(issues))
		for i, iss := range issues {
			msgs[i] = iss.String()
		}
		return env, "", "the setup scenario is not valid: " + strings.Join(msgs, "; ")
	}
	setupEnv := env
	if baseURL != "" && !envHasVariable(env, "baseUrl") {
		withBase, _ := variables.WithOverrides(env, []string{"baseUrl=" + baseURL})
		setupEnv = withBase
	}
	keep := make([]string, 0, len(in.Vars))
	for _, v := range in.Vars {
		name, _, _ := strings.Cut(v, "=")
		keep = append(keep, name)
	}
	dir := filepath.Dir(in.SetupScenario)
	out := application.RunSetup(s.ctx, application.SetupInput{
		Exec: buildExecutor(s.settings), Scenario: sc, Env: setupEnv, Keep: keep,
		RefDirs: []string{dir, filepath.Join(dir, "..", "requests")},
	})
	label := filepath.Base(in.SetupScenario)
	if !out.Passed {
		return env, "", "The setup scenario " + label + " did not pass, so the check was not started:\n" + reporting.Text(out.Result)
	}
	if len(out.Names) == 0 {
		return env, "Setup scenario " + label + " passed; it extracted nothing.", ""
	}
	result := out.Env
	if setupEnv != env {
		// the {{baseUrl}} added for the setup alone stays out of the check's environment
		kept := make([]domain.EnvironmentVariable, 0, len(result.Variables))
		for _, v := range result.Variables {
			if v.Name == "baseUrl" && !envHasVariable(env, "baseUrl") {
				continue
			}
			kept = append(kept, v)
		}
		c := *result
		c.Variables = kept
		result = &c
	}
	return result, "Setup scenario " + label + " passed; values available to the check: " + strings.Join(out.Names, ", "), ""
}

func envHasVariable(env *domain.Environment, name string) bool {
	if env == nil {
		return false
	}
	for _, v := range env.Variables {
		if v.Name == name {
			return true
		}
	}
	return false
}

// ConformViolationIO is one violation in a baseline comparison. A version 1 baseline entry has no
// check, so Summary carries what it recorded.
type ConformViolationIO struct {
	Operation string `json:"operation"`
	Check     string `json:"check,omitempty"`
	Where     string `json:"where,omitempty"`
	Summary   string `json:"summary"`
}

// ConformBaselineIO says how a run compares with its baseline: the three lists of clrsig conform
// --baseline. Failed on the run output already accounts for it (only new violations fail).
type ConformBaselineIO struct {
	Path string `json:"path"`
	// Legacy is set for a version 1 file: it still works, and Record as baseline upgrades it.
	Legacy bool                 `json:"legacy"`
	Known  []ConformViolationIO `json:"known"`
	New    []ConformViolationIO `json:"new"`
	Fixed  []ConformViolationIO `json:"fixed"`
}

func baselineIO(path string, d *conformance.BaselineDiff) *ConformBaselineIO {
	if d == nil {
		return nil
	}
	conv := func(vs []conformance.Violation) []ConformViolationIO {
		out := []ConformViolationIO{}
		for _, v := range vs {
			item := ConformViolationIO{Operation: conformance.SpecStyle(v.Operation), Check: v.Check, Where: v.Where, Summary: v.Summary}
			if item.Summary == "" {
				item.Summary = v.Detail
				if i := strings.Index(item.Summary, "\n"); i >= 0 {
					item.Summary = item.Summary[:i]
				}
			}
			out = append(out, item)
		}
		return out
	}
	return &ConformBaselineIO{Path: path, Legacy: d.Legacy, Known: conv(d.Known), New: conv(d.New), Fixed: conv(d.Fixed)}
}

// findBaselines lists the workspace files whose name says they are a baseline (conform-baseline.yaml,
// baseline.json, ...), so the page can offer them.
func findBaselines(ws string) []CliFileIO {
	const maxDepth, maxFiles = 4, 20
	out := []CliFileIO{}
	_ = filepath.WalkDir(ws, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != ws && (skipDirs[d.Name()] || strings.Count(relLabel(ws, path), "/") >= maxDepth) {
				return fs.SkipDir
			}
			return nil
		}
		if len(out) < maxFiles && dataExts[strings.ToLower(filepath.Ext(path))] && strings.Contains(strings.ToLower(d.Name()), "baseline") {
			out = append(out, CliFileIO{Path: path, Label: relLabel(ws, path)})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// RecordedBaselineIO is the outcome of RecordConformBaseline.
type RecordedBaselineIO struct {
	Path       string `json:"path"`
	Violations int    `json:"violations"`
	Replaced   bool   `json:"replaced"`
	Error      string `json:"error,omitempty"`
}

// RecordConformBaseline writes the violations of the latest run to a baseline file in the workspace
// (version 2, atomically), as clrsig conform --write-baseline does. A run limited by tags or path
// prefixes cannot be recorded: it would drop the entries of the operations it left out. An existing
// file is replaced, so the caller asks first (the page lists the existing baselines).
func (s *ConformanceService) RecordConformBaseline(path string) RecordedBaselineIO {
	s.mu.Lock()
	last, filtered := s.last, s.lastFiltered
	s.mu.Unlock()
	switch {
	case last == nil:
		return RecordedBaselineIO{Error: "run the check first: a baseline records the violations of a run"}
	case filtered:
		return RecordedBaselineIO{Error: "this run was limited by tags or path prefixes, and a baseline recorded from it would drop the other operations: run the check without them, then record the baseline"}
	case path == "":
		return RecordedBaselineIO{Error: "choose a file name for the baseline"}
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
	default:
		return RecordedBaselineIO{Error: "the baseline file must be .yaml, .yml or .json"}
	}
	if err := s.insideWorkspace("baseline", path); err != nil {
		return RecordedBaselineIO{Error: err.Error()}
	}
	_, statErr := os.Stat(path)
	b := conformance.NewBaseline(last.Violations)
	if err := conformance.WriteBaselineFile(path, b); err != nil {
		return RecordedBaselineIO{Error: "writing the baseline: " + err.Error()}
	}
	return RecordedBaselineIO{Path: path, Violations: len(b.Violations), Replaced: statErr == nil}
}
