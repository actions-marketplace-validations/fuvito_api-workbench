package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fuvito/api-workbench/packages/core/importer"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ImportService handles Postman and Insomnia collection import.
type ImportService struct {
	ctx       context.Context
	workspace *WorkspaceService
}

func NewImportService(ws *WorkspaceService) *ImportService {
	return &ImportService{workspace: ws}
}

func (s *ImportService) setContext(ctx context.Context) {
	s.ctx = ctx
}

// CollectionImportResult is returned to the frontend after a collection import.
type CollectionImportResult struct {
	CollectionName string              `json:"collectionName"`
	Imported       []ImportedRequestIO `json:"imported"`
	Skipped        int                 `json:"skipped"`
	Error          string              `json:"error,omitempty"`
}

// ImportPostmanCollection opens a file picker, parses a Postman Collection v2.1 file,
// and writes request YAML files into the current workspace.
func (s *ImportService) ImportPostmanCollection() CollectionImportResult {
	path, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select Postman Collection",
		Filters: []runtime.FileFilter{
			{DisplayName: "Postman Collection (*.json)", Pattern: "*.json"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return CollectionImportResult{Error: "no file selected"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return CollectionImportResult{Error: fmt.Sprintf("reading file: %v", err)}
	}

	name, requests, err := importer.ParsePostmanCollection(data)
	if err != nil {
		return CollectionImportResult{Error: err.Error()}
	}

	return s.saveImportedRequests(name, requests)
}

// ImportInsomniaExport opens a file picker, parses an Insomnia v4 export file,
// and writes request YAML files into the current workspace.
func (s *ImportService) ImportInsomniaExport() CollectionImportResult {
	path, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select Insomnia Export",
		Filters: []runtime.FileFilter{
			{DisplayName: "Insomnia Export (*.json)", Pattern: "*.json"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return CollectionImportResult{Error: "no file selected"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return CollectionImportResult{Error: fmt.Sprintf("reading file: %v", err)}
	}

	name, requests, err := importer.ParseInsomniaExport(data)
	if err != nil {
		return CollectionImportResult{Error: err.Error()}
	}

	return s.saveImportedRequests(name, requests)
}

func (s *ImportService) saveImportedRequests(collectionName string, requests []importer.ImportedRequest) CollectionImportResult {
	wsPath := s.workspace.workspacePath
	if wsPath == "" {
		return CollectionImportResult{Error: "no workspace open — open or create a workspace first"}
	}

	saved := importer.Save(filepath.Join(wsPath, "requests"), collectionName, requests, false)
	result := CollectionImportResult{CollectionName: collectionName, Skipped: saved.Skipped}
	for _, sv := range saved.Saved {
		result.Imported = append(result.Imported, ImportedRequestIO{
			OperationID: "",
			Summary:     sv.Title,
			Method:      sv.Method,
			URL:         sv.URL,
			SavedPath:   sv.Path,
		})
	}
	return result
}
