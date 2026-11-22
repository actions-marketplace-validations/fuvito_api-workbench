package reporting

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// Text returns a human-readable scenario result suitable for terminal output.
// Uses plain ASCII (no ANSI codes) so it is safe in CI logs.
func Text(result *scenario.Result) string { return textScenario(result, true) }

// textScenario is Text; showSeed is false when several scenarios share one run seed that the
// combined report prints once.
func textScenario(result *scenario.Result, showSeed bool) string {
	var b strings.Builder
	x := redactorFor(result)

	fmt.Fprintf(&b, "Scenario: %s\n", result.Scenario.Title)
	if result.Scenario.Description != "" {
		fmt.Fprintf(&b, "          %s\n", result.Scenario.Description)
	}
	if showSeed && result.UsedRandomness {
		fmt.Fprintf(&b, "          seed: %d\n", result.Seed)
	}
	for _, v := range result.Variables {
		fmt.Fprintf(&b, "          var %s = %s\n", v.Name, x.variable(v))
	}
	if result.SetupError != "" {
		fmt.Fprintf(&b, "          error: %s\n", x.text(result.SetupError))
	}
	fmt.Fprintln(&b)

	for _, sr := range result.Steps {
		if sr.Skipped {
			fmt.Fprintf(&b, "  SKIP  %s\n", sr.Label)
			continue
		}

		status := "PASS"
		if sr.Error != "" || !sr.Passed {
			status = "FAIL"
		}

		suffix := ""
		if sr.Response != nil {
			suffix = fmt.Sprintf("  %d  %dms", sr.Response.StatusCode, sr.Response.Duration.Milliseconds())
		}

		if sr.Attempts > 1 {
			suffix += fmt.Sprintf("  (%d attempts)", sr.Attempts)
		}
		fmt.Fprintf(&b, "  %s  %s%s\n", status, sr.Label, suffix)

		if sr.Error != "" {
			fmt.Fprintf(&b, "         error: %s\n", x.text(sr.Error))
		}

		for _, ar := range sr.Assertions {
			mark := "✓"
			if !ar.Passed {
				mark = "✗"
			}
			label := assertionLabel(ar.Assertion)
			if ar.Passed {
				fmt.Fprintf(&b, "         %s %s\n", mark, label)
			} else {
				fmt.Fprintf(&b, "         %s %s: %s\n", mark, label, x.text(ar.Message))
			}
		}

		shown := x.extracted(sr.Extracted)
		for _, name := range sortedKeys(shown) {
			fmt.Fprintf(&b, "         → %s = %q\n", name, shown[name])
		}
	}

	fmt.Fprintln(&b)
	if line := notComparedLine(result); line != "" {
		fmt.Fprintln(&b, line)
	}
	passed, failed, skipped := result.Counts()
	apassed, afailed := result.AssertionCounts()

	verdict := "PASSED"
	if !result.Passed {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "%s  %d passed, %d failed, %d skipped  (%d/%d assertions passed)  %dms\n",
		verdict, passed, failed, skipped, apassed, apassed+afailed, result.Duration.Milliseconds())

	return b.String()
}

// notComparedLine names the steps whose response was not compared with the scenario's spec and
// why, so a run that checked nothing cannot be read as a run that found nothing. It is empty when
// every step was compared (or the scenario has no spec).
func notComparedLine(result *scenario.Result) string {
	reasons := map[string]int{}
	total := 0
	for _, sr := range result.Steps {
		if sr.Spec.State == "" {
			continue
		}
		total++
		if sr.Spec.State == scenario.SpecNotChecked {
			reasons[sr.Spec.Reason]++
		}
	}
	if len(reasons) == 0 {
		return ""
	}
	names := make([]string, 0, len(reasons))
	n := 0
	for r, c := range reasons {
		names = append(names, fmt.Sprintf("%s: %d", r, c))
		n += c
	}
	sort.Strings(names)
	return fmt.Sprintf("Spec: %d of %d responses were not compared with the spec (%s)", n, total, strings.Join(names, "; "))
}

