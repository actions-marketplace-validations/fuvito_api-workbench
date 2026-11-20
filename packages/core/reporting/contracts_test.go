package reporting_test

import (
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

var contractsNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

func contractsFixture() (coverage.Report, *coverage.QualityReport) {
	r := coverage.Report{
		SpecFile: "specs/users-api.yaml", Covered: 2, Total: 4, Percent: 50,
		TotalRuns: 3, PassedRuns: 2, FailedRuns: 1, LastRunAt: contractsNow.Add(-4 * time.Minute),
		Exercised: []string{"GET /users", "POST /users"},
		Missed:    []string{"DELETE /users/{id}", "GET /health"},
		OperationResults: map[string]coverage.OperationResult{
			"GET /users":  {LastPassed: true, LastRunAt: contractsNow.Add(-4 * time.Minute)},
			"POST /users": {LastPassed: false, LastRunAt: contractsNow.Add(-3 * time.Hour)},
		},
	}
	q := &coverage.QualityReport{
		NeverExercised:       []string{"DELETE /users/{id}", "GET /health"},
		HappyPathOnly:        []string{"GET /users"},
		SecuredNeverAuthed:   []string{"POST /users"},
		ErrorsNeverTriggered: []coverage.ErrorGap{{Operation: "POST /users", Missing: []int{400, 409}}},
	}
	return r, q
}

func TestContractsCSV(t *testing.T) {
	r, q := contractsFixture()
	lines := strings.Split(strings.TrimSpace(string(reporting.ContractsCSV(r, q))), "\n")
	if lines[0] != "Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes,Status" || len(lines) != 5 {
		t.Fatalf("header/rows: %q", lines)
	}
	want := []string{
		"GET,/health,Not Covered,,,Never exercised,,Not covered",
		"GET,/users,Covered,Pass,2026-11-17T11:56:00Z,Happy-path only,,Passing",
		"POST,/users,Covered,Fail,2026-11-17T09:00:00Z,No auth tested; Errors never triggered,400 409,Failing",
		"DELETE,/users/{id},Not Covered,,,Never exercised,,Not covered",
	}
	for i, w := range want {
		if lines[i+1] != w {
			t.Errorf("row %d\n got %s\nwant %s", i+1, lines[i+1], w)
		}
	}
	r.Exercised = []string{"GET /users", "POST /users"}
	r.Exercised, r.OperationResults = r.Exercised[:1], map[string]coverage.OperationResult{}
	if got := string(reporting.ContractsCSV(r, nil)); !strings.Contains(got, "GET,/users,Covered,,,,,Covered (no result)") {
		t.Errorf("a covered operation without a result is labelled as such: %s", got)
	}
	r.Missed = []string{`GET /say,"hi"`}
	r.Exercised = nil
	if got := string(reporting.ContractsCSV(r, nil)); !strings.Contains(got, `"/say,""hi"""`) {
		t.Errorf("cells with commas and quotes are quoted: %s", got)
	}
}

func TestContractsMarkdown(t *testing.T) {
	r, q := contractsFixture()
	md := string(reporting.ContractsMarkdown(r.SpecFile, r, q, contractsNow))
	for _, want := range []string{
		"# Contracts Report — users-api.yaml",
		"| Covered | Not covered | Passing | Failing |", "| 2 (50%) | 2 (50%) | 1 (25%) | 1 (25%) |",
		"Covered + Not covered = 4 operations; Passing + Failing = Covered.", "## Scenario runs",
		"| Total runs | 3 |", "| Failed runs | 1 |",
		"| `GET` | `/users` | ✅ Covered | Pass | 4m ago |",
		"| `POST` | `/users` | ✅ Covered | Fail | 3h ago |",
		"| `GET` | `/health` | — | — | — |",
		"### Happy-path only (1)", "- `GET /users`",
		"### Error responses never triggered (1)", "| `POST /users` | 400, 409 |",
		"### Secured — never tested with auth (1)",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}

	if strings.Contains(md, "operations covered (") || strings.Contains(md, "no recorded result") {
		t.Errorf("the old headline is gone and a fully recorded spec has no missing-result note:\n%s", md)
	}
	noResults := r
	noResults.OperationResults = nil
	if got := string(reporting.ContractsMarkdown(r.SpecFile, noResults, nil, contractsNow)); !strings.Contains(got, "_2 covered operation(s) have no recorded result yet._") {
		t.Errorf("missing-result note:\n%s", got)
	}

	clear := string(reporting.ContractsMarkdown(r.SpecFile, r, &coverage.QualityReport{}, contractsNow))
	if !strings.Contains(clear, "_All quality signals clear._") {
		t.Errorf("no gaps:\n%s", clear)
	}
	if none := string(reporting.ContractsMarkdown(r.SpecFile, r, nil, contractsNow)); strings.Contains(none, "Quality Gaps") {
		t.Error("without quality data there is no quality section")
	}
}

func TestContractsHTML(t *testing.T) {
	r, q := contractsFixture()
	r.Missed = append(r.Missed, `GET /<script>alert(1)</script>`)
	b, err := reporting.ContractsHTML(r.SpecFile, r, q, contractsNow)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		"<!DOCTYPE html>", "Contracts Report — users-api.yaml",
		`<div class="l">Covered</div><div class="n" style="color:#111827">2<small>50%</small></div>`,
		`<div class="l">Not covered</div>`, `<div class="l">Passing</div>`, `<div class="l">Failing</div><div class="n" style="color:#dc2626">1<small>25%</small></div>`,
		"Scenario runs",
		"3</strong> runs", "1</strong> failed", "✓ Covered", "✓ Pass", "✗ Fail",
		"Happy-path only (1)", "Error responses never triggered (1)", `<span class="code">400</span>`, "Secured — never tested with auth (1)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(s, "<script>alert") || !strings.Contains(s, "&lt;script&gt;") {
		t.Errorf("paths from a spec must be escaped:\n%s", s)
	}
	if strings.Contains(s, "operations covered") || strings.Contains(s, "border-radius:999px") {
		t.Error("the single coverage percentage and its full-width bar are gone")
	}

	// a fully covered spec still shows its failing operation, and zero tiles are muted
	full := coverage.Report{Total: 1, Covered: 1, Percent: 100, Exercised: []string{"GET /a"},
		OperationResults: map[string]coverage.OperationResult{"GET /a": {LastPassed: false}}}
	fb, _ := reporting.ContractsHTML(r.SpecFile, full, &coverage.QualityReport{}, contractsNow)
	fs := string(fb)
	if !strings.Contains(fs, `<div class="l">Failing</div><div class="n" style="color:#dc2626">1<small>100%</small></div>`) ||
		!strings.Contains(fs, `<div class="l">Not covered</div><div class="n" style="color:#9ca3af">0<small>0%</small></div>`) ||
		!strings.Contains(fs, "All quality signals clear.") {
		t.Errorf("100%% coverage with a failure:\n%s", fs)
	}
	empty, _ := reporting.ContractsHTML(r.SpecFile, coverage.Report{}, nil, contractsNow)
	if strings.Contains(string(empty), "Quality Gaps") || !strings.Contains(string(empty), `<div class="l">Covered</div><div class="n" style="color:#111827">0<small>0%</small></div>`) {
		t.Error("an empty report has zero tiles and no quality section")
	}
}

