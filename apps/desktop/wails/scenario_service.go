package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/history"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

//go:embed scenario_templates/*.yaml
var scenarioTemplateFS embed.FS

// ---- I/O types ----

type ScenarioInfoIO struct {
	Title     string `json:"title"`
	Path      string `json:"path"`
	Folder    string `json:"folder"` // relative path from {workspace}/scenarios/, "/" separated, "" for root
	StepCount int    `json:"stepCount"`
	IsFolder  bool   `json:"isFolder,omitempty"` // sentinel: represents an empty directory, not a file
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
	Label             string               `json:"label"`
	Notes             string               `json:"notes,omitempty"`
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
	SpecFile    string         `json:"specFile,omitempty"`
	Steps       []StepDetailIO `json:"steps"`
	Error       string         `json:"error,omitempty"`
}

type AssertionResultIO struct {
	Type    string `json:"type"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type StepResultIO struct {
	Label           string              `json:"label"`
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

type CoverageIO struct {
	Covered   int      `json:"covered"`
	Total     int      `json:"total"`
	Percent   float64  `json:"percent"`
	Exercised []string `json:"exercised"`
	Missed    []string `json:"missed"`
}

type ScenarioResultIO struct {
	Title               string                 `json:"title"`
	Passed              bool                   `json:"passed"`
	DurationMs          int64                  `json:"durationMs"`
	Steps               []StepResultIO         `json:"steps"`
	Error               string                 `json:"error,omitempty"`
	SpecError           string                 `json:"specError,omitempty"`
	Coverage            *CoverageIO            `json:"coverage,omitempty"`
	AccumulatedCoverage *AccumulatedCoverageIO `json:"accumulatedCoverage,omitempty"`
}

type StepRunEntryIO struct {
	Label      string `json:"label"`
	Passed     bool   `json:"passed"`
	Skipped    bool   `json:"skipped"`
	StatusCode int    `json:"statusCode,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
}

type ScenarioRunEntryIO struct {
	ID           int64            `json:"id"`
	ExecutedAt   string           `json:"executedAt"`
	ScenarioPath string           `json:"scenarioPath"`
	Title        string           `json:"title"`
	Passed       bool             `json:"passed"`
	DurationMs   int64            `json:"durationMs"`
	StepTotal    int              `json:"stepTotal"`
	StepPassed   int              `json:"stepPassed"`
	StepFailed   int              `json:"stepFailed"`
	StepSkipped  int              `json:"stepSkipped"`
	Steps        []StepRunEntryIO `json:"steps"`
}

// ---- Service ----

