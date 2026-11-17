package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

func coverageWorkspace(t *testing.T) (ws, spec string) {
	t.Helper()
	ws = t.TempDir()
	spec = writeTempFile(t, "api.yaml", qualitySpec)
	err := coverage.Save(ws, coverage.StoredCoverage{
		SpecFile: spec, TotalOps: 3, TotalRuns: 2, PassedRuns: 1, FailedRuns: 1,
		Exercised: []string{"GET /a", "GET /b"},
		OperationResults: map[string]coverage.OperationResult{
			"GET /a": {LastPassed: true, StatusCodes: []int{200}},
			"GET /b": {LastPassed: false, StatusCodes: []int{200}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return ws, spec
}

func TestCoverageQuality_FileFormats(t *testing.T) {
	ws, spec := coverageWorkspace(t)

	csv, err := execFull(t, "coverage", "quality", "-w", ws, "-f", "csv", spec)
	if exitCode(err) != 1 {
		t.Fatalf("gaps exit 1 in every format, got %v", err)
	}
	for _, want := range []string{"Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes", "GET,/a,Covered,Pass", "GET,/b,Covered,Fail", "GET,/c,Not Covered", "404"} {
		if !strings.Contains(csv, want) {
			t.Errorf("csv missing %q:\n%s", want, csv)
		}
	}

	md, _ := execFull(t, "coverage", "quality", "-w", ws, "-f", "markdown", spec)
	for _, want := range []string{"# Contracts Report — ", "2 / 3 operations covered", "### Happy-path only", "### Error responses never triggered", "| `GET /a` | 404 |"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}

	html, _ := execFull(t, "coverage", "quality", "-w", ws, "-f", "html", spec)
	if !strings.HasPrefix(html, "<!DOCTYPE html>") || !strings.Contains(html, "Quality Gaps") || !strings.Contains(html, "✗ Fail") {
		t.Errorf("html:\n%s", html)
	}
}

func TestCoverageQuality_OutputFileIsWrittenWithFolders(t *testing.T) {
	ws, spec := coverageWorkspace(t)
	file := filepath.Join(t.TempDir(), "reports", "contracts.html")
	out, err := execFull(t, "coverage", "quality", "-w", ws, "-f", "html", "-o", file, spec)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %v", err)
	}
	if strings.Contains(out, "<!DOCTYPE") {
		t.Error("with --output nothing is printed to the screen")
	}
	if !strings.Contains(readFile(t, file), "Contracts Report") {
		t.Error("report file not written")
	}
}

func TestCoverageList_FileFormats(t *testing.T) {
	ws, spec := coverageWorkspace(t)

	csv, err := execFull(t, "coverage", "list", "-w", ws, "-f", "csv")
	if err != nil || !strings.HasPrefix(csv, "Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run") || !strings.Contains(csv, filepath.Base(spec)) {
		t.Errorf("csv: %v\n%s", err, csv)
	}
	md, _ := execFull(t, "coverage", "list", "-w", ws, "-f", "markdown")
	if !strings.Contains(md, "# Contracts coverage") || !strings.Contains(md, "2 runs") && !strings.Contains(md, "| 2 | 1 | 1 |") {
		t.Errorf("markdown:\n%s", md)
	}
	html, _ := execFull(t, "coverage", "list", "-w", ws, "-f", "html")
	if !strings.Contains(html, "<table>") {
		t.Errorf("html:\n%s", html)
	}

	empty, err := execFull(t, "coverage", "list", "-w", t.TempDir(), "-f", "csv")
	if err != nil || strings.TrimSpace(empty) != "Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run" {
		t.Errorf("an empty workspace still gives a valid, empty report: %v\n%q", err, empty)
	}
}