func assertionLabel(a domain.Assertion) string {
	op := string(a.Operator)
	if op == "" {
		op = "eq"
	}
	switch a.Type {
	case domain.AssertionStatus:
		return fmt.Sprintf("status %s %v", op, a.Expected)
	case domain.AssertionHeader:
		return fmt.Sprintf("header %s %s %v", a.Path, op, a.Expected)
	case domain.AssertionJSONPath:
		return fmt.Sprintf("jsonpath %s %s %v", a.Path, op, a.Expected)
	case domain.AssertionSchema:
		return "schema"
	case domain.AssertionDuration:
		return fmt.Sprintf("duration %s %v ms", op, a.Expected)
	default:
		if a.Name != "" {
			return a.Name
		}
		return string(a.Type)
	}
}

// JSONReport is the machine-readable output structure for CI/CD pipelines.
type JSONReport struct {
	Scenario   string                `json:"scenario"`
	Passed     bool                  `json:"passed"`
	DurationMs int64                 `json:"durationMs"`
	Seed       *int64                `json:"seed,omitempty"`
	Error      string                `json:"error,omitempty"`
	Variables  []JSONVariable        `json:"variables,omitempty"`
	Steps      []JSONStep            `json:"steps"`
	Coverage   *JSONScenarioCoverage `json:"coverage,omitempty"`
}

// JSONScenarioCoverage is the coverage section in a scenario JSON report.
type JSONScenarioCoverage struct {
	Total     int              `json:"total"`
	Covered   int              `json:"covered"`
	Percent   float64          `json:"percent"`
	Exercised []JSONCoverageOp `json:"exercised"`
	Missed    []JSONCoverageOp `json:"missed"`
}

// JSONVariable is a scenario variable and the value it was given for the run.
type JSONVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type JSONStep struct {
	Name       string            `json:"name"`
	Passed     bool              `json:"passed"`
	Skipped    bool              `json:"skipped,omitempty"`
	Error      string            `json:"error,omitempty"`
	StatusCode int               `json:"statusCode,omitempty"`
	DurationMs int64             `json:"durationMs,omitempty"`
	Attempts   int               `json:"attempts,omitempty"` // only when the step was re-sent
	Extracted  map[string]string `json:"extracted,omitempty"`
	Assertions []JSONAssertion   `json:"assertions,omitempty"`
}

type JSONAssertion struct {
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Path     string `json:"path,omitempty"`
	Operator string `json:"operator,omitempty"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message,omitempty"`
}

// ---- JUnit XML ----

type jUnitTestSuites struct {
	XMLName    xml.Name     `xml:"testsuites"`
	Name       string       `xml:"name,attr"`
	Tests      int          `xml:"tests,attr"`
	Failures   int          `xml:"failures,attr"`
	Errors     int          `xml:"errors,attr"`
	Time       string       `xml:"time,attr"`
	TestSuites []jUnitSuite `xml:"testsuite"`
}

type jUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type jUnitProperties struct {
	Property []jUnitProperty `xml:"property"`
}

