package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/serialization"
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
}

func NewConformanceService(ws *WorkspaceService, ss *SettingsService) *ConformanceService {
	return &ConformanceService{ctx: context.Background(), ws: ws, settings: ss}
}

func (s *ConformanceService) setContext(ctx context.Context) { s.ctx = ctx }

type ConformContextIO struct {
	Workspace    string      `json:"workspace"`
	Specs        []CliFileIO `json:"specs"`
	Environments []CliFileIO `json:"environments"`
}

// GetConformContext lists the workspace files the page offers.
func (s *ConformanceService) GetConformContext() ConformContextIO {
	ws := s.ws.GetWorkspacePath()
	out := ConformContextIO{Workspace: ws, Specs: []CliFileIO{}, Environments: []CliFileIO{}}
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
	// RecordCoverage adds the run to the Contracts coverage store. Only unfiltered runs are
	// recorded, so a run limited by tags or paths cannot skew the accumulated totals.
	RecordCoverage bool `json:"recordCoverage"`
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

type ConformOperationIO struct {
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Status      string   `json:"status"` // passed, failed, error, skipped
	StatusCode  int      `json:"statusCode"`
	DurationMs  int64    `json:"durationMs"`
	Violations  []string `json:"violations"`
	Error       string   `json:"error,omitempty"`
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
	// CoverageNote says what happened to RecordCoverage: recorded under which spec name, or why not.
	CoverageNote string `json:"coverageNote,omitempty"`
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

	ctx, cancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	s.cancelRun = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		s.cancelRun = nil
		s.mu.Unlock()
	}()

	out := application.RunConformance(ctx, application.ConformInput{
		Exec:             buildExecutor(s.settings),
		Spec:             spec,
		BaseURL:          baseURL,
		Env:              env,
		Tags:             in.Tags,
		Paths:            in.Paths,
		FailFast:         in.FailFast,
		RequestTimeoutMs: in.TimeoutMs,
		Coverage:         true,
		Negative:         in.Negative,
		ReadOnly:         !in.AllowWrites,
	})
	s.mu.Lock()
	s.last = &out
	s.mu.Unlock()

	res := conformOutputIO(out, baseURL, in.AllowWrites)
	if in.RecordCoverage {
		res.CoverageNote = s.recordCoverage(in, spec, baseURL, out)
	}
	return res
}

// recordCoverage adds an unfiltered run to the Contracts coverage store, under the spec's path
// relative to the workspace, and says what it did.
func (s *ConformanceService) recordCoverage(in ConformRunInput, spec *openapi.Spec, baseURL string, out application.ConformOutput) string {
	if len(in.Tags) > 0 || len(in.Paths) > 0 {
		return "Not added to coverage: a run limited by tags or paths would skew the totals."
	}
	ws := s.ws.GetWorkspacePath()
	if ws == "" {
		return "Not added to coverage: no workspace is open."
	}
	specName := in.SpecPath
	if rel, err := filepath.Rel(ws, in.SpecPath); err == nil && !strings.HasPrefix(rel, "..") {
		specName = filepath.ToSlash(rel)
	}
	var executed []openapi.ExecutedRequest
	for _, op := range out.Result.Operations {
		if op.Skipped || op.StatusCode == 0 {
			continue
		}
		executed = append(executed, openapi.ExecutedRequest{
			Method: op.Method, URL: baseURL + op.Path, StatusCode: op.StatusCode, Passed: op.Passed(),
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
		b, err := reporting.ConformJUnit(last.Result)
		if err != nil {
			return RunReportIO{Error: fmt.Sprintf("generating the report: %v", err)}
		}
		return RunReportIO{Content: string(b)}
	case "json":
		b, err := reporting.ConformJSON(last.Result, last.Coverage)
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
		SpecTitle:  r.SpecTitle,
		BaseURL:    baseURL,
		Writes:     writes,
		DurationMs: r.Duration.Milliseconds(),
		Operations: []ConformOperationIO{},
		Probes:     []ConformProbeIO{},
		Total:      len(r.Operations),
	}
	for _, op := range r.Operations {
		item := ConformOperationIO{
			OperationID: op.OperationID, Method: op.Method, Path: op.Path,
			StatusCode: op.StatusCode, DurationMs: op.DurationMs, Error: op.Error,
			Violations: []string{},
		}
		for _, v := range op.RequestViolations {
			item.Violations = append(item.Violations, "request: "+v.Error())
		}
		for _, v := range op.Violations {
			item.Violations = append(item.Violations, v.Error())
		}
		switch {
		case op.Skipped:
			item.Status = "skipped"
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
