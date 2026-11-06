package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ImportedRequestIO is the Wails-serializable summary of one imported request.
type ImportedRequestIO struct {
	OperationID string `json:"operationId"`
	Summary     string `json:"summary"`
	Method      string `json:"method"`
	URL         string `json:"url"`
	SavedPath   string `json:"savedPath"`
	Tags        []string `json:"tags,omitempty"`
}

// ImportResult is returned from ImportOpenAPISpec.
type ImportResult struct {
	SpecTitle  string              `json:"specTitle"`
	SpecVersion string             `json:"specVersion"`
	Imported   []ImportedRequestIO `json:"imported"`
	Error      string              `json:"error,omitempty"`
}

// OpenAPIService handles OpenAPI spec import from the desktop application.
type OpenAPIService struct {
	ctx       context.Context
	workspace *WorkspaceService
}

func NewOpenAPIService(ws *WorkspaceService) *OpenAPIService {
	return &OpenAPIService{workspace: ws}
}

func (s *OpenAPIService) setContext(ctx context.Context) {
	s.ctx = ctx
}

// ImportOpenAPISpec opens a file dialog, parses the selected spec, generates
// request files in the workspace, and returns a summary.
func (s *OpenAPIService) ImportOpenAPISpec(baseURL string) ImportResult {
	path, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select OpenAPI Specification",
		Filters: []runtime.FileFilter{
			{DisplayName: "OpenAPI Spec (*.yaml, *.yml, *.json)", Pattern: "*.yaml;*.yml;*.json"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return ImportResult{Error: "no file selected"}
	}

	return s.importFromPath(path, baseURL)
}

// ImportOpenAPISpecFromPath imports a spec from an absolute file path.
// Used for testing and CLI-adjacent workflows.
func (s *OpenAPIService) ImportOpenAPISpecFromPath(path, baseURL string) ImportResult {
	return s.importFromPath(path, baseURL)
}

// ImportOpenAPISpecFromURL fetches an OpenAPI spec from a URL and imports it.
func (s *OpenAPIService) ImportOpenAPISpecFromURL(url, baseURL string) ImportResult {
	if url == "" {
		return ImportResult{Error: "URL is required"}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("fetching URL: %v", err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ImportResult{Error: fmt.Sprintf("server returned %d", resp.StatusCode)}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("reading response: %v", err)}
	}
	format := "yaml"
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") || strings.HasSuffix(strings.ToLower(url), ".json") {
		format = "json"
	}
	return s.importFromBytes(data, format, baseURL)
}

func (s *OpenAPIService) importFromPath(path, baseURL string) ImportResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("reading file: %v", err)}
	}

	format := strings.TrimPrefix(filepath.Ext(path), ".")
	return s.importFromBytes(data, format, baseURL)
}

func (s *OpenAPIService) importFromBytes(data []byte, format, baseURL string) ImportResult {
	spec, err := openapi.ParseBytes(data, format)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("parsing spec: %v", err)}
	}

	title, version := spec.Info()

	reqs, err := openapi.GenerateRequests(spec, baseURL)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("generating requests: %v", err)}
	}

	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return ImportResult{Error: "no workspace open — open or create a workspace first"}
	}

	// Determine the directory to save requests into; use a subdir per spec title.
	dirName := sanitizeDirName(title)
	if dirName == "" {
		dirName = "imported"
	}
	destDir := filepath.Join(wsPath, "requests", dirName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return ImportResult{Error: fmt.Sprintf("creating import directory: %v", err)}
	}

	result := ImportResult{SpecTitle: title, SpecVersion: version}

	for _, gr := range reqs {
		yamlData, err := serialization.WriteRequest(gr.Request, "yaml")
		if err != nil {
			continue
		}

		filename := sanitizeFilename(gr.OperationID)
		if filename == "" {
			filename = sanitizeFilename(gr.Request.Title)
		}
		if filename == "" {
			filename = "request"
		}
		savePath := filepath.Join(destDir, filename+".yaml")

		if err := atomicWrite(savePath, yamlData); err != nil {
			continue
		}

		result.Imported = append(result.Imported, ImportedRequestIO{
			OperationID: gr.OperationID,
			Summary:     gr.Summary,
			Method:      string(gr.Request.Method),
			URL:         gr.Request.URL,
			SavedPath:   savePath,
			Tags:        gr.Tags,
		})
	}

	return result
}

func sanitizeDirName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	return b.String()
}