type jUnitSuite struct {
	Name       string           `xml:"name,attr"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       string           `xml:"time,attr"`
	Properties *jUnitProperties `xml:"properties,omitempty"`
	Cases      []jUnitCase      `xml:"testcase"`
}

type jUnitCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Skipped   *struct{}     `xml:"skipped"`
	Failure   *jUnitFailure `xml:"failure"`
	Error     *jUnitError   `xml:"error"`
}

type jUnitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type jUnitError struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func secStr(ms int64) string { return fmt.Sprintf("%.3f", float64(ms)/1000.0) }

// JUnit returns a JUnit-compatible XML report.
// Supported by Jenkins (Publish JUnit Results), GitHub Actions (dorny/test-reporter),
// GitLab CI (junit artifact), and CircleCI (store_test_results).
func JUnit(result *scenario.Result) ([]byte, error) {
	suite := junitSuiteFor(result)
	return marshalJUnit(jUnitTestSuites{
		Name:       result.Scenario.Title,
		Tests:      suite.Tests,
		Failures:   suite.Failures,
		Errors:     suite.Errors,
		Time:       suite.Time,
		TestSuites: []jUnitSuite{suite},
	})
}

// JUnitMulti returns one JUnit document with a test suite per scenario, for runs that
// execute several scenarios (for example a bundle). name labels the whole document.
func JUnitMulti(name string, results []*scenario.Result) ([]byte, error) {
	report := jUnitTestSuites{Name: name}
	var totalMs int64
	for _, r := range results {
		suite := junitSuiteFor(r)
		report.Tests += suite.Tests
		report.Failures += suite.Failures
		report.Errors += suite.Errors
		totalMs += r.Duration.Milliseconds()
		report.TestSuites = append(report.TestSuites, suite)
	}
	report.Time = secStr(totalMs)
	return marshalJUnit(report)
}

func marshalJUnit(report jUnitTestSuites) ([]byte, error) {
	out, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func junitSuiteFor(result *scenario.Result) jUnitSuite {
	_, failed, _ := result.Counts()
	x := redactorFor(result)

	suite := jUnitSuite{
		Name:     result.Scenario.Title,
		Tests:    len(result.Steps),
		Failures: failed,
		Time:     secStr(result.Duration.Milliseconds()),
	}

	if result.UsedRandomness {
		suite.Properties = &jUnitProperties{Property: []jUnitProperty{{Name: "seed", Value: strconv.FormatInt(result.Seed, 10)}}}
	}
	if result.SetupError != "" {
		suite.Tests++
		suite.Errors++
		suite.Cases = append(suite.Cases, jUnitCase{
			Name: "scenario variables", ClassName: result.Scenario.Title, Time: "0.000",
			Error: &jUnitError{Message: x.text(result.SetupError), Text: x.text(result.SetupError)},
		})
	}
	for _, sr := range result.Steps {
		stepMs := int64(0)
		if sr.Response != nil {
			stepMs = sr.Response.Duration.Milliseconds()
		}
		tc := jUnitCase{
			Name:      sr.Label,
			ClassName: result.Scenario.Title,
			Time:      secStr(stepMs),
		}
		switch {
		case sr.Skipped:
			skipped := struct{}{}
			tc.Skipped = &skipped
		case sr.Error != "":
			suite.Errors++
			tc.Error = &jUnitError{Message: x.text(sr.Error), Text: x.text(sr.Error)}
		case !sr.Passed:
			var lines []string
			for _, ar := range sr.Assertions {
				if !ar.Passed {
					lines = append(lines, fmt.Sprintf("  ✗ %s: %s", assertionLabel(ar.Assertion), x.text(ar.Message)))
				}
			}
			tc.Failure = &jUnitFailure{
				Message: fmt.Sprintf("%d assertion(s) failed", len(lines)),
				Text:    strings.Join(lines, "\n"),
			}
		}
		suite.Cases = append(suite.Cases, tc)
	}
	return suite
}

// ---- Conformance reporting ----

// ConformText returns a human-readable conformance report suitable for terminal / CI output.
// Pass a non-nil coverage to append a coverage summary section.
func ConformText(result *conformance.ConformResult, coverage *conformance.CoverageReport) string {
	return ConformTextExplained(result, coverage, false)
}

// ConformTextExplained is ConformText that can also explain each failure in plain language (what
// the spec expected, what the server sent, why that fails and what to change). The explanation is
// opt-in so the default output, which scripts and CI logs rely on, stays as it was.
func ConformTextExplained(result *conformance.ConformResult, coverage *conformance.CoverageReport, explainFailures bool) string {
	var b strings.Builder
	x := newRedactor(result.Secrets, false)

	fmt.Fprintf(&b, "%s  %s\n", result.SpecTitle, result.SpecVersion)
	fmt.Fprintf(&b, "Base URL: %s\n\n", x.text(result.BaseURL))

	sent, unanswered, firstErr := 0, 0, ""
	for _, op := range result.Operations {
		if op.Skipped || op.Blocked != "" {
			continue
		}
		sent++
		if op.Error != "" {
			unanswered++
			if firstErr == "" {
				firstErr = op.Error
			}
		}
	}
	nobodyAnswered := sent > 0 && unanswered == sent

	for _, op := range result.Operations {
		path := op.Path
		if op.SpecPath != "" {
			path = op.SpecPath
		}
		if op.Skipped {
			fmt.Fprintf(&b, "  SKIP  %-7s %s\n", op.Method, path)
			continue
		}
		status := "PASS"
		if op.Error != "" || !op.Passed() {
			status = "FAIL"
		}
		suffix := ""
		if op.StatusCode > 0 {
			suffix = fmt.Sprintf("  %d  %dms", op.StatusCode, op.DurationMs)
		}
		fmt.Fprintf(&b, "  %s  %-7s %s%s\n", status, op.Method, path, suffix)
		if op.Error != "" && !nobodyAnswered {
			fmt.Fprintf(&b, "         error: %s\n", x.text(op.Error))
		}
		if op.Blocked != "" {
			fmt.Fprintf(&b, "         ✗ %s\n", x.text(op.Blocked))
		}
		for _, v := range op.RequestViolations {
			fmt.Fprintf(&b, "         ✗ request: %s\n", x.text(v.Error()))
		}
		for _, v := range op.Violations {
			fmt.Fprintf(&b, "         ✗ %s\n", x.text(v.Error()))
		}
		if explainFailures {
			for _, f := range explain(op, x) {
				fmt.Fprintf(&b, "         → %s\n", explanationLine(f))
			}
		}
	}

	fmt.Fprintln(&b)
	if nobodyAnswered {
		fmt.Fprintf(&b, "No operation got an answer from %s: is the server running?\n  First error: %s\n\n", x.text(result.BaseURL), x.text(firstErr))
	}
	passed, failed, skipped := result.Counts()
	total := passed + failed + skipped

	verdict := "PASSED"
	if failed > 0 {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "%s  %d operations  ·  %d passed  ·  %d failed  ·  %d skipped  %dms\n",
		verdict, total, passed, failed, skipped, result.Duration.Milliseconds())
	if n, missing := result.BlockedSummary(); n > 0 {
		fmt.Fprintf(&b, "%s\n", conformance.MissingValuesAdvice(n, missing))
	}

	if coverage != nil {
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "Coverage: %d / %d operations (%.0f%%)\n", coverage.Covered, coverage.Total, coverage.Percent)
		if len(coverage.CoveredOps) > 0 {
			fmt.Fprintln(&b)
			fmt.Fprintln(&b, "Covered:")
			for _, op := range coverage.CoveredOps {
				fmt.Fprintf(&b, "  %-7s %s\n", op.Method, op.Path)
			}
		}
		if len(coverage.NotCoveredOps) > 0 {
			fmt.Fprintln(&b)
			fmt.Fprintln(&b, "Not covered:")
			for _, op := range coverage.NotCoveredOps {
				fmt.Fprintf(&b, "  %-7s %s\n", op.Method, op.Path)
			}
		}
	}

	return b.String()
}

// JSONConformReport is the machine-readable structure for a conformance run.
type JSONConformReport struct {
	SpecTitle   string `json:"specTitle"`
	SpecVersion string `json:"specVersion"`
	BaseURL     string `json:"baseURL"`
	Passed      bool   `json:"passed"`
	DurationMs  int64  `json:"durationMs"`
	// MissingValues lists the placeholders that kept operations from being sent (each once).
	MissingValues []string        `json:"missingValues,omitempty"`
	Operations    []JSONConformOp `json:"operations"`
	Coverage      *JSONCoverage   `json:"coverage,omitempty"`
	// Probes lists the negative probes (only with --negative): each one is a request that breaks
	// the spec on purpose and should be answered with a 4xx.
	Probes []JSONConformProbe `json:"probes,omitempty"`
}

// JSONConformProbe is one negative probe in a conformance JSON report.
type JSONConformProbe struct {
	OperationID string `json:"operationId,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	StatusCode  int    `json:"statusCode,omitempty"`
	DurationMs  int64  `json:"durationMs,omitempty"`
	Passed      bool   `json:"passed"`
	Skipped     bool   `json:"skipped,omitempty"`
	Error       string `json:"error,omitempty"`
}