type ScenarioService struct {
	ctx       context.Context
	workspace *WorkspaceService
	settings  *SettingsService
	mu        sync.Mutex
	cancelRun context.CancelFunc
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
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p == scenDir {
				return nil // skip root itself
			}
			// Emit a folder sentinel so the UI can render empty directories.
			rel, _ := filepath.Rel(scenDir, p)
			out = append(out, ScenarioInfoIO{IsFolder: true, Folder: filepath.ToSlash(rel), Path: p})
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		rel, _ := filepath.Rel(scenDir, filepath.Dir(p))
		folder := ""
		if rel != "." {
			folder = filepath.ToSlash(rel)
		}
		info := ScenarioInfoIO{Title: name, Path: p, Folder: folder}
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
		SpecFile:    sc.SpecFile,
	}
	for _, step := range sc.Steps {
		sd := StepDetailIO{
			Label:       step.Label,
			Notes:       step.Notes,
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

// ValidateScenarioYAML parses and validates YAML content as a scenario file.
// Returns "" on success, or a human-readable error/issue list on failure.
func (s *ScenarioService) ValidateScenarioYAML(content string) string {
	sc, err := serialization.ParseScenario([]byte(content), "yaml")
	if err != nil {
		return "YAML parse error: " + err.Error()
	}
	issues := scenario.Validate(sc, "")
	if len(issues) == 0 {
		return ""
	}
	msgs := make([]string, len(issues))
	for i, iss := range issues {
		msgs[i] = iss.String()
	}
	return strings.Join(msgs, "\n")
}

// CreateScenarioResultIO is the return type for CreateScenario.
type CreateScenarioResultIO struct {
	Path  string `json:"path"`
	Error string `json:"error,omitempty"`
}

// CreateScenario creates a starter scenario YAML file. folderPath is relative to
// {workspace}/scenarios/ (e.g. "auth" or ""). Returns the new file path, or an error.
func (s *ScenarioService) CreateScenario(name, folderPath string) CreateScenarioResultIO {
	if name == "" {
		return CreateScenarioResultIO{Error: "name is required"}
	}
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return CreateScenarioResultIO{Error: "no workspace open"}
	}

	scenDir := filepath.Join(wsPath, "scenarios")
	targetDir := scenDir
	if folderPath != "" {
		clean := filepath.Clean(filepath.FromSlash(folderPath))
		if strings.HasPrefix(clean, "..") {
			return CreateScenarioResultIO{Error: "invalid folder path"}
		}
		targetDir = filepath.Join(scenDir, clean)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}

	slug := toSlug(name)
	if slug == "" {
		slug = "scenario"
	}
	filePath := filepath.Join(targetDir, slug+".yaml")

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
  - label: Step 1
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

// MoveScenario moves a scenario file into a different folder under {workspace}/scenarios/.
// destFolderRelPath is relative to scenarios/ (use "" for the root).
// Returns the new absolute path on success, or an error.
func (s *ScenarioService) MoveScenario(srcPath, destFolderRelPath string) (string, error) {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return "", fmt.Errorf("no workspace open")
	}
	scenDir := filepath.Join(wsPath, "scenarios")
	rel, err := filepath.Rel(scenDir, srcPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("source is outside the scenarios directory")
	}
	var destDir string
	if destFolderRelPath == "" {
		destDir = scenDir
	} else {
		clean := filepath.Clean(filepath.FromSlash(destFolderRelPath))
		if strings.HasPrefix(clean, "..") {
			return "", fmt.Errorf("invalid destination folder path")
		}
		destDir = filepath.Join(scenDir, clean)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating destination: %w", err)
	}
	newPath := filepath.Join(destDir, filepath.Base(srcPath))
	if newPath == srcPath {
		return srcPath, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("a file named %q already exists in the destination folder", filepath.Base(srcPath))
	}
	if err := os.Rename(srcPath, newPath); err != nil {
		return "", fmt.Errorf("moving file: %w", err)
	}
	return newPath, nil
}

// CopyScenario copies a scenario file into a different folder under {workspace}/scenarios/.
// destFolderRelPath is relative to scenarios/ (use "" for the root).
// Returns the new absolute path on success, or an error.
func (s *ScenarioService) CopyScenario(srcPath, destFolderRelPath string) (string, error) {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return "", fmt.Errorf("no workspace open")
	}
	scenDir := filepath.Join(wsPath, "scenarios")
	rel, err := filepath.Rel(scenDir, srcPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("source is outside the scenarios directory")
	}
	var destDir string
	if destFolderRelPath == "" {
		destDir = scenDir
	} else {
		clean := filepath.Clean(filepath.FromSlash(destFolderRelPath))
		if strings.HasPrefix(clean, "..") {
			return "", fmt.Errorf("invalid destination folder path")
		}
		destDir = filepath.Join(scenDir, clean)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating destination: %w", err)
	}
	base := filepath.Base(srcPath)
	newPath := filepath.Join(destDir, base)
	if _, err := os.Stat(newPath); err == nil {
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		for i := 2; ; i++ {
			if i == 2 {
				newPath = filepath.Join(destDir, stem+"-copy"+ext)
			} else {
				newPath = filepath.Join(destDir, fmt.Sprintf("%s-copy-%d%s", stem, i, ext))
			}
			if _, err := os.Stat(newPath); err != nil {
				break
			}
		}
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("reading source: %w", err)
	}
	if err := atomicWrite(newPath, data); err != nil {
		return "", fmt.Errorf("writing copy: %w", err)
	}
	return newPath, nil
}

