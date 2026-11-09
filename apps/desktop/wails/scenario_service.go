package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// ---- I/O types ----

type ScenarioInfoIO struct {
	Title     string `json:"title"`
	Path      string `json:"path"`
	StepCount int    `json:"stepCount"`
}

type KVPairIO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AssertionDetailIO struct {
	Type     string      `json:"type"`
	Path     string      `json:"path,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Expected interface{} `json:"expected,omitempty"`
	Schema   string      `json:"schema,omitempty"`
}

type ExtractionDetailIO struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
}

type StepDetailIO struct {
	Title             string               `json:"title"`
	Description       string               `json:"description,omitempty"`
	Disabled          bool                 `json:"disabled,omitempty"`
	Method            string               `json:"method,omitempty"`
	URL               string               `json:"url,omitempty"`
	Headers           []KVPairIO           `json:"headers"`
	QueryParams       []KVPairIO           `json:"queryParams"`
	BodyType          string               `json:"bodyType,omitempty"`
	BodyContent       string               `json:"bodyContent,omitempty"`
	RequestRef        string               `json:"requestRef,omitempty"`
	Assertions        []AssertionDetailIO  `json:"assertions"`
	Extractions       []ExtractionDetailIO `json:"extractions"`
	FailPolicy        string               `json:"failPolicy,omitempty"`
	TimeoutMs         int                  `json:"timeoutMs,omitempty"`
	RetryMaxAttempts  int                  `json:"retryMaxAttempts,omitempty"`
	RetryDelayMs      int                  `json:"retryDelayMs,omitempty"`
}

type ScenarioDetailIO struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Steps       []StepDetailIO `json:"steps"`
	Error       string         `json:"error,omitempty"`
}

type AssertionResultIO struct {
	Type    string `json:"type"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type StepResultIO struct {
	Title           string              `json:"title"`
	Passed          bool                `json:"passed"`
	Skipped         bool                `json:"skipped"`
	StatusCode      int                 `json:"statusCode"`
	DurationMs      int64               `json:"durationMs"`
	Error           string              `json:"error,omitempty"`
	Extracted       map[string]string   `json:"extracted"`
	Assertions      []AssertionResultIO `json:"assertions"`
	ResponseBody    string              `json:"responseBody,omitempty"`
	ResponseHeaders []HeaderIO          `json:"responseHeaders,omitempty"`
}

type ScenarioResultIO struct {
	Title      string         `json:"title"`
	Passed     bool           `json:"passed"`
	DurationMs int64          `json:"durationMs"`
	Steps      []StepResultIO `json:"steps"`
	Error      string         `json:"error,omitempty"`
}

// ---- Service ----

type ScenarioService struct {
	ctx       context.Context
	workspace *WorkspaceService
	settings  *SettingsService
}

func NewScenarioService(ws *WorkspaceService, ss *SettingsService) *ScenarioService {
	return &ScenarioService{ctx: context.Background(), workspace: ws, settings: ss}
}

func (s *ScenarioService) setContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *ScenarioService) resultCachePath(scenarioPath string) (string, error) {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return "", fmt.Errorf("no workspace open")
	}
	rel, err := filepath.Rel(wsPath, scenarioPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside the current workspace")
	}
	base := strings.TrimSuffix(strings.TrimSuffix(rel, ".yaml"), ".yml")
	return filepath.Join(wsPath, ".clrsigcache", base+".result.json"), nil
}

// SaveScenarioResult persists the last run result for a scenario file.
// Returns "" on success or an error message.
func (s *ScenarioService) SaveScenarioResult(scenarioPath string, result ScenarioResultIO) string {
	path, err := s.resultCachePath(scenarioPath)
	if err != nil {
		return err.Error()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err.Error()
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err.Error()
	}
	if err := atomicWrite(path, data); err != nil {
		return err.Error()
	}
	return ""
}

// LoadScenarioResult returns the last cached run result for a scenario file, or nil if none exists.
func (s *ScenarioService) LoadScenarioResult(scenarioPath string) *ScenarioResultIO {
	path, err := s.resultCachePath(scenarioPath)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var result ScenarioResultIO
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return &result
}