type JSONConformOp struct {
	OperationID string `json:"operationId,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	StatusCode  int    `json:"statusCode,omitempty"`
	DurationMs  int64  `json:"durationMs,omitempty"`
	Passed      bool   `json:"passed"`
	Skipped     bool   `json:"skipped,omitempty"`
	Error       string `json:"error,omitempty"`
	// Blocked says why the request was not sent (a missing value); Missing names the placeholders.
	Blocked           string   `json:"blocked,omitempty"`
	Missing           []string `json:"missing,omitempty"`
	RequestViolations []string `json:"requestViolations,omitempty"`
	Violations        []string `json:"violations,omitempty"`
	// Findings explain each failure in plain language (added in addition to the fields above,
	// which are unchanged).
	Findings []JSONConformFinding `json:"findings,omitempty"`
}

// JSONConformFinding is the machine-readable form of conformance.Finding.
type JSONConformFinding struct {
	Kind string `json:"kind"`
	// CheckID is the check's ID in the catalog (for example CS-RESP-005); absent when the violation
	// is not recognised.
	CheckID string `json:"checkId,omitempty"`
	// Part and Where say where in the response the problem is (a header name, the status, a body
	// field); Spec says where the expectation comes from in the spec.
	Part     string `json:"part,omitempty"`
	Where    string `json:"where,omitempty"`
	Spec     string `json:"spec,omitempty"`
	Summary  string `json:"summary"`
	Expected string `json:"expected,omitempty"`
	Got      string `json:"got,omitempty"`
	Why      string `json:"why"`
	Fix      string `json:"fix"`
}

// JSONCoverage is the machine-readable coverage summary.
type JSONCoverage struct {
	Total         int              `json:"total"`
	Covered       int              `json:"covered"`
	NotCovered    int              `json:"notCovered"`
	Percent       float64          `json:"percent"`
	CoveredOps    []JSONCoverageOp `json:"coveredOps"`
	NotCoveredOps []JSONCoverageOp `json:"notCoveredOps"`
}

// JSONCoverageOp identifies a single operation in a coverage list.
type JSONCoverageOp struct {
	OperationID string `json:"operationId,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
}