// CreateScenarioFolder creates a subfolder under {workspace}/scenarios/.
// folderPath is relative to scenarios/ (e.g. "auth" or "auth/subsection").
// Returns "" on success or an error message.
func (s *ScenarioService) CreateScenarioFolder(folderPath string) string {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return "no workspace open"
	}
	if folderPath == "" {
		return "folder name is required"
	}
	clean := filepath.Clean(filepath.FromSlash(folderPath))
	if strings.HasPrefix(clean, "..") {
		return "invalid folder path"
	}
	dir := filepath.Join(wsPath, "scenarios", clean)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err.Error()
	}
	return ""
}

// RenameScenarioFolder renames the last segment of a folder.
// folderPath is the current relative path from scenarios/; newName is just the new last segment.
// Returns the new relative path on success, or "error: ..." on failure.
func (s *ScenarioService) RenameScenarioFolder(folderPath, newName string) string {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return "error: no workspace open"
	}
	if newName == "" {
		return "error: name is required"
	}
	clean := filepath.Clean(filepath.FromSlash(folderPath))
	if strings.HasPrefix(clean, "..") || clean == "." {
		return "error: invalid folder path"
	}
	slug := toSlug(newName)
	if slug == "" {
		return "error: invalid folder name"
	}
	scenDir := filepath.Join(wsPath, "scenarios")
	oldAbs := filepath.Join(scenDir, clean)
	newAbs := filepath.Join(filepath.Dir(oldAbs), slug)
	if oldAbs == newAbs {
		return folderPath
	}
	if _, err := os.Stat(newAbs); err == nil {
		return "error: a folder with that name already exists"
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return "error: " + err.Error()
	}
	rel, _ := filepath.Rel(scenDir, newAbs)
	return filepath.ToSlash(rel)
}

// DeleteScenarioFolder removes a folder and all its contents.
// folderPath is relative to {workspace}/scenarios/.
// Returns "" on success or an error message.
func (s *ScenarioService) DeleteScenarioFolder(folderPath string) string {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return "no workspace open"
	}
	clean := filepath.Clean(filepath.FromSlash(folderPath))
	if strings.HasPrefix(clean, "..") || clean == "." {
		return "invalid folder path"
	}
	dir := filepath.Join(wsPath, "scenarios", clean)
	if err := os.RemoveAll(dir); err != nil {
		return err.Error()
	}
	return ""
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

	// Load the OpenAPI spec now (before creating the runner) so that
	// type:schema assertions with an operationId value can resolve against it.
	var loadedSpec *openapi.Spec
	var specLoadErr string
	if sc.SpecFile != "" {
		specData, specExt, err := loadSpecBytes(sc.SpecFile, filepath.Dir(path))
		if err != nil {
			specLoadErr = fmt.Sprintf("specFile %q: %v", sc.SpecFile, err)
		} else {
			spec, err := openapi.ParseBytes(specData, specExt)
			if err != nil {
				specLoadErr = fmt.Sprintf("specFile %q: %v", sc.SpecFile, err)
			} else {
				loadedSpec = spec
			}
		}
	}

	var runner *scenario.Runner
	if loadedSpec != nil {
		runner = scenario.NewWithSpec(exec, loadedSpec)
	} else {
		runner = scenario.New(exec)
	}

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

	result := runner.RunFrom(ctx, sc, env, fromStep)

	out := ScenarioResultIO{
		Title:      result.Scenario.Title,
		Passed:     result.Passed,
		DurationMs: result.Duration.Milliseconds(),
		SpecError:  specLoadErr,
	}
	for _, sr := range result.Steps {
		step := StepResultIO{
			Label:      sr.Label,
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

	// Compute spec coverage using the already-loaded spec (if any).
	if loadedSpec != nil {
		var executed []openapi.ExecutedRequest
		for _, sr := range result.Steps {
			if sr.Skipped || sr.Request == nil {
				continue
			}
			executed = append(executed, openapi.ExecutedRequest{
				Method: string(sr.Request.Method),
				URL:    sr.Request.URL,
				Passed: sr.Passed,
			})
		}
		if cr := loadedSpec.ScenarioCoverage(executed); cr != nil {
			cov := &CoverageIO{
				Covered: cr.Covered,
				Total:   cr.Total,
				Percent: cr.Percent,
			}
			for _, op := range cr.Exercised {
				cov.Exercised = append(cov.Exercised, op.String())
			}
			for _, op := range cr.Missed {
				cov.Missed = append(cov.Missed, op.String())
			}
			out.Coverage = cov

			// Accumulate into the workspace coverage store.
			if s.workspace.workspacePath != "" {
				stored := loadStoredCoverage(s.workspace.workspacePath, sc.SpecFile)
				stored.TotalRuns++
				stored.LastRunAt = time.Now()
				if result.Passed {
					stored.PassedRuns++
				} else {
					stored.FailedRuns++
				}
				stored.Exercised = mergeExercised(stored.Exercised, cov.Exercised)
				stored.TotalOps = cr.Total
				now := time.Now()
				if stored.OperationResults == nil {
					stored.OperationResults = make(map[string]coverage.OperationResult)
				}
				for _, er := range cr.ExercisedResults {
					stored.OperationResults[er.String()] = coverage.OperationResult{
						LastPassed: er.Passed,
						LastRunAt:  now,
					}
				}
				if err := saveStoredCoverage(s.workspace.workspacePath, stored); err == nil {
					acc := buildAccumulatedCoverageIO(stored, loadedSpec)
					out.AccumulatedCoverage = &acc
				}
			}
		}
	}

	// Record this run in the scenario run log (silently — a broken db never blocks results).
	s.recordScenarioRun(path, out)

	return out
}

// CancelScenario aborts the currently in-flight RunScenario call, if any.
func (s *ScenarioService) CancelScenario() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelRun != nil {
		s.cancelRun()
		s.cancelRun = nil
	}
}