// ListScenarios returns all scenario files under {workspace}/scenarios/.
func (s *ScenarioService) ListScenarios() []ScenarioInfoIO {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return nil
	}
	scenDir := filepath.Join(wsPath, "scenarios")
	var out []ScenarioInfoIO
	_ = filepath.WalkDir(scenDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		info := ScenarioInfoIO{Title: name, Path: p}
		if data, err := os.ReadFile(p); err == nil {
			if sc, err := serialization.ParseScenarioFile(data, d.Name()); err == nil {
				info.StepCount = len(sc.Steps)
			}
		}
		out = append(out, info)
		return nil
	})
	return out
}

// LoadScenario parses a scenario file and returns its structure for the UI.
func (s *ScenarioService) LoadScenario(path string) ScenarioDetailIO {
	data, err := os.ReadFile(path)
	if err != nil {
		return ScenarioDetailIO{Path: path, Error: err.Error()}
	}
	sc, err := serialization.ParseScenarioFile(data, filepath.Base(path))
	if err != nil {
		return ScenarioDetailIO{Path: path, Error: err.Error()}
	}
	detail := ScenarioDetailIO{
		Path:        path,
		Title:       sc.Title,
		Description: sc.Description,
	}
	for _, step := range sc.Steps {
		sd := StepDetailIO{
			Title:       step.Title,
			Description: step.Description,
			Disabled:    step.Disabled,
			RequestRef:  step.RequestRef,
			FailPolicy:  string(step.FailPolicy),
			TimeoutMs:   step.TimeoutMs,
			Headers:     []KVPairIO{},
			QueryParams: []KVPairIO{},
			Assertions:  []AssertionDetailIO{},
			Extractions: []ExtractionDetailIO{},
		}
		if step.Retry != nil && step.Retry.MaxAttempts > 0 {
			sd.RetryMaxAttempts = step.Retry.MaxAttempts
			sd.RetryDelayMs = int(step.Retry.Delay.Milliseconds())
		}
		if step.Request != nil {
			sd.Method = string(step.Request.Method)
			sd.URL = step.Request.URL
			for _, h := range step.Request.Headers {
				sd.Headers = append(sd.Headers, KVPairIO{Name: h.Name, Value: h.Value})
			}
			for _, q := range step.Request.QueryParams {
				sd.QueryParams = append(sd.QueryParams, KVPairIO{Name: q.Name, Value: q.Value})
			}
			if step.Request.Body != nil && step.Request.Body.Type != "" {
				sd.BodyType = string(step.Request.Body.Type)
				sd.BodyContent = step.Request.Body.Content
			}
		}
		for _, a := range step.Assertions {
			sd.Assertions = append(sd.Assertions, AssertionDetailIO{
				Type:     string(a.Type),
				Path:     a.Path,
				Operator: string(a.Operator),
				Expected: a.Expected,
				Schema:   a.Schema,
			})
		}
		for _, e := range step.Extractions {
			sd.Extractions = append(sd.Extractions, ExtractionDetailIO{
				Name:   e.Name,
				Source: string(e.Source),
				Path:   e.Path,
			})
		}
		detail.Steps = append(detail.Steps, sd)
	}
	return detail
}