// ConformJSON returns a machine-readable JSON conformance report.
// Pass a non-nil coverage to include a coverage summary in the output and a non-nil neg to include
// the negative probes (a failed probe makes the report not passed, as it does the exit code).
func ConformJSON(result *conformance.ConformResult, coverage *conformance.CoverageReport, neg *conformance.NegativeResult) ([]byte, error) {
	_, failed, _ := result.Counts()
	if neg != nil {
		_, negFailed, _, negErrored := neg.Counts()
		failed += negFailed + negErrored
	}
	x := newRedactor(result.Secrets, false)
	report := JSONConformReport{
		SpecTitle:   result.SpecTitle,
		SpecVersion: result.SpecVersion,
		BaseURL:     x.text(result.BaseURL),
		Passed:      failed == 0,
		DurationMs:  result.Duration.Milliseconds(),
	}
	_, report.MissingValues = result.BlockedSummary()
	for _, op := range result.Operations {
		jo := JSONConformOp{
			OperationID: op.OperationID,
			Method:      op.Method,
			Path:        op.Path,
			StatusCode:  op.StatusCode,
			DurationMs:  op.DurationMs,
			Passed:      op.Passed(),
			Skipped:     op.Skipped,
			Error:       x.text(op.Error),
			Blocked:     x.text(op.Blocked),
			Missing:     op.Missing,
		}
		for _, v := range op.RequestViolations {
			jo.RequestViolations = append(jo.RequestViolations, x.text(v.Error()))
		}
		for _, v := range op.Violations {
			jo.Violations = append(jo.Violations, x.text(v.Error()))
		}
		for _, f := range explain(op, x) {
			jo.Findings = append(jo.Findings, JSONConformFinding{
				Kind: f.Kind, CheckID: f.CheckID, Part: f.Part, Where: f.Where, Spec: f.Spec, Summary: f.Summary, Expected: f.Expected, Got: f.Got, Why: f.Why, Fix: f.Fix,
			})
		}
		report.Operations = append(report.Operations, jo)
	}
	if neg != nil {
		for _, p := range neg.Probes {
			report.Probes = append(report.Probes, JSONConformProbe{
				OperationID: p.OperationID, Method: p.Method, Path: p.Path, Kind: string(p.Kind), Description: x.text(p.Description),
				StatusCode: p.StatusCode, DurationMs: p.DurationMs, Passed: !p.Skipped && p.Error == "" && p.Passed(),
				Skipped: p.Skipped, Error: x.text(p.Error),
			})
		}
	}
	if coverage != nil {
		jc := &JSONCoverage{
			Total:         coverage.Total,
			Covered:       coverage.Covered,
			NotCovered:    coverage.NotCovered,
			Percent:       coverage.Percent,
			CoveredOps:    make([]JSONCoverageOp, 0, len(coverage.CoveredOps)),
			NotCoveredOps: make([]JSONCoverageOp, 0, len(coverage.NotCoveredOps)),
		}
		for _, op := range coverage.CoveredOps {
			jc.CoveredOps = append(jc.CoveredOps, JSONCoverageOp{OperationID: op.OperationID, Method: op.Method, Path: op.Path})
		}
		for _, op := range coverage.NotCoveredOps {
			jc.NotCoveredOps = append(jc.NotCoveredOps, JSONCoverageOp{OperationID: op.OperationID, Method: op.Method, Path: op.Path})
		}
		report.Coverage = jc
	}
	return json.MarshalIndent(report, "", "  ")
}

