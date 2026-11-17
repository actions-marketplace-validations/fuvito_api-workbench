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
	if lines[0] != "Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes" || len(lines) != 5 {
		t.Fatalf("header/rows: %q", lines)
	}
	want := []string{
		"GET,/health,Not Covered,,,Never exercised,",
		"GET,/users,Covered,Pass,2026-11-17T11:56:00Z,Happy-path only,",
		"POST,/users,Covered,Fail,2026-11-17T09:00:00Z,No auth tested; Errors never triggered,400 409",
		"DELETE,/users/{id},Not Covered,,,Never exercised,",
	}
	for i, w := range want {
		if lines[i+1] != w {
			t.Errorf("row %d\n got %s\nwant %s", i+1, lines[i+1], w)
		}
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
		"# Contracts Report — users-api.yaml", "**2 / 4 operations covered (50%)**",
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
		"<!DOCTYPE html>", "Contracts Report — users-api.yaml", ">50%<", "2 / 4 operations covered",
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
	if !strings.Contains(s, "color:#f59e0b") {
		t.Error("50% coverage uses the amber bar colour")
	}

	r.Percent, r.Covered = 90, 4
	high, _ := reporting.ContractsHTML(r.SpecFile, r, &coverage.QualityReport{}, contractsNow)
	if !strings.Contains(string(high), "color:#10b981") || !strings.Contains(string(high), "All quality signals clear.") {
		t.Error("90% is green and an empty quality report says all clear")
	}
	low, _ := reporting.ContractsHTML(r.SpecFile, coverage.Report{Percent: 10}, nil, contractsNow)
	if !strings.Contains(string(low), "color:#ef4444") || strings.Contains(string(low), "Quality Gaps") {
		t.Error("10% is red and no quality data means no quality section")
	}
}

func TestContractsList(t *testing.T) {
	r, _ := contractsFixture()
	other := coverage.Report{SpecFile: "specs/b,c.yaml", Covered: 1, Total: 1, Percent: 100, TotalRuns: 1, PassedRuns: 1}
	csv := string(reporting.ContractsListCSV([]coverage.Report{r, other}))
	if !strings.HasPrefix(csv, "Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run\n") ||
		!strings.Contains(csv, "specs/users-api.yaml,2,4,50,3,2,1,2026-11-17T11:56:00Z") ||
		!strings.Contains(csv, `"specs/b,c.yaml",1,1,100,1,1,0,`) {
		t.Errorf("csv:\n%s", csv)
	}
	md := string(reporting.ContractsListMarkdown([]coverage.Report{r}, contractsNow))
	if !strings.Contains(md, "| `specs/users-api.yaml` | 2 / 4 | 50% | 3 | 2 | 1 | 4m ago |") {
		t.Errorf("markdown:\n%s", md)
	}
	if !strings.Contains(string(reporting.ContractsListMarkdown(nil, contractsNow)), "No coverage has been recorded yet") {
		t.Error("empty list message")
	}
	html, err := reporting.ContractsListHTML([]coverage.Report{r}, contractsNow)
	if err != nil || !strings.Contains(string(html), "specs/users-api.yaml") || !strings.Contains(string(html), "2 / 4") {
		t.Errorf("html: %v", err)
	}
	empty, _ := reporting.ContractsListHTML(nil, contractsNow)
	if !strings.Contains(string(empty), "No coverage has been recorded yet.") {
		t.Error("empty html")
	}
}
