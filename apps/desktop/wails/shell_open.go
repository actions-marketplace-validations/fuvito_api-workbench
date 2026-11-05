package main

import (
	"fmt"
	"os"
	"os/exec"
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
