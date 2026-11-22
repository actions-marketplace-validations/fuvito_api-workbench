package reporting

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

// ContractsFormats are the formats a Contracts (accumulated coverage) report can take.
var ContractsFormats = []string{"csv", "markdown", "html"}

func splitOp(op string) (method, path string) {
	if i := strings.IndexByte(op, ' '); i >= 0 {
		return op[:i], op[i+1:]
	}
	return op, ""
}

func specLabel(spec string) string {
	s := strings.ReplaceAll(spec, "\\", "/")
	if i := strings.LastIndexByte(s, '/'); i >= 0 && i < len(s)-1 {
		return s[i+1:]
	}
	return s
}

type contractsOp struct {
	Op, Method, Path  string
	Covered           bool
	Result            string // "Pass", "Fail" or ""
	LastRunAt         time.Time
	LastMs, SlowestMs int64 // response times in milliseconds; 0 when none was recorded
}

// contractsOps lists every operation of the spec, covered or not, sorted by path then method.
func contractsOps(r coverage.Report) []contractsOp {
	covered := map[string]bool{}
	for _, op := range r.Exercised {
		covered[op] = true
	}
	all := append(append([]string{}, r.Exercised...), r.Missed...)
	out := make([]contractsOp, 0, len(all))
	for _, op := range all {
		m, p := splitOp(op)
		c := contractsOp{Op: op, Method: m, Path: p, Covered: covered[op]}
		if res, ok := r.OperationResults[op]; ok {
			c.Result = "Fail"
			if res.LastPassed {
				c.Result = "Pass"
			}
			c.LastRunAt = res.LastRunAt
			c.LastMs, c.SlowestMs = res.LastDurationMs, res.SlowestDurationMs
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// OutcomesLine is the four totals on one line, for text output:
// "Covered 10 (71%) · Not covered 4 (29%) · Passing 9 (64%) · Failing 1 (7%) · of 14 operations".
func OutcomesLine(o coverage.Outcomes) string {
	part := func(label string, n int) string {
		return fmt.Sprintf("%s %d (%d%%)", label, n, coverage.Share(n, o.Total))
	}
	parts := []string{part("Covered", o.Covered), part("Not covered", o.NotCovered), part("Passing", o.Passing), part("Failing", o.Failing)}
	if o.NoResult > 0 {
		// covered operations whose last result was not recorded, so the parts still add up
		parts = append(parts, part("No result", o.NoResult))
	}
	return strings.Join(append(parts, fmt.Sprintf("of %d operations", o.Total)), " · ")
}

// statusLabel is the one-word state of an operation, using the names of the Contracts totals.
func statusLabel(op contractsOp) string {
	switch {
	case !op.Covered:
		return "Not covered"
	case op.Result == "Pass":
		return "Passing"
	case op.Result == "Fail":
		return "Failing"
	}
	return "Covered (no result)"
}

type outcomeTile struct {
	Label, Hint string
	Count, Pct  int
	Total       int
	Value, Edge template.CSS
}

// outcomeTiles are the four totals in display order: Covered | Not covered, then Passing | Failing.
// Passing, Failing and Not covered are muted at zero, so a clean zero is not celebrated.
func outcomeTiles(o coverage.Outcomes) []outcomeTile {
	const muted, edgeIdle = "#9ca3af", "#e5e7eb"
	tile := func(label, hint string, count int, value, edge string, always bool) outcomeTile {
		if !always && count == 0 {
			value, edge = muted, edgeIdle
		}
		return outcomeTile{Label: label, Hint: hint, Count: count, Pct: coverage.Share(count, o.Total), Total: o.Total,
			Value: template.CSS(value), Edge: template.CSS(edge)}
	}
	return []outcomeTile{
		tile("Covered", "called by a scenario", o.Covered, "#111827", "#9ca3af", true),
		tile("Not covered", "no scenario calls them", o.NotCovered, "#b45309", "#f59e0b", false),
		tile("Passing", "covered, last result passed", o.Passing, "#059669", "#10b981", false),
		tile("Failing", "covered, last result failed", o.Failing, "#dc2626", "#ef4444", false),
	}
}

func csvCell(v string) string {
	if strings.ContainsAny(v, ",\"\n") {
		return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
	}
	return v
}

func qualityColumns(op contractsOp, q *coverage.QualityReport) (flags, missing string) {
	if !op.Covered {
		return "Never exercised", ""
	}
	if q == nil {
		return "", ""
	}
	var f []string
	for _, o := range q.HappyPathOnly {
		if o == op.Op {
			f = append(f, "Happy-path only")
		}
	}
	for _, o := range q.SecuredNeverAuthed {
		if o == op.Op {
			f = append(f, "No auth tested")
		}
	}
	for _, g := range q.ErrorsNeverTriggered {
		if g.Operation == op.Op {
			f = append(f, "Errors never triggered")
			codes := make([]string, len(g.Missing))
			for i, c := range g.Missing {
				codes[i] = fmt.Sprint(c)
			}
			missing = strings.Join(codes, " ")
		}
	}
	return strings.Join(f, "; "), missing
}

func isoOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ContractsCSV renders one spec's operations and quality flags as CSV.
func ContractsCSV(r coverage.Report, q *coverage.QualityReport) []byte {
	var b strings.Builder
	b.WriteString("Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes,Status,Last ms,Slowest ms\n")
	for _, op := range contractsOps(r) {
		covered := "Not Covered"
		result := ""
		if op.Covered {
			covered = "Covered"
			result = op.Result
		}
		flags, missing := qualityColumns(op, q)
		cells := []string{op.Method, op.Path, covered, result, isoOrEmpty(op.LastRunAt), flags, missing, statusLabel(op), msOrEmpty(op.LastMs), msOrEmpty(op.SlowestMs)}
		for i, c := range cells {
			cells[i] = csvCell(c)
		}
		b.WriteString(strings.Join(cells, ",") + "\n")
	}
	return []byte(b.String())
}

func relTime(t, now time.Time) string {
	if t.IsZero() {
		return "—"
	}
	s := int(now.Sub(t).Seconds())
	switch {
	case s < 60:
		return "just now"
	case s < 3600:
		return fmt.Sprintf("%dm ago", s/60)
	case s < 86400:
		return fmt.Sprintf("%dh ago", s/3600)
	}
	return fmt.Sprintf("%dd ago", s/86400)
}

func hasQualityGaps(q *coverage.QualityReport) bool {
	return len(q.HappyPathOnly)+len(q.SecuredNeverAuthed)+len(q.ErrorsNeverTriggered) > 0
}

func joinCodes(codes []int) string {
	s := make([]string, len(codes))
	for i, c := range codes {
		s[i] = fmt.Sprint(c)
	}
	return strings.Join(s, ", ")
}

// ContractsMarkdown renders one spec's coverage and quality gaps as Markdown for a wiki or a
// pull-request description.
func ContractsMarkdown(spec string, r coverage.Report, q *coverage.QualityReport, now time.Time) []byte {
	o := r.Outcomes()
	cell := func(n int) string { return fmt.Sprintf("%d (%d%%)", n, coverage.Share(n, o.Total)) }
	var l []string
	add := func(s ...string) { l = append(l, s...) }

	add(fmt.Sprintf("# Contracts Report — %s", mdEscape(specLabel(spec))), "",
		fmt.Sprintf("Generated: %s", now.Format("2006-01-02 15:04")), "",
		"## Summary", "",
		"| Covered | Not covered | Passing | Failing |", "|--------:|------------:|--------:|--------:|",
		fmt.Sprintf("| %s | %s | %s | %s |", cell(o.Covered), cell(o.NotCovered), cell(o.Passing), cell(o.Failing)), "",
		fmt.Sprintf("Covered + Not covered = %d operations; Passing + Failing = Covered. Passing and failing are each operation's most recent result.", o.Total))
	if o.NoResult > 0 {
		add("", fmt.Sprintf("_%d covered operation(s) have no recorded result yet._", o.NoResult))
	}
	add("", "## Scenario runs", "",
		"| Metric | Value |", "|--------|-------|",
		fmt.Sprintf("| Total runs | %d |", r.TotalRuns),
		fmt.Sprintf("| Passed runs | %d |", r.PassedRuns),
		fmt.Sprintf("| Failed runs | %d |", r.FailedRuns))
	if !r.LastRunAt.IsZero() {
		add(fmt.Sprintf("| Last run | %s |", r.LastRunAt.Format("2006-01-02 15:04")))
	}
	add("", "## Operations", "",
		"| Method | Path | Coverage | Last Result | Last Run | Last ms | Slowest ms |", "|--------|------|----------|-------------|---------|---------|------------|")
	for _, op := range contractsOps(r) {
		cov, result := "—", "—"
		if op.Covered {
			cov = "✅ Covered"
		}
		if op.Result != "" {
			result = op.Result
		}
		add(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s | %s | %s |", op.Method, op.Path, cov, result, relTime(op.LastRunAt, now), msOrDash(op.LastMs), msOrDash(op.SlowestMs)))
	}

	if q != nil {
		add("", "## Quality Gaps")
		if !hasQualityGaps(q) {
			add("", "_All quality signals clear._")
		}
		if len(q.HappyPathOnly) > 0 {
			add("", fmt.Sprintf("### Happy-path only (%d)", len(q.HappyPathOnly)), "",
				"Only 2xx responses observed — declared error paths untested.", "")
			for _, op := range q.HappyPathOnly {
				add(fmt.Sprintf("- `%s`", op))
			}
		}
		if len(q.ErrorsNeverTriggered) > 0 {
			add("", fmt.Sprintf("### Error responses never triggered (%d)", len(q.ErrorsNeverTriggered)), "",
				"| Operation | Missing status codes |", "|-----------|---------------------|")
			for _, g := range q.ErrorsNeverTriggered {
				add(fmt.Sprintf("| `%s` | %s |", g.Operation, joinCodes(g.Missing)))
			}
		}
		if len(q.SecuredNeverAuthed) > 0 {
			add("", fmt.Sprintf("### Secured — never tested with auth (%d)", len(q.SecuredNeverAuthed)), "",
				"Operations that require authentication but were always called without credentials.", "")
			for _, op := range q.SecuredNeverAuthed {
				add(fmt.Sprintf("- `%s`", op))
			}
		}
	}
	return []byte(strings.Join(l, "\n") + "\n")
}

var methodColor = map[string]string{
	"GET": "#0ea5e9", "POST": "#10b981", "PUT": "#f59e0b", "PATCH": "#f97316",
	"DELETE": "#ef4444", "HEAD": "#a855f7", "OPTIONS": "#ec4899",
}

type contractsHTMLOp struct {
	Method, Path string
	Color, Bg    template.CSS
	Covered      bool
	Result       string
	When         string
	LastMs       string
	SlowestMs    string
}

type contractsHTMLView struct {
	Label                string
	Generated, LastRun   string
	Tiles                []outcomeTile
	NoResult             int
	Runs, Passed, Failed int
	Ops                  []contractsHTMLOp
	HasQuality           bool
	AllClear             bool
	Happy, Secured       []string
	Gaps                 []contractsHTMLGap
}

type contractsHTMLGap struct {
	Operation string
	Missing   []int
}

var contractsTmpl = template.Must(template.New("contracts").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Contracts Report — {{.Label}}</title>
<style>
*{box-sizing:border-box}
body{font-family:system-ui,-apple-system,sans-serif;max-width:960px;margin:0 auto;padding:2rem;color:#111;font-size:14px;line-height:1.5}
h1{font-size:1.4rem;margin:0 0 .25rem}
h2{font-size:1.1rem;border-bottom:1px solid #e5e7eb;padding-bottom:.3rem;margin-top:2rem}
h3{margin:1.5rem 0 .4rem;font-size:.95rem}
table{width:100%;border-collapse:collapse;margin-top:.5rem}
th{text-align:left;padding:5px 8px;border-bottom:2px solid #e5e7eb;font-size:11px;text-transform:uppercase;letter-spacing:.05em;color:#6b7280;white-space:nowrap}
td{padding:5px 8px;border-bottom:1px solid #f3f4f6;vertical-align:middle}
tr:last-child td{border-bottom:none}
.tiles{display:grid;grid-template-columns:1fr 1fr;gap:1.5rem}
.pair{display:grid;grid-template-columns:1fr 1fr;gap:.75rem}
.tile{border:1px solid #e5e7eb;border-left-width:4px;border-radius:6px;padding:.6rem .8rem}
.tile .l{font-size:10px;text-transform:uppercase;letter-spacing:.08em;color:#6b7280;font-weight:600}
.tile .n{font-size:1.9rem;font-weight:700;line-height:1.2}.tile .n small{font-size:.9rem;font-weight:400;color:#6b7280;margin-left:.4rem}
.tile .h{font-size:11px;color:#6b7280}
.stat{display:inline-block;margin-right:1.5rem;font-size:.85rem;color:#6b7280}
.stat strong{color:#111}
.mono{font-family:monospace;font-size:12px}
.badge{font-family:monospace;font-weight:700;font-size:11px;padding:1px 5px;border-radius:3px}
.pass{color:#059669;font-weight:600}.fail{color:#dc2626;font-weight:600}.none{color:#9ca3af}
.code{background:#fff3cd;padding:1px 4px;border-radius:3px;font-size:11px;font-weight:600}
ul{margin:0;padding-left:1.5rem}li{font-family:monospace;font-size:12px;padding:1px 0}
@media print{body{padding:0}@page{margin:1.5cm}}
</style>
</head>
<body>
<h1>Contracts Report — {{.Label}}</h1>
<p style="margin:.25rem 0 1rem;font-size:.85rem;color:#6b7280">Generated {{.Generated}}</p>
<div class="tiles">
{{range $i, $t := .Tiles}}{{if eq $i 0}}<div class="pair">{{else if eq $i 2}}</div><div class="pair">{{end}}<div class="tile" style="border-left-color:{{$t.Edge}}"><div class="l">{{$t.Label}}</div><div class="n" style="color:{{$t.Value}}">{{$t.Count}}<small>{{$t.Pct}}%</small></div><div class="h">{{$t.Hint}} · of {{$t.Total}}</div></div>{{end}}</div>
</div>
<p style="margin:.5rem 0 0;font-size:.75rem;color:#6b7280">Covered + Not covered = all operations; Passing + Failing = Covered (each operation's most recent result).{{if .NoResult}} {{.NoResult}} covered operation(s) have no recorded result yet.{{end}}</p>
<div style="margin-top:.75rem">
  <span class="stat" style="text-transform:uppercase;letter-spacing:.08em;font-size:10px;font-weight:600">Scenario runs</span>
  <span class="stat"><strong>{{.Runs}}</strong> run{{if ne .Runs 1}}s{{end}}</span>
  <span class="stat"><strong style="color:#059669">{{.Passed}}</strong> passed</span>
  {{if gt .Failed 0}}<span class="stat"><strong style="color:#dc2626">{{.Failed}}</strong> failed</span>{{end}}
  {{if .LastRun}}<span class="stat">Last run <strong>{{.LastRun}}</strong></span>{{end}}
</div>
<h2>Operations</h2>
<table>
<thead><tr><th>Method</th><th>Path</th><th>Coverage</th><th>Last Result</th><th>Last Run</th><th>Last ms</th><th>Slowest ms</th></tr></thead>
<tbody>
{{range .Ops}}<tr>
<td><span class="badge" style="color:{{.Color}};background:{{.Bg}}">{{.Method}}</span></td>
<td class="mono">{{.Path}}</td>
<td>{{if .Covered}}<span class="pass">✓ Covered</span>{{else}}<span class="none">—</span>{{end}}</td>
<td>{{if eq .Result "Pass"}}<span class="pass">✓ Pass</span>{{else if eq .Result "Fail"}}<span class="fail">✗ Fail</span>{{else}}<span class="none">—</span>{{end}}</td>
<td style="color:#6b7280;font-size:12px">{{.When}}</td>
<td class="mono">{{.LastMs}}</td>
<td class="mono">{{.SlowestMs}}</td>
</tr>
{{end}}</tbody>
</table>
{{if .HasQuality}}<h2>Quality Gaps</h2>
{{if .AllClear}}<p class="pass" style="font-size:.9rem">All quality signals clear.</p>{{end}}
{{if .Happy}}<h3 style="color:#b45309">Happy-path only ({{len .Happy}})</h3>
<p style="margin:0 0 .5rem;font-size:.8rem;color:#6b7280">Only 2xx responses observed — declared error paths untested.</p>
<ul>{{range .Happy}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{if .Gaps}}<h3 style="color:#b45309">Error responses never triggered ({{len .Gaps}})</h3>
<table><thead><tr><th>Operation</th><th>Missing codes</th></tr></thead><tbody>
{{range .Gaps}}<tr><td class="mono">{{.Operation}}</td><td>{{range .Missing}}<span class="code">{{.}}</span> {{end}}</td></tr>
{{end}}</tbody></table>{{end}}
{{if .Secured}}<h3 style="color:#1d4ed8">Secured — never tested with auth ({{len .Secured}})</h3>
<p style="margin:0 0 .5rem;font-size:.8rem;color:#6b7280">Operations requiring credentials but always called without.</p>
<ul>{{range .Secured}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{end}}
</body>
</html>
`))

// ContractsHTML renders one spec's coverage and quality gaps as a self-contained, print-ready
// HTML page. All text is escaped.
func ContractsHTML(spec string, r coverage.Report, q *coverage.QualityReport, now time.Time) ([]byte, error) {
	o := r.Outcomes()
	v := contractsHTMLView{
		Label: specLabel(spec), Generated: now.Format("2006-01-02 15:04"),
		Tiles: outcomeTiles(o), NoResult: o.NoResult,
		Runs: r.TotalRuns, Passed: r.PassedRuns, Failed: r.FailedRuns,
	}
	if !r.LastRunAt.IsZero() {
		v.LastRun = r.LastRunAt.Format("2006-01-02 15:04")
	}
	for _, op := range contractsOps(r) {
		color := methodColor[op.Method]
		if color == "" {
			color = "#6b7280"
		}
		v.Ops = append(v.Ops, contractsHTMLOp{
			Method: op.Method, Path: op.Path, Covered: op.Covered, Result: op.Result,
			Color: template.CSS(color), Bg: template.CSS(color + "18"), When: relTime(op.LastRunAt, now),
			LastMs: msOrDash(op.LastMs), SlowestMs: msOrDash(op.SlowestMs),
		})
	}
	if q != nil {
		v.HasQuality = true
		v.AllClear = !hasQualityGaps(q)
		v.Happy, v.Secured = q.HappyPathOnly, q.SecuredNeverAuthed
		for _, g := range q.ErrorsNeverTriggered {
			v.Gaps = append(v.Gaps, contractsHTMLGap{Operation: g.Operation, Missing: g.Missing})
		}
	}
	var buf bytes.Buffer
	if err := contractsTmpl.Execute(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ContractsListMarkdown, ContractsListCSV and ContractsListHTML render the summary of every
// spec that has accumulated coverage, one row per spec.

// ContractsListCSV is the per-spec summary as CSV.
func ContractsListCSV(reports []coverage.Report) []byte {
	var b strings.Builder
	b.WriteString("Spec,Covered,Total,Percent,Runs,Passed,Failed,Last Run,Not Covered,Passing,Failing\n")
	for _, r := range reports {
		o := r.Outcomes()
		cells := []string{r.SpecFile, fmt.Sprint(r.Covered), fmt.Sprint(r.Total), fmt.Sprintf("%.0f", r.Percent),
			fmt.Sprint(r.TotalRuns), fmt.Sprint(r.PassedRuns), fmt.Sprint(r.FailedRuns), isoOrEmpty(r.LastRunAt),
			fmt.Sprint(o.NotCovered), fmt.Sprint(o.Passing), fmt.Sprint(o.Failing)}
		for i, c := range cells {
			cells[i] = csvCell(c)
		}
		b.WriteString(strings.Join(cells, ",") + "\n")
	}
	return []byte(b.String())
}

// ContractsListMarkdown is the per-spec summary as a Markdown table.
func ContractsListMarkdown(reports []coverage.Report, now time.Time) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Contracts coverage\n\nGenerated: %s\n\n", now.Format("2006-01-02 15:04"))
	if len(reports) == 0 {
		b.WriteString("_No coverage has been recorded yet._\n")
		return []byte(b.String())
	}
	b.WriteString("| Spec | Covered | Not covered | Passing | Failing | Scenario runs | Last run |\n|------|--------:|------------:|--------:|--------:|---------------|----------|\n")
	for _, r := range reports {
		o := r.Outcomes()
		cell := func(n int) string { return fmt.Sprintf("%d (%d%%)", n, coverage.Share(n, o.Total)) }
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %d · %d passed · %d failed | %s |\n",
			r.SpecFile, cell(o.Covered), cell(o.NotCovered), cell(o.Passing), cell(o.Failing),
			r.TotalRuns, r.PassedRuns, r.FailedRuns, relTime(r.LastRunAt, now))
	}
	b.WriteString("\nCovered + Not covered = all operations in the spec; Passing + Failing = Covered. Percentages are shares of all operations.\n")
	return []byte(b.String())
}

var contractsListTmpl = template.Must(template.New("list").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><title>Contracts coverage</title>
<style>body{font-family:system-ui,sans-serif;max-width:960px;margin:0 auto;padding:2rem;color:#111;font-size:14px}
table{width:100%;border-collapse:collapse}th{text-align:left;padding:5px 8px;border-bottom:2px solid #e5e7eb;font-size:11px;text-transform:uppercase;color:#6b7280}
td{padding:5px 8px;border-bottom:1px solid #f3f4f6}small{color:#6b7280}.mono{font-family:monospace;font-size:12px}@media print{body{padding:0}}</style></head>
<body><h1>Contracts coverage</h1><p style="color:#6b7280;font-size:.85rem">Generated {{.Generated}}</p>
{{if .Rows}}<table><thead><tr><th>Spec</th><th>Covered</th><th>Not covered</th><th>Passing</th><th>Failing</th><th>Scenario runs</th><th>Last run</th></tr></thead><tbody>
{{range .Rows}}<tr><td class="mono">{{.Spec}}</td><td>{{.Covered}} <small>{{.CoveredPct}}%</small></td><td>{{.NotCovered}} <small>{{.NotCoveredPct}}%</small></td><td{{if .Passing}} style="color:#059669;font-weight:600"{{end}}>{{.Passing}} <small>{{.PassingPct}}%</small></td><td{{if .Failing}} style="color:#dc2626;font-weight:600"{{end}}>{{.Failing}} <small>{{.FailingPct}}%</small></td><td>{{.Runs}} · {{.Passed}} passed · {{.Failed}} failed</td><td>{{.When}}</td></tr>
{{end}}</tbody></table>
<p style="color:#6b7280;font-size:.8rem">Covered + Not covered = all operations in the spec; Passing + Failing = Covered. Percentages are shares of all operations.</p>{{else}}<p>No coverage has been recorded yet.</p>{{end}}
</body></html>
`))

// ContractsListHTML is the per-spec summary as an HTML page.
func ContractsListHTML(reports []coverage.Report, now time.Time) ([]byte, error) {
	type row struct {
		Spec                                              string
		Covered, NotCovered, Passing, Failing             int
		CoveredPct, NotCoveredPct, PassingPct, FailingPct int
		Runs, Passed, Failed                              int
		When                                              string
	}
	rows := make([]row, 0, len(reports))
	for _, r := range reports {
		o := r.Outcomes()
		sh := func(n int) int { return coverage.Share(n, o.Total) }
		rows = append(rows, row{
			Spec: r.SpecFile, Covered: o.Covered, NotCovered: o.NotCovered, Passing: o.Passing, Failing: o.Failing,
			CoveredPct: sh(o.Covered), NotCoveredPct: sh(o.NotCovered), PassingPct: sh(o.Passing), FailingPct: sh(o.Failing),
			Runs: r.TotalRuns, Passed: r.PassedRuns, Failed: r.FailedRuns, When: relTime(r.LastRunAt, now),
		})
	}
	var buf bytes.Buffer
	err := contractsListTmpl.Execute(&buf, struct {
		Generated string
		Rows      []row
	}{now.Format("2006-01-02 15:04"), rows})
	return buf.Bytes(), err
}

func msOrEmpty(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return strconv.FormatInt(ms, 10)
}

func msOrDash(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	return strconv.FormatInt(ms, 10)
}
