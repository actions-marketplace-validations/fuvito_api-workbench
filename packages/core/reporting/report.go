package reporting

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// Text returns a human-readable scenario result suitable for terminal output.
// Uses plain ASCII (no ANSI codes) so it is safe in CI logs.
func Text(result *scenario.Result) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Scenario: %s\n", result.Scenario.Title)
	if result.Scenario.Description != "" {
		fmt.Fprintf(&b, "          %s\n", result.Scenario.Description)
	}
	fmt.Fprintln(&b)

	for _, sr := range result.Steps {
		if sr.Skipped {
			fmt.Fprintf(&b, "  SKIP  %s\n", sr.Title)
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

		fmt.Fprintf(&b, "  %s  %s%s\n", status, sr.Title, suffix)

		if sr.Error != "" {
			fmt.Fprintf(&b, "         error: %s\n", sr.Error)
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
				fmt.Fprintf(&b, "         %s %s: %s\n", mark, label, ar.Message)
			}
		}

		for name, val := range sr.Extracted {
			fmt.Fprintf(&b, "         → %s = %q\n", name, val)
		}
	}

	fmt.Fprintln(&b)
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
	default:
		if a.Name != "" {
			return a.Name
		}
		return string(a.Type)
	}
}

// JSONReport is the machine-readable output structure for CI/CD pipelines.
type JSONReport struct {
	Scenario   string     `json:"scenario"`
	Passed     bool       `json:"passed"`
	DurationMs int64      `json:"durationMs"`
	Steps      []JSONStep `json:"steps"`
}

type JSONStep struct {
	Name       string            `json:"name"`
	Passed     bool              `json:"passed"`
	Skipped    bool              `json:"skipped,omitempty"`
	Error      string            `json:"error,omitempty"`
	StatusCode int               `json:"statusCode,omitempty"`
	DurationMs int64             `json:"durationMs,omitempty"`
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
	XMLName    xml.Name       `xml:"testsuites"`
	Name       string         `xml:"name,attr"`
	Tests      int            `xml:"tests,attr"`
	Failures   int            `xml:"failures,attr"`
	Errors     int            `xml:"errors,attr"`
	Time       string         `xml:"time,attr"`
	TestSuites []jUnitSuite   `xml:"testsuite"`
}

type jUnitSuite struct {
	Name     string        `xml:"name,attr"`
	Tests    int           `xml:"tests,attr"`
	Failures int           `xml:"failures,attr"`
	Errors   int           `xml:"errors,attr"`
	Time     string        `xml:"time,attr"`
	Cases    []jUnitCase   `xml:"testcase"`
}