// recordScenarioRun opens the history database and appends a run record.
// Errors are silently discarded so a broken database never blocks scenario output.
func (s *ScenarioService) recordScenarioRun(scenarioPath string, out ScenarioResultIO) {
	dbPath, err := history.DefaultPath()
	if err != nil {
		return
	}
	store, err := history.Open(dbPath)
	if err != nil {
		return
	}
	defer store.Close()

	counts := result{passed: 0, failed: 0, skipped: 0}
	steps := make([]history.ScenarioStepSummary, len(out.Steps))
	for i, sr := range out.Steps {
		steps[i] = history.ScenarioStepSummary{
			Label:      sr.Label,
			Passed:     sr.Passed,
			Skipped:    sr.Skipped,
			StatusCode: sr.StatusCode,
			DurationMs: sr.DurationMs,
			Error:      sr.Error,
		}
		if sr.Skipped {
			counts.skipped++
		} else if sr.Passed {
			counts.passed++
		} else {
			counts.failed++
		}
	}

	_ = store.RecordScenarioRun(history.ScenarioRunRecord{
		WorkspacePath: s.workspace.workspacePath,
		ScenarioPath:  scenarioPath,
		Title:         out.Title,
		Passed:        out.Passed,
		DurationMs:    out.DurationMs,
		StepTotal:     len(out.Steps),
		StepPassed:    counts.passed,
		StepFailed:    counts.failed,
		StepSkipped:   counts.skipped,
		Steps:         steps,
	})
}

type result struct{ passed, failed, skipped int }

