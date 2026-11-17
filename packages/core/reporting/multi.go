package reporting

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// SpecCoverage is the coverage of one OpenAPI spec across all the scenarios that used it.
type SpecCoverage struct {
	Spec   string
	Result *openapi.ScenarioCoverageResult
}

// MergeCoverage combines the coverage of several runs against the same spec: an operation
// is covered if any run exercised it.
func MergeCoverage(covs []*openapi.ScenarioCoverageResult) *openapi.ScenarioCoverageResult {
	var first *openapi.ScenarioCoverageResult
	exercised := map[string]openapi.CoveredOperation{}
	universe := map[string]openapi.CoveredOperation{}
	for _, c := range covs {
		if c == nil {
			continue
		}
		if first == nil {
			first = c
		}
		for _, op := range c.Exercised {
			exercised[op.String()] = op
			universe[op.String()] = op
		}
		for _, op := range c.Missed {
			universe[op.String()] = op
		}
	}
	if first == nil {
		return nil
	}
	out := &openapi.ScenarioCoverageResult{Total: first.Total}
	for key, op := range universe {
		if _, ok := exercised[key]; ok {
			out.Exercised = append(out.Exercised, op)
		} else {
			out.Missed = append(out.Missed, op)
		}
	}
	out.Exercised = sortedCovOps(out.Exercised)
	out.Missed = sortedCovOps(out.Missed)
	out.Covered = len(out.Exercised)
	if out.Total > 0 {
		out.Percent = float64(out.Covered) / float64(out.Total) * 100
	}
	return out
}

// CoverageText renders a coverage section. label names the spec when several are shown; an
// empty label gives the plain "Spec coverage" heading.
func CoverageText(cov *openapi.ScenarioCoverageResult, label string) string {
	var b strings.Builder
	heading := "Spec coverage"
	if label != "" {
		heading += " (" + label + ")"
	}
	fmt.Fprintf(&b, "%s: %d / %d operations (%.0f%%)\n", heading, cov.Covered, cov.Total, cov.Percent)
	if exercised := sortedCovOps(cov.Exercised); len(exercised) > 0 {
		b.WriteByte('\n')
		fmt.Fprintln(&b, "Covered:")
		for _, op := range exercised {
			fmt.Fprintf(&b, "  %-7s %s\n", op.Method, op.Path)
		}
	}
	if missed := sortedCovOps(cov.Missed); len(missed) > 0 {
		b.WriteByte('\n')
		fmt.Fprintln(&b, "Not covered:")
		for _, op := range missed {
			fmt.Fprintf(&b, "  %-7s %s\n", op.Method, op.Path)
		}
	}
	return b.String()
}

func multiTotals(results []*scenario.Result) (passedScenarios, steps, assertsPassed, assertsTotal int) {
	for _, r := range results {
		if r.Passed {
			passedScenarios++
		}
		p, f, _ := r.Counts()
		steps += p + f
		ap, af := r.AssertionCounts()
		assertsPassed += ap
		assertsTotal += ap + af
	}
	return
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// TextMulti renders several scenario results as one plain-text report with an overall
// summary, followed by the coverage of each spec when given.
func TextMulti(title string, results []*scenario.Result, cov []SpecCoverage, total time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Run: %s  (%d scenario%s)\n\n", title, len(results), plural(len(results)))
	for _, r := range results {
		b.WriteString(Text(r))
		b.WriteString("\n")
	}
	passed, steps, ap, at := multiTotals(results)
	verdict := "PASSED"
	if passed != len(results) {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "Run %s  %d/%d scenarios passed  (%d steps, %d/%d assertions passed)  %dms\n",
		verdict, passed, len(results), steps, ap, at, total.Milliseconds())
	for _, c := range cov {
		label := ""
		if len(cov) > 1 {
			label = c.Spec
		}
		b.WriteByte('\n')
		b.WriteString(CoverageText(c.Result, label))
	}
	return b.String()
}

// JSONSpecCoverage is the coverage of one spec in a combined JSON report.
type JSONSpecCoverage struct {
	Spec string `json:"spec"`
	JSONScenarioCoverage
}

// JSONMulti renders several scenario results as one machine-readable report.
func JSONMulti(title string, results []*scenario.Result, cov []SpecCoverage, total time.Duration) ([]byte, error) {
	type out struct {
		Run        string             `json:"run"`
		Passed     bool               `json:"passed"`
		DurationMs int64              `json:"durationMs"`
		Scenarios  []json.RawMessage  `json:"scenarios"`
		Coverage   []JSONSpecCoverage `json:"coverage,omitempty"`
	}
	doc := out{Run: title, Passed: true, DurationMs: total.Milliseconds(), Scenarios: []json.RawMessage{}}
	for _, r := range results {
		raw, err := JSON(r)
		if err != nil {
			return nil, err
		}
		doc.Passed = doc.Passed && r.Passed
		doc.Scenarios = append(doc.Scenarios, raw)
	}
	for _, c := range cov {
		doc.Coverage = append(doc.Coverage, JSONSpecCoverage{Spec: c.Spec, JSONScenarioCoverage: *jsonCoverage(c.Result)})
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func jsonCoverage(cov *openapi.ScenarioCoverageResult) *JSONScenarioCoverage {
	jc := &JSONScenarioCoverage{
		Total:     cov.Total,
		Covered:   cov.Covered,
		Percent:   cov.Percent,
		Exercised: make([]JSONCoverageOp, 0, len(cov.Exercised)),
		Missed:    make([]JSONCoverageOp, 0, len(cov.Missed)),
	}
	for _, op := range sortedCovOps(cov.Exercised) {
		jc.Exercised = append(jc.Exercised, JSONCoverageOp{Method: op.Method, Path: op.Path})
	}
	for _, op := range sortedCovOps(cov.Missed) {
		jc.Missed = append(jc.Missed, JSONCoverageOp{Method: op.Method, Path: op.Path})
	}
	return jc
}