// ConformJUnit returns a JUnit XML conformance report. A non-nil neg adds one test case per
// negative probe.
func ConformJUnit(result *conformance.ConformResult, neg *conformance.NegativeResult) ([]byte, error) {
	_, failed, _ := result.Counts()

	suite := jUnitSuite{
		Name:     result.SpecTitle,
		Tests:    len(result.Operations),
		Failures: failed,
		Time:     secStr(result.Duration.Milliseconds()),
	}

	x := newRedactor(result.Secrets, false)
	for _, op := range result.Operations {
		tc := jUnitCase{
			Name:      op.Method + " " + op.Path,
			ClassName: result.SpecTitle,
			Time:      secStr(op.DurationMs),
		}
		switch {
		case op.Skipped:
			skipped := struct{}{}
			tc.Skipped = &skipped
		case op.Error != "":
			suite.Errors++
			tc.Error = &jUnitError{Message: x.text(op.Error), Text: x.text(op.Error)}
		case !op.Passed():
			var msgs []string
			if op.Blocked != "" {
				msgs = append(msgs, "  ✗ "+x.text(op.Blocked))
			}
			for _, v := range op.RequestViolations {
				msgs = append(msgs, "  ✗ request: "+x.text(v.Error()))
			}
			for _, v := range op.Violations {
				msgs = append(msgs, "  ✗ "+x.text(v.Error()))
			}
			violations := len(msgs)
			for _, f := range explain(op, x) {
				msgs = append(msgs, "  → "+explanationLine(f))
			}
			tc.Failure = &jUnitFailure{
				Message: fmt.Sprintf("%d violation(s)", violations),
				Text:    strings.Join(msgs, "\n"),
			}
		}
		suite.Cases = append(suite.Cases, tc)
	}

	if neg != nil {
		for _, p := range neg.Probes {
			tc := jUnitCase{Name: "probe: " + x.text(p.Description), ClassName: result.SpecTitle, Time: secStr(p.DurationMs)}
			suite.Tests++
			switch {
			case p.Skipped:
				skipped := struct{}{}
				tc.Skipped = &skipped
			case p.Error != "":
				suite.Errors++
				tc.Error = &jUnitError{Message: x.text(p.Error), Text: x.text(p.Error)}
			case !p.Passed():
				suite.Failures++
				msg := fmt.Sprintf("the server accepted an invalid request with %d", p.StatusCode)
				tc.Failure = &jUnitFailure{Message: msg, Text: msg}
			}
			suite.Cases = append(suite.Cases, tc)
		}
	}

	report := jUnitTestSuites{
		Name:       result.SpecTitle,
		Tests:      suite.Tests,
		Failures:   suite.Failures,
		Errors:     suite.Errors,
		Time:       suite.Time,
		TestSuites: []jUnitSuite{suite},
	}

	out, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// NegativeText returns a human-readable negative conformance report.
func NegativeText(result *conformance.NegativeResult) string {
	var b strings.Builder
	x := newRedactor(result.Secrets, false)

	fmt.Fprintf(&b, "\nNegative conformance probes — %s %s\n", result.SpecTitle, result.SpecVersion)

	for _, p := range result.Probes {
		switch {
		case p.Skipped:
			fmt.Fprintf(&b, "  SKIP  %s\n", p.Description)
		case p.Error != "":
			fmt.Fprintf(&b, "  ERR   %s\n         error: %s\n", p.Description, x.text(p.Error))
		case p.Passed():
			fmt.Fprintf(&b, "  PASS  %s  %d  %dms\n", p.Description, p.StatusCode, p.DurationMs)
		default:
			fmt.Fprintf(&b, "  FAIL  %s\n         server accepted invalid request with %d\n", p.Description, p.StatusCode)
		}
	}

	fmt.Fprintln(&b)
	passed, failed, skipped, errored := result.Counts()
	total := passed + failed + errored

	verdict := "PASSED"
	if failed > 0 {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "%s  %d probes  ·  %d passed  ·  %d failed  ·  %d skipped  ·  %d errors  %dms\n",
		verdict, total, passed, failed, skipped, errored, result.Duration.Milliseconds())
	if failed > 0 {
		fmt.Fprintf(&b, "       %d operation(s) accepted requests they should have rejected — fix server validation\n", failed)
	}

	return b.String()
}

// DiffJUnit returns a JUnit XML report for a spec diff result.
// Each change is emitted as a test case; breaking changes (ERR level) are
// marked as failures so PR annotation tools can surface them inline.
func DiffJUnit(result *openapi.DiffResult) ([]byte, error) {
	return DiffJUnitOn(result, openapi.FailOnBreaking)
}

// DiffJUnitOn is DiffJUnit with a chosen --fail-on threshold: every change at or above it is a
// failure.
func DiffJUnitOn(result *openapi.DiffResult, on openapi.FailOn) ([]byte, error) {
	breaking := result.FailCount(on)

	var cases []jUnitCase
	if len(result.Changes) == 0 {
		cases = append(cases, jUnitCase{
			Name:      "No breaking changes",
			ClassName: "openapi.diff",
			Time:      "0.000",
		})
	} else {
		for _, c := range result.Changes {
			name := fmt.Sprintf("[%s] %s", c.Level, c.ID)
			if c.Method != "" && c.Path != "" {
				name = fmt.Sprintf("[%s] %s  %s %s", c.Level, c.ID, c.Method, c.Path)
			}
			detail := c.Text
			if c.Method != "" && c.Path != "" {
				detail = fmt.Sprintf("%s  %s %s", c.Text, c.Method, c.Path)
			}
			tc := jUnitCase{
				Name:      name,
				ClassName: "openapi.diff",
				Time:      "0.000",
			}
			if c.Fails(on) {
				tc.Failure = &jUnitFailure{Message: detail, Text: detail}
			}
			cases = append(cases, tc)
		}
	}

	suite := jUnitSuite{
		Name:     "API Spec Diff",
		Tests:    len(cases),
		Failures: breaking,
		Time:     "0.000",
		Cases:    cases,
	}
	report := jUnitTestSuites{
		Name:       "API Spec Diff",
		Tests:      suite.Tests,
		Failures:   suite.Failures,
		Time:       suite.Time,
		TestSuites: []jUnitSuite{suite},
	}

	out, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// TextWithCoverage is like Text but appends a spec coverage section.
func TextWithCoverage(result *scenario.Result, cov *openapi.ScenarioCoverageResult) string {
	base := Text(result)
	if cov == nil {
		return base
	}
	return base + "\n" + CoverageText(cov, "")
}

func sortedCovOps(ops []openapi.CoveredOperation) []openapi.CoveredOperation {
	out := make([]openapi.CoveredOperation, len(ops))
	copy(out, ops)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// JSONWithCoverage is like JSON but includes spec coverage data.
func JSONWithCoverage(result *scenario.Result, cov *openapi.ScenarioCoverageResult) ([]byte, error) {
	b, err := JSON(result)
	if err != nil || cov == nil {
		return b, err
	}
	// Unmarshal → attach coverage → re-marshal.
	var report JSONReport
	if err := json.Unmarshal(b, &report); err != nil {
		return b, nil // shouldn't happen; return plain report
	}
	report.Coverage = jsonCoverage(cov)
	return json.MarshalIndent(report, "", "  ")
}

// JSON returns a machine-readable JSON report.
func JSON(result *scenario.Result) ([]byte, error) {
	x := redactorFor(result)
	report := JSONReport{
		Scenario:   result.Scenario.Title,
		Passed:     result.Passed,
		DurationMs: result.Duration.Milliseconds(),
	}

	if result.UsedRandomness {
		seed := result.Seed
		report.Seed = &seed
	}
	report.Error = x.text(result.SetupError)
	for _, v := range result.Variables {
		report.Variables = append(report.Variables, JSONVariable{Name: v.Name, Value: x.variable(v)})
	}
	for _, sr := range result.Steps {
		step := JSONStep{
			Name:      sr.Label,
			Passed:    sr.Passed,
			Skipped:   sr.Skipped,
			Error:     x.text(sr.Error),
			Extracted: x.extracted(sr.Extracted),
		}
		if sr.Response != nil {
			step.StatusCode = sr.Response.StatusCode
			step.DurationMs = sr.Response.Duration.Milliseconds()
		}
		if sr.Attempts > 1 {
			step.Attempts = sr.Attempts
		}
		for _, ar := range sr.Assertions {
			step.Assertions = append(step.Assertions, JSONAssertion{
				Type:     string(ar.Assertion.Type),
				Name:     ar.Assertion.Name,
				Path:     ar.Assertion.Path,
				Operator: string(ar.Assertion.Operator),
				Expected: fmt.Sprintf("%v", ar.Assertion.Expected),
				Actual:   x.text(ar.Actual),
				Passed:   ar.Passed,
				Message:  x.text(ar.Message),
			})
		}
		report.Steps = append(report.Steps, step)
	}

	return json.MarshalIndent(report, "", "  ")
}