type jUnitCase struct {
	Name      string         `xml:"name,attr"`
	ClassName string         `xml:"classname,attr"`
	Time      string         `xml:"time,attr"`
	Skipped   *struct{}      `xml:"skipped"`
	Failure   *jUnitFailure  `xml:"failure"`
	Error     *jUnitError    `xml:"error"`
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
	_, failed, _ := result.Counts()

	suite := jUnitSuite{
		Name:     result.Scenario.Title,
		Tests:    len(result.Steps),
		Failures: failed,
		Time:     secStr(result.Duration.Milliseconds()),
	}

	for _, sr := range result.Steps {
		stepMs := int64(0)
		if sr.Response != nil {
			stepMs = sr.Response.Duration.Milliseconds()
		}
		tc := jUnitCase{
			Name:      sr.Title,
			ClassName: result.Scenario.Title,
			Time:      secStr(stepMs),
		}
		switch {
		case sr.Skipped:
			skipped := struct{}{}
			tc.Skipped = &skipped
		case sr.Error != "":
			suite.Errors++
			tc.Error = &jUnitError{Message: sr.Error, Text: sr.Error}
		case !sr.Passed:
			var lines []string
			for _, ar := range sr.Assertions {
				if !ar.Passed {
					lines = append(lines, fmt.Sprintf("  ✗ %s: %s", assertionLabel(ar.Assertion), ar.Message))
				}
			}
			tc.Failure = &jUnitFailure{
				Message: fmt.Sprintf("%d assertion(s) failed", len(lines)),
				Text:    strings.Join(lines, "\n"),
			}
		}
		suite.Cases = append(suite.Cases, tc)
	}

	report := jUnitTestSuites{
		Name:       result.Scenario.Title,
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

// ---- Conformance reporting ----

// ConformText returns a human-readable conformance report suitable for terminal / CI output.
// Pass a non-nil coverage to append a coverage summary section.
func ConformText(result *conformance.ConformResult, coverage *conformance.CoverageReport) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s  %s\n", result.SpecTitle, result.SpecVersion)
	fmt.Fprintf(&b, "Base URL: %s\n\n", result.BaseURL)

	for _, op := range result.Operations {
		if op.Skipped {
			fmt.Fprintf(&b, "  SKIP  %-7s %s\n", op.Method, op.Path)
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
		fmt.Fprintf(&b, "  %s  %-7s %s%s\n", status, op.Method, op.Path, suffix)
		if op.Error != "" {
			fmt.Fprintf(&b, "         error: %s\n", op.Error)
		}
		for _, v := range op.Violations {
			fmt.Fprintf(&b, "         ✗ %s\n", v.Error())
		}
	}

	fmt.Fprintln(&b)
	passed, failed, skipped := result.Counts()
	total := passed + failed + skipped

	verdict := "PASSED"
	if failed > 0 {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "%s  %d operations  ·  %d passed  ·  %d failed  ·  %d skipped  %dms\n",
		verdict, total, passed, failed, skipped, result.Duration.Milliseconds())

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
	SpecTitle   string           `json:"specTitle"`
	SpecVersion string           `json:"specVersion"`
	BaseURL     string           `json:"baseURL"`
	Passed      bool             `json:"passed"`
	DurationMs  int64            `json:"durationMs"`
	Operations  []JSONConformOp  `json:"operations"`
	Coverage    *JSONCoverage    `json:"coverage,omitempty"`
}

type JSONConformOp struct {
	OperationID string   `json:"operationId,omitempty"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	StatusCode  int      `json:"statusCode,omitempty"`
	DurationMs  int64    `json:"durationMs,omitempty"`
	Passed      bool     `json:"passed"`
	Skipped     bool     `json:"skipped,omitempty"`
	Error       string   `json:"error,omitempty"`
	Violations  []string `json:"violations,omitempty"`
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
// Pass a non-nil coverage to include a coverage summary in the output.
func ConformJSON(result *conformance.ConformResult, coverage *conformance.CoverageReport) ([]byte, error) {
	_, failed, _ := result.Counts()
	report := JSONConformReport{
		SpecTitle:   result.SpecTitle,
		SpecVersion: result.SpecVersion,
		BaseURL:     result.BaseURL,
		Passed:      failed == 0,
		DurationMs:  result.Duration.Milliseconds(),
	}
	for _, op := range result.Operations {
		jo := JSONConformOp{
			OperationID: op.OperationID,
			Method:      op.Method,
			Path:        op.Path,
			StatusCode:  op.StatusCode,
			DurationMs:  op.DurationMs,
			Passed:      op.Passed(),
			Skipped:     op.Skipped,
			Error:       op.Error,
		}
		for _, v := range op.Violations {
			jo.Violations = append(jo.Violations, v.Error())
		}
		report.Operations = append(report.Operations, jo)
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

// ConformJUnit returns a JUnit XML conformance report.
func ConformJUnit(result *conformance.ConformResult) ([]byte, error) {
	_, failed, _ := result.Counts()

	suite := jUnitSuite{
		Name:     result.SpecTitle,
		Tests:    len(result.Operations),
		Failures: failed,
		Time:     secStr(result.Duration.Milliseconds()),
	}

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
			tc.Error = &jUnitError{Message: op.Error, Text: op.Error}
		case !op.Passed():
			msgs := make([]string, len(op.Violations))
			for i, v := range op.Violations {
				msgs[i] = "  ✗ " + v.Error()
			}
			tc.Failure = &jUnitFailure{
				Message: fmt.Sprintf("%d violation(s)", len(op.Violations)),
				Text:    strings.Join(msgs, "\n"),
			}
		}
		suite.Cases = append(suite.Cases, tc)
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

// JSON returns a machine-readable JSON report.
func JSON(result *scenario.Result) ([]byte, error) {
	report := JSONReport{
		Scenario:   result.Scenario.Title,
		Passed:     result.Passed,
		DurationMs: result.Duration.Milliseconds(),
	}

	for _, sr := range result.Steps {
		step := JSONStep{
			Name:      sr.Title,
			Passed:    sr.Passed,
			Skipped:   sr.Skipped,
			Error:     sr.Error,
			Extracted: sr.Extracted,
		}
		if sr.Response != nil {
			step.StatusCode = sr.Response.StatusCode
			step.DurationMs = sr.Response.Duration.Milliseconds()
		}
		for _, ar := range sr.Assertions {
			step.Assertions = append(step.Assertions, JSONAssertion{
				Type:     string(ar.Assertion.Type),
				Name:     ar.Assertion.Name,
				Path:     ar.Assertion.Path,
				Operator: string(ar.Assertion.Operator),
				Expected: fmt.Sprintf("%v", ar.Assertion.Expected),
				Actual:   ar.Actual,
				Passed:   ar.Passed,
				Message:  ar.Message,
			})
		}
		report.Steps = append(report.Steps, step)
	}

	return json.MarshalIndent(report, "", "  ")
}