// LoadScenarioRaw returns the raw file content of a scenario file as a string.
func (s *ScenarioService) LoadScenarioRaw(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// SaveScenarioRaw atomically writes content back to a scenario file.
// Returns an error message on failure, or "" on success.
func (s *ScenarioService) SaveScenarioRaw(path string, content string) string {
	if err := atomicWrite(path, []byte(content)); err != nil {
		return err.Error()
	}
	return ""
}

// CreateScenarioResultIO is the return type for CreateScenario.
type CreateScenarioResultIO struct {
	Path  string `json:"path"`
	Error string `json:"error,omitempty"`
}

// CreateScenario creates a starter scenario YAML file in {workspace}/scenarios/
// using the provided name. Returns the new file path, or an error message.
func (s *ScenarioService) CreateScenario(name string) CreateScenarioResultIO {
	if name == "" {
		return CreateScenarioResultIO{Error: "name is required"}
	}
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return CreateScenarioResultIO{Error: "no workspace open"}
	}

	scenDir := filepath.Join(wsPath, "scenarios")
	if err := os.MkdirAll(scenDir, 0755); err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}

	slug := toSlug(name)
	if slug == "" {
		slug = "scenario"
	}
	filePath := filepath.Join(scenDir, slug+".yaml")

	// Avoid overwriting an existing file by appending a suffix.
	if _, err := os.Stat(filePath); err == nil {
		for i := 2; ; i++ {
			candidate := filepath.Join(scenDir, fmt.Sprintf("%s-%d.yaml", slug, i))
			if _, err := os.Stat(candidate); err != nil {
				filePath = candidate
				break
			}
		}
	}

	starter := fmt.Sprintf(`schemaVersion: "1"
title: %s
description: ""
steps:
  - title: Step 1
    request:
      method: GET
      url: https://api.example.com
    assertions:
      - type: status
        operator: eq
        expected: 200
`, name)

	if err := atomicWrite(filePath, []byte(starter)); err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}
	return CreateScenarioResultIO{Path: filePath}
}

func toSlug(s string) string {
	var out strings.Builder
	prev := '-'
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			prev = r
		} else if prev != '-' {
			out.WriteRune('-')
			prev = '-'
		}
	}
	return strings.Trim(out.String(), "-")
}

// DuplicateScenario copies a scenario file into the same directory with a "-copy" suffix.
// Returns the new file path on success, or an error message in the Error field.
func (s *ScenarioService) DuplicateScenario(path string) CreateScenarioResultIO {
	data, err := os.ReadFile(path)
	if err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}

	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	newBase := base + "-copy"
	newPath := filepath.Join(dir, newBase+".yaml")
	for i := 2; ; i++ {
		if _, err := os.Stat(newPath); err != nil {
			break
		}
		newPath = filepath.Join(dir, fmt.Sprintf("%s-%d.yaml", newBase, i))
	}

	if err := atomicWrite(newPath, data); err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}
	return CreateScenarioResultIO{Path: newPath}
}

// DeleteScenario removes a scenario file. Returns "" on success or an error message.
func (s *ScenarioService) DeleteScenario(path string) string {
	if err := os.Remove(path); err != nil {
		return err.Error()
	}
	return ""
}

// RenameScenario renames a scenario file to {dir}/{slug(newName)}.yaml.
// Returns the new path on success, or an error message prefixed with "error:".
func (s *ScenarioService) RenameScenario(path, newName string) string {
	if newName == "" {
		return "error: name is required"
	}
	slug := toSlug(newName)
	if slug == "" {
		slug = "scenario"
	}
	dir := filepath.Dir(path)
	newPath := filepath.Join(dir, slug+".yaml")
	if newPath != path {
		if _, err := os.Stat(newPath); err == nil {
			return "error: a scenario with that name already exists"
		}
		if err := os.Rename(path, newPath); err != nil {
			return "error: " + err.Error()
		}
	}
	return newPath
}

