package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/importer"
	"github.com/fuvito/api-workbench/packages/core/serialization"
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

	dirName := sanitizeDirName(collectionName)
	if dirName == "" {
		dirName = "imported"
	}

	result := CollectionImportResult{CollectionName: collectionName}

	for _, ir := range requests {
		destDir := filepath.Join(wsPath, "requests", dirName)
		if ir.FolderPath != "" {
			for _, seg := range strings.Split(ir.FolderPath, "/") {
				destDir = filepath.Join(destDir, sanitizeDirName(seg))
			}
		}
		if err := os.MkdirAll(destDir, 0755); err != nil {
			result.Skipped++
			continue
		}

		yamlData, err := serialization.WriteRequest(ir.Request, "yaml")
		if err != nil {
			result.Skipped++
			continue
		}

		filename := sanitizeFilename(ir.Request.Title)
		if filename == "" {
			filename = "request"
		}
		savePath := filepath.Join(destDir, filename+".yaml")

		if err := atomicWrite(savePath, yamlData); err != nil {
			result.Skipped++
			continue
		}

		result.Imported = append(result.Imported, ImportedRequestIO{
			OperationID: "",
			Summary:     ir.Request.Title,
			Method:      string(ir.Request.Method),
			URL:         ir.Request.URL,
			SavedPath:   savePath,
		})
	}

	return result
}