// GetScenarioRunHistory returns recent scenario runs for the current workspace.
func (s *ScenarioService) GetScenarioRunHistory(limit int) []ScenarioRunEntryIO {
	if limit <= 0 {
		limit = 100
	}
	dbPath, err := history.DefaultPath()
	if err != nil {
		return nil
	}
	store, err := history.Open(dbPath)
	if err != nil {
		return nil
	}
	defer store.Close()

	records, err := store.RecentScenarioRuns(s.workspace.workspacePath, limit)
	if err != nil {
		return nil
	}
	out := make([]ScenarioRunEntryIO, len(records))
	for i, r := range records {
		steps := make([]StepRunEntryIO, len(r.Steps))
		for j, sr := range r.Steps {
			steps[j] = StepRunEntryIO{
				Label:      sr.Label,
				Passed:     sr.Passed,
				Skipped:    sr.Skipped,
				StatusCode: sr.StatusCode,
				DurationMs: sr.DurationMs,
				Error:      sr.Error,
			}
		}
		out[i] = ScenarioRunEntryIO{
			ID:           r.ID,
			ExecutedAt:   r.ExecutedAt.Format(time.RFC3339),
			ScenarioPath: r.ScenarioPath,
			Title:        r.Title,
			Passed:       r.Passed,
			DurationMs:   r.DurationMs,
			StepTotal:    r.StepTotal,
			StepPassed:   r.StepPassed,
			StepFailed:   r.StepFailed,
			StepSkipped:  r.StepSkipped,
			Steps:        steps,
		}
	}
	return out
}

// GetAccumulatedCoverage returns the accumulated spec coverage for the given specFile.
// Returns nil (no error) when no data has been recorded yet.
func (s *ScenarioService) GetAccumulatedCoverage(specFile string) (*AccumulatedCoverageIO, error) {
	if s.workspace.workspacePath == "" || specFile == "" {
		return nil, nil
	}
	stored := loadStoredCoverage(s.workspace.workspacePath, specFile)
	if stored.TotalRuns == 0 {
		return nil, nil
	}
	specData, specExt, err := loadSpecBytes(specFile, "")
	if err != nil {
		return nil, fmt.Errorf("loading spec: %w", err)
	}
	spec, err := openapi.ParseBytes(specData, specExt)
	if err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	acc := buildAccumulatedCoverageIO(stored, spec)
	return &acc, nil
}

// ResetAccumulatedCoverage deletes the stored coverage data for the given specFile.
func (s *ScenarioService) ResetAccumulatedCoverage(specFile string) error {
	if s.workspace.workspacePath == "" || specFile == "" {
		return nil
	}
	return coverage.Reset(s.workspace.workspacePath, specFile)
}

// ListAccumulatedCoverage returns all stored accumulated coverage entries for the workspace,
// sorted by most recently run first. Uses cached TotalOps so no spec files are reloaded.
func (s *ScenarioService) ListAccumulatedCoverage() []AccumulatedCoverageIO {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return nil
	}
	reports := coverage.ListAll(wsPath)
	result := make([]AccumulatedCoverageIO, len(reports))
	for i, r := range reports {
		result[i] = reportToIO(r)
	}
	return result
}

// loadSpecBytes fetches the OpenAPI spec at specRef (HTTP/HTTPS URL or file path).
// scenarioDir is used to resolve relative paths; it is ignored for URLs.
// Returns the raw bytes and a format hint ("yaml" or "json").
func loadSpecBytes(specRef, scenarioDir string) (data []byte, format string, err error) {
	if strings.HasPrefix(specRef, "http://") || strings.HasPrefix(specRef, "https://") {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, specRef, nil)
		if reqErr != nil {
			return nil, "", reqErr
		}
		resp, httpErr := http.DefaultClient.Do(req)
		if httpErr != nil {
			return nil, "", httpErr
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("HTTP %d fetching spec", resp.StatusCode)
		}
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, "", readErr
		}
		// Detect format from URL path; default to yaml.
		format = "yaml"
		if strings.HasSuffix(strings.ToLower(specRef), ".json") {
			format = "json"
		}
		return body, format, nil
	}

	// File path — resolve relative paths against the scenario's directory.
	if !filepath.IsAbs(specRef) {
		specRef = filepath.Join(scenarioDir, specRef)
	}
	body, readErr := os.ReadFile(specRef)
	if readErr != nil {
		return nil, "", readErr
	}
	format = strings.TrimPrefix(strings.ToLower(filepath.Ext(specRef)), ".")
	if format == "" {
		format = "yaml"
	}
	return body, format, nil
}

// ---- Scenario template gallery ----

