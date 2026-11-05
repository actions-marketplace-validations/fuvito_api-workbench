package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
)

// OpenInSystemViewer writes content to a temp file with the appropriate extension
// and opens it with the OS default application for that file type.
// Returns a non-empty error string on failure.
func (s *WorkspaceService) OpenInSystemViewer(content, format string) string {
	ext := formatToExtension(format)
	f, err := os.CreateTemp("", "apidesk-preview-*"+ext)
	if err != nil {
		return "could not create temp file: " + err.Error()
	}
	path := f.Name()
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "could not write temp file: " + err.Error()
	}
	f.Close()

	if err := openWithSystemApp(path); err != nil {
		return "could not open viewer: " + err.Error()
	}
	return ""
}

// OpenInExplorer opens the given path in the OS file manager.
// For files it selects the item in its parent folder; for directories it opens the folder.
func (s *WorkspaceService) OpenInExplorer(path string) string {
	native := filepath.FromSlash(path)
	fi, statErr := os.Stat(native)
	isDir := statErr == nil && fi.IsDir()

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		if isDir {
			cmd = exec.Command("explorer", native)
		} else {
			cmd = exec.Command("explorer", "/select,"+native)
		}
	case "darwin":
		if isDir {
			cmd = exec.Command("open", native)
		} else {
			cmd = exec.Command("open", "-R", native)
		}
	default:
		if isDir {
			cmd = exec.Command("xdg-open", native)
		} else {
			cmd = exec.Command("xdg-open", filepath.Dir(native))
		}
	}
	if err := cmd.Start(); err != nil {
		return "could not open explorer: " + err.Error()
	}
	go cmd.Wait() //nolint:errcheck
	return ""
}

func formatToExtension(format string) string {
	switch format {
	case "json":
		return ".json"
	case "xml":
		return ".xml"
	case "html":
		return ".html"
	default:
		return ".txt"
	}
}

func openWithSystemApp(path string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launching viewer: %w", err)
	}
	// Don't wait — the app owns the process from here.
	go cmd.Wait() //nolint:errcheck
	return nil
}