func TestOutcomesLine(t *testing.T) {
	r, _ := contractsFixture()
	if got := reporting.OutcomesLine(r.Outcomes()); got != "Covered 2 (50%) · Not covered 2 (50%) · Passing 1 (25%) · Failing 1 (25%) · of 4 operations" {
		t.Errorf("line: %s", got)
	}
	if got := reporting.OutcomesLine(coverage.Report{Total: 3, Covered: 2, Exercised: []string{"a", "b"}}.Outcomes()); !strings.Contains(got, "Failing 0 (0%) · No result 2 (67%) · of 3 operations") {
		t.Errorf("covered operations without a result are listed so the parts add up: %s", got)
	}
	if got := reporting.OutcomesLine(coverage.Outcomes{}); got != "Covered 0 (0%) · Not covered 0 (0%) · Passing 0 (0%) · Failing 0 (0%) · of 0 operations" {
		t.Errorf("empty: %s", got)
	}
}

func TestContractsList(t *testing.T) {
	r, _ := contractsFixture()
	other := coverage.Report{SpecFile: "specs/b,c.yaml", Covered: 1, Total: 1, Percent: 100, TotalRuns: 1, PassedRuns: 1}
	csv := string(reporting.ContractsListCSV([]coverage.Report{r, other}))
	if !strings.HasPrefix(csv, "Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run,Not Covered,Passing,Failing\n") ||
		!strings.Contains(csv, "specs/users-api.yaml,2,4,50,3,2,1,2026-11-17T11:56:00Z,2,1,1") ||
		!strings.Contains(csv, `"specs/b,c.yaml",1,1,100,1,1,0,,0,0,0`) {
		t.Errorf("csv:\n%s", csv)
	}
	md := string(reporting.ContractsListMarkdown([]coverage.Report{r}, contractsNow))
	if !strings.Contains(md, "| `specs/users-api.yaml` | 2 (50%) | 2 (50%) | 1 (25%) | 1 (25%) | 3 · 2 passed · 1 failed | 4m ago |") {
		t.Errorf("markdown:\n%s", md)
	}
	if !strings.Contains(string(reporting.ContractsListMarkdown(nil, contractsNow)), "No coverage has been recorded yet") {
		t.Error("empty list message")
	}
	html, err := reporting.ContractsListHTML([]coverage.Report{r}, contractsNow)
	if err != nil || !strings.Contains(string(html), "specs/users-api.yaml") || !strings.Contains(string(html), "<th>Not covered</th>") || !strings.Contains(string(html), `style="color:#dc2626;font-weight:600">1 <small>25%</small>`) {
		t.Errorf("html: %v", err)
	}
	empty, _ := reporting.ContractsListHTML(nil, contractsNow)
	if !strings.Contains(string(empty), "No coverage has been recorded yet.") {
		t.Error("empty html")
	}
}
