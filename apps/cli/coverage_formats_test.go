package main

import (
	"encoding/json"
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

	csv, err := execFull(t, "coverage", "quality", "--fail-on-gaps", "-w", ws, "-f", "csv", spec)
	if exitCode(err) != 1 {
		t.Fatalf("gaps exit 1 in every format, got %v", err)
	}
	for _, want := range []string{"Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes,Status", "GET,/a,Covered,Pass", "Passing", "GET,/b,Covered,Fail", "Failing", "GET,/c,Not Covered", "Not covered", "404"} {
		if !strings.Contains(csv, want) {
			t.Errorf("csv missing %q:\n%s", want, csv)
		}
	}

	md, _ := execFull(t, "coverage", "quality", "-w", ws, "-f", "markdown", spec)
	for _, want := range []string{"# Contracts Report — ", "| Covered | Not covered | Passing | Failing |", "| 2 (67%) | 1 (33%) | 1 (33%) | 1 (33%) |", "### Happy-path only", "### Error responses never triggered", "| `GET /a` | 404 |"} {
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
	if err != nil { // gaps fail the command only with --fail-on-gaps
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
	if err != nil || !strings.HasPrefix(csv, "Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run,Not Covered,Passing,Failing") || !strings.Contains(csv, filepath.Base(spec)) {
		t.Errorf("csv: %v\n%s", err, csv)
	}
	md, _ := execFull(t, "coverage", "list", "-w", ws, "-f", "markdown")
	if !strings.Contains(md, "# Contracts coverage") || !strings.Contains(md, "| Spec | Covered | Not covered | Passing | Failing | Scenario runs | Last run |") {
		t.Errorf("markdown:\n%s", md)
	}
	html, _ := execFull(t, "coverage", "list", "-w", ws, "-f", "html")
	if !strings.Contains(html, "<table>") {
		t.Errorf("html:\n%s", html)
	}

	empty, err := execFull(t, "coverage", "list", "-w", t.TempDir(), "-f", "csv")
	if err != nil || strings.TrimSpace(empty) != "Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run,Not Covered,Passing,Failing" {
		t.Errorf("an empty workspace still gives a valid, empty report: %v\n%q", err, empty)
	}
}

func TestCoverageList_TextAndJSONCarryTheFourTotals(t *testing.T) {
	ws, _ := coverageWorkspace(t)

	text, err := execFull(t, "coverage", "list", "-w", ws)
	if err != nil || !strings.Contains(text, "Covered 2 (67%) · Not covered 1 (33%) · Passing 1 (33%) · Failing 1 (33%) · of 3 operations") {
		t.Errorf("text: %v\n%s", err, text)
	}
	if strings.Contains(text, "[=") || strings.Contains(text, "[.") {
		t.Errorf("the coverage bar is gone:\n%s", text)
	}

	out, err := execFull(t, "coverage", "list", "-w", ws, "-f", "json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		Covered, Total, NotCovered, Passing, Failing, NoResult int
	}
	if err := json.Unmarshal([]byte(out), &entries); err != nil || len(entries) != 1 {
		t.Fatalf("json: %v\n%s", err, out)
	}
	e := entries[0]
	if e.Covered != 2 || e.Total != 3 || e.NotCovered != 1 || e.Passing != 1 || e.Failing != 1 || e.NoResult != 0 {
		t.Errorf("totals: %+v", e)
	}
	if e.Covered+e.NotCovered != e.Total || e.Passing+e.Failing != e.Covered {
		t.Errorf("totals must add up: %+v", e)
	}
}

func TestCoverageQuality_TextAndJSONCarryTheFourTotals(t *testing.T) {
	ws, spec := coverageWorkspace(t)

	text, _ := execFull(t, "coverage", "quality", "-w", ws, spec)
	if !strings.Contains(text, "Covered 2 (67%) · Not covered 1 (33%) · Passing 1 (33%) · Failing 1 (33%) · of 3 operations") {
		t.Errorf("text:\n%s", text)
	}

	out, _ := execFull(t, "coverage", "quality", "--fail-on-gaps", "-w", ws, "-f", "json", spec)
	var got struct {
		Totals struct {
			Total, Covered, NotCovered, Passing, Failing, NoResult int
		} `json:"totals"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if got.Totals.Total != 3 || got.Totals.Covered != 2 || got.Totals.NotCovered != 1 || got.Totals.Passing != 1 || got.Totals.Failing != 1 {
		t.Errorf("totals: %+v", got.Totals)
	}
}
