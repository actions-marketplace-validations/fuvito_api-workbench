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
		Exec:     buildExecutor(s.settings),
		Spec:     spec,
		BaseURL:  baseURL,
		Env:      env,
		Coverage: true,
		ReadOnly: !in.AllowWrites,
	})
	return conformOutputIO(out, baseURL, in.AllowWrites)
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
	return res
}