// TemplateInfoIO describes a bundled scenario template available for the user to copy.
type TemplateInfoIO struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	StepCount   int      `json:"stepCount"`
	// Note is shown as a small callout on the card (e.g. "Requires {{baseUrl}}").
	Note string `json:"note,omitempty"`
}

// categoryAndTags derives category and tag metadata from the template filename.
func templateMeta(id string) (category string, tags []string, note string) {
	conformanceIDs := map[string]bool{
		"user-lifecycle":        true,
		"pagination":            true,
		"401-www-authenticate":  true,
		"405-method-not-allowed": true,
		"503-service-unavailable": true,
		"content-negotiation":   true,
		"etag-conditional":      true,
		"problem-details":       true,
		"rate-limit":            true,
	}
	if conformanceIDs[id] {
		return "Conformance", []string{"RFC", "headers"}, "Requires {{baseUrl}} (use with demo server)"
	}
	switch id {
	case "jsonplaceholder-crud":
		return "Public APIs", []string{"CRUD", "REST", "no auth"}, ""
	case "github-public-api":
		return "Public APIs", []string{"GitHub", "no auth"}, ""
	}
	return "Other", nil, ""
}

// ListTemplates returns all bundled scenario templates.
func (s *ScenarioService) ListTemplates() []TemplateInfoIO {
	entries, err := scenarioTemplateFS.ReadDir("scenario_templates")
	if err != nil {
		return nil
	}
	var out []TemplateInfoIO
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		data, err := scenarioTemplateFS.ReadFile("scenario_templates/" + e.Name())
		if err != nil {
			continue
		}
		sc, err := serialization.ParseScenarioFile(data, e.Name())
		if err != nil {
			continue
		}
		desc := strings.TrimSpace(sc.Description)
		// Truncate long descriptions for the card preview.
		if len(desc) > 160 {
			desc = desc[:157] + "…"
		}
		category, tags, note := templateMeta(id)
		out = append(out, TemplateInfoIO{
			ID:          id,
			Title:       sc.Title,
			Description: desc,
			Category:    category,
			Tags:        tags,
			StepCount:   len(sc.Steps),
			Note:        note,
		})
	}
	return out
}

// CreateFromTemplate copies a bundled template into the workspace scenarios directory
// under the given name and returns the new file path.
func (s *ScenarioService) CreateFromTemplate(templateID, name string) CreateScenarioResultIO {
	if templateID == "" {
		return CreateScenarioResultIO{Error: "templateID is required"}
	}
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return CreateScenarioResultIO{Error: "no workspace open"}
	}

	srcPath := "scenario_templates/" + templateID + ".yaml"
	data, err := scenarioTemplateFS.ReadFile(srcPath)
	if err != nil {
		return CreateScenarioResultIO{Error: "template not found: " + templateID}
	}

	// Parse and re-serialise with the new title so the YAML reflects the user's chosen name.
	sc, err := serialization.ParseScenarioFile(data, templateID+".yaml")
	if err != nil {
		return CreateScenarioResultIO{Error: "parsing template: " + err.Error()}
	}
	if strings.TrimSpace(name) != "" {
		sc.Title = strings.TrimSpace(name)
	}

	scenDir := filepath.Join(wsPath, "scenarios")
	if err := os.MkdirAll(scenDir, 0755); err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}

	slug := toSlug(sc.Title)
	if slug == "" {
		slug = templateID
	}
	filePath := filepath.Join(scenDir, slug+".yaml")
	for i := 2; ; i++ {
		if _, err := os.Stat(filePath); err != nil {
			break
		}
		filePath = filepath.Join(scenDir, fmt.Sprintf("%s-%d.yaml", slug, i))
	}

	out, err := serialization.WriteScenario(sc, "yaml")
	if err != nil {
		return CreateScenarioResultIO{Error: "serialising template: " + err.Error()}
	}
	if err := atomicWrite(filePath, out); err != nil {
		return CreateScenarioResultIO{Error: err.Error()}
	}
	return CreateScenarioResultIO{Path: filePath}
}
