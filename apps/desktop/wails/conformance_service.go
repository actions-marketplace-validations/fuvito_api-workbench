package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
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
	last      *application.ConformOutput              // the latest run, for Export
	emit      func(event string, data ...interface{}) // nil outside the running app (tests)
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
}

// GetConformContext lists the workspace files the page offers.
func (s *ConformanceService) GetConformContext() ConformContextIO {
	ws := s.ws.GetWorkspacePath()
	out := ConformContextIO{Workspace: ws, Specs: []CliFileIO{}, Environments: []CliFileIO{}, ActiveEnvironment: s.ws.activeEnvPath}
	if ws != "" {
		out.Specs = findSpecs(ws)
		out.Environments = listFiles(ws, "environments")
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
	info := ConformSpecInfoIO{Title: title, Version: version, ServerURL: spec.FirstServerURL()}
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
			Kind: f.Kind, CheckID: f.CheckID, Part: f.Part, Where: f.Where, Spec: f.Spec, Summary: f.Summary, Expected: f.Expected,
			Got: f.Got, Why: f.Why, Fix: f.Fix, Detail: f.Detail,
		})
	}
	return out
}

type ConformOperationIO struct {
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Status      string   `json:"status"` // passed, failed, error, blocked (not sent: a value is missing), skipped
	StatusCode  int      `json:"statusCode"`
	DurationMs  int64    `json:"durationMs"`
	Violations  []string `json:"violations"`
	Error       string   `json:"error,omitempty"`
	// Findings explain the failure in plain language; empty for a passing or skipped operation.
	Findings []ConformFindingIO `json:"findings"`
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
		Progress: func(done, total int, op conformance.OperationResult) {
			if s.emit != nil {
				s.emit("conformance:progress", ConformProgressEvent{Done: done, Total: total, Operation: op.Method + " " + op.Path})
			}
		},
	})
	s.mu.Lock()
	s.last = &out
	s.mu.Unlock()

	res := conformOutputIO(out, baseURL, in.AllowWrites)
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
			Method: op.Method, URL: baseURL + op.Path, StatusCode: op.StatusCode, Passed: op.Passed(), DurationMs: op.DurationMs,
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
			OperationID: op.OperationID, Method: op.Method, Path: op.Path,
			StatusCode: op.StatusCode, DurationMs: op.DurationMs, Error: op.Error,
			Violations: []string{},
			Findings:   []ConformFindingIO{},
		}
		item.Findings = append(item.Findings, findingsIO(conformance.Diagnose(op))...)
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
		case op.Passed():
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
