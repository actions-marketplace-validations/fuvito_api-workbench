package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/bundle"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type BundleExportResultIO struct {
	Path            string   `json:"path"` // empty with no Error: the user cancelled
	Environments    int      `json:"environments"`
	Requests        int      `json:"requests"`
	Scenarios       int      `json:"scenarios"`
	RedactedSecrets int      `json:"redactedSecrets"`
	Skipped         []string `json:"skipped"`
	Warnings        []string `json:"warnings"`
	Error           string   `json:"error,omitempty"`
}

// ExportBundle writes the open workspace's environments, requests and scenarios to one
// bundle file chosen in a save dialog. Secret values are replaced with a placeholder so
// the file is safe to share or paste into a chat.
func (s *BundleService) ExportBundle() BundleExportResultIO {
	ws := s.workspace.workspacePath
	if ws == "" {
		return BundleExportResultIO{Error: "no workspace open — open or create a workspace first"}
	}
	path, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Export Bundle",
		DefaultFilename: filepath.Base(ws) + "-bundle.yaml",
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil {
		return BundleExportResultIO{Error: err.Error()}
	}
	if path == "" {
		return BundleExportResultIO{}
	}
	if ext := strings.ToLower(filepath.Ext(path)); ext != ".yaml" && ext != ".yml" {
		path += ".yaml"
	}
	return s.exportBundleTo(path)
}

func (s *BundleService) exportBundleTo(path string) BundleExportResultIO {
	ws := s.workspace.workspacePath
	if ws == "" {
		return BundleExportResultIO{Error: "no workspace open — open or create a workspace first"}
	}
	res, err := bundle.Export(ws, bundle.ExportOptions{})
	out := BundleExportResultIO{Skipped: []string{}, Warnings: []string{}}
	if res != nil {
		for _, sk := range res.Skipped {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s — %s", sk.Path, sk.Reason))
		}
		out.Warnings = append(out.Warnings, res.Warnings...)
	}
	if err != nil {
		out.Error = err.Error()
		return out
	}
	data, err := res.Marshal()
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if err := atomicWrite(path, data); err != nil {
		out.Error = fmt.Sprintf("writing %s: %v", path, err)
		return out
	}
	out.Path = path
	out.Environments = len(res.Bundle.Environments)
	out.Requests = len(res.Bundle.Requests)
	out.Scenarios = len(res.Bundle.Scenarios)
	out.RedactedSecrets = res.RedactedSecrets
	return out
}
