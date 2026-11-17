package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

func TestExportCoverageReport(t *testing.T) {
	ss, ws := newSS(t)
	specPath := writeFile(t, filepath.Join(ws.workspacePath, "api.yaml"), pingSpec)
	if err := coverage.Save(ws.workspacePath, coverage.StoredCoverage{
		SpecFile: specPath, TotalOps: 2, TotalRuns: 2, PassedRuns: 2,
		Exercised: []string{"GET /ping"},
		OperationResults: map[string]coverage.OperationResult{
			"GET /ping": {LastPassed: true, StatusCodes: []int{200}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	csv := ss.ExportCoverageReport(specPath, "csv")
	if csv.Error != "" || !strings.HasPrefix(csv.Content, "Method,Path,Coverage") ||
		!strings.Contains(csv.Content, "GET,/ping,Covered,Pass") || !strings.Contains(csv.Content, "GET,/other,Not Covered") {
		t.Errorf("csv: %+v", csv)
	}
	md := ss.ExportCoverageReport(specPath, "markdown")
	if md.Error != "" || !strings.Contains(md.Content, "# Contracts Report") || !strings.Contains(md.Content, "1 / 2 operations covered (50%)") {
		t.Errorf("markdown: %+v", md)
	}
	html := ss.ExportCoverageReport(specPath, "html")
	if html.Error != "" || !strings.HasPrefix(html.Content, "<!DOCTYPE html>") || !strings.Contains(html.Content, "Quality Gaps") {
		t.Errorf("html: %+v", html)
	}
}

func TestExportCoverageReportErrors(t *testing.T) {
	ss, ws := newSS(t)
	if got := ss.ExportCoverageReport("", "csv"); got.Error == "" {
		t.Error("no spec chosen")
	}
	specPath := writeFile(t, filepath.Join(ws.workspacePath, "api.yaml"), pingSpec)
	if got := ss.ExportCoverageReport(specPath, "csv"); !strings.Contains(got.Error, "no coverage has been recorded") {
		t.Errorf("nothing recorded yet: %+v", got)
	}
	if err := coverage.Save(ws.workspacePath, coverage.StoredCoverage{SpecFile: specPath, TotalRuns: 1, TotalOps: 2}); err != nil {
		t.Fatal(err)
	}
	if got := ss.ExportCoverageReport(specPath, "pdf"); !strings.Contains(got.Error, "unknown report format") {
		t.Errorf("unknown format: %+v", got)
	}

	closed := newWS(t, "")
	if got := NewScenarioService(closed, NewSettingsService()).ExportCoverageReport(specPath, "csv"); got.Error == "" {
		t.Error("no workspace open")
	}
}