// FolderRunItemIO is a single request result from a folder run.
type FolderRunItemIO struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Method     string `json:"method"`
	StatusCode int    `json:"statusCode"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
	Passed     bool   `json:"passed"` // true when status is 2xx and no network error
}

// RunFolder executes every request file directly under folderPath (non-recursive)
// using the active workspace environment. Results are returned in directory order.
func (s *ScenarioService) RunFolder(folderPath string) []FolderRunItemIO {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return []FolderRunItemIO{{Error: err.Error()}}
	}

	env := s.workspace.ActiveEnvironment()
	exec := buildExecutor(s.settings)

	var results []FolderRunItemIO
	for _, e := range entries {
		if e.IsDir() || !isDataFile(e.Name()) {
			continue
		}
		path := filepath.Join(folderPath, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			results = append(results, FolderRunItemIO{Name: e.Name(), Path: path, Error: err.Error()})
			continue
		}
		req, err := serialization.ParseRequestFile(data, e.Name())
		if err != nil {
			results = append(results, FolderRunItemIO{Name: e.Name(), Path: path, Error: "parse: " + err.Error()})
			continue
		}

		name := req.Title
		if name == "" {
			name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}

		// Apply variable interpolation with the active environment.
		resolver := variables.NewResolver(env)
		resolved := resolver.InterpolateRequest(req)

		resp, execErr := exec.Execute(s.ctx, resolved)
		if execErr != nil {
			results = append(results, FolderRunItemIO{
				Name:   name,
				Path:   path,
				Method: string(req.Method),
				Error:  execErr.Error(),
			})
			continue
		}
		results = append(results, FolderRunItemIO{
			Name:       name,
			Path:       path,
			Method:     string(req.Method),
			StatusCode: resp.StatusCode,
			DurationMs: resp.Duration.Milliseconds(),
			Passed:     resp.StatusCode >= 200 && resp.StatusCode < 300,
		})
	}
	return results
}

// RunScenario executes a scenario file and returns the full result.
// envPath overrides the workspace active environment; pass "" to use the active env.
// fromStep is the 0-based index of the first step to execute; pass 0 to run all steps.
func (s *ScenarioService) RunScenario(path, envPath string, fromStep int) ScenarioResultIO {
	data, err := os.ReadFile(path)
	if err != nil {
		return ScenarioResultIO{Error: fmt.Sprintf("read file: %v", err)}
	}
	sc, err := serialization.ParseScenarioFile(data, filepath.Base(path))
	if err != nil {
		return ScenarioResultIO{Error: fmt.Sprintf("parse: %v", err)}
	}

	// Resolve requestRef steps against the scenario's directory, then workspace requests/.
	scenDir := filepath.Dir(path)
	wsRequests := filepath.Join(s.workspace.workspacePath, "requests")
	for i := range sc.Steps {
		step := &sc.Steps[i]
		if step.Request != nil || step.RequestRef == "" {
			continue
		}
		candidates := []string{
			filepath.Join(scenDir, step.RequestRef),
			filepath.Join(wsRequests, step.RequestRef),
		}
		if filepath.IsAbs(step.RequestRef) {
			candidates = []string{step.RequestRef}
		}
		for _, refPath := range candidates {
			refData, err := os.ReadFile(refPath)
			if err != nil {
				continue
			}
			req, err := serialization.ParseRequestFile(refData, filepath.Base(refPath))
			if err == nil {
				step.Request = req
				break
			}
		}
	}

	exec := buildExecutorWithCookies(s.settings)
	runner := scenario.New(exec)

	env := s.workspace.ActiveEnvironment()
	if envPath != "" {
		envData, err := os.ReadFile(envPath)
		if err != nil {
			return ScenarioResultIO{Error: fmt.Sprintf("read env file: %v", err)}
		}
		if loaded, err := serialization.ParseEnvironmentFile(envData, filepath.Base(envPath)); err == nil {
			env = loaded
		}
	}

	result := runner.RunFrom(s.ctx, sc, env, fromStep)

	out := ScenarioResultIO{
		Title:      result.Scenario.Title,
		Passed:     result.Passed,
		DurationMs: result.Duration.Milliseconds(),
	}
	for _, sr := range result.Steps {
		step := StepResultIO{
			Title:      sr.Title,
			Passed:     sr.Passed,
			Skipped:    sr.Skipped,
			Error:      sr.Error,
			Extracted:  sr.Extracted,
			Assertions: []AssertionResultIO{},
		}
		if sr.Response != nil {
			step.StatusCode = sr.Response.StatusCode
			step.DurationMs = sr.Response.Duration.Milliseconds()
			step.ResponseBody = sr.Response.BodyString()
			for _, h := range sr.Response.Headers {
				step.ResponseHeaders = append(step.ResponseHeaders, HeaderIO{Name: h.Name, Value: h.Value})
			}
			if step.ResponseHeaders == nil {
				step.ResponseHeaders = []HeaderIO{}
			}
		}
		for _, ar := range sr.Assertions {
			step.Assertions = append(step.Assertions, AssertionResultIO{
				Type:    string(ar.Assertion.Type),
				Passed:  ar.Passed,
				Message: ar.Message,
			})
		}
		if step.Extracted == nil {
			step.Extracted = make(map[string]string)
		}
		out.Steps = append(out.Steps, step)
	}
	return out
}
