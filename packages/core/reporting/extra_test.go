package reporting_test

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestNegativeText(t *testing.T) {
	res := &conformance.NegativeResult{
		SpecTitle: "API", SpecVersion: "1", Duration: 120 * time.Millisecond,
		Probes: []conformance.NegativeProbeResult{
			{Description: "skipped probe", Skipped: true},
			{Description: "errored probe", Error: "connection refused"},
			{Description: "good probe", StatusCode: 401, DurationMs: 7},
			{Description: "bad probe", StatusCode: 200},
		},
	}
	out := reporting.NegativeText(res)
	for _, want := range []string{
		"Negative conformance probes — API 1",
		"SKIP  skipped probe",
		"ERR   errored probe", "error: connection refused",
		"PASS  good probe  401  7ms",
		"FAIL  bad probe", "server accepted invalid request with 200",
		"FAILED  3 probes  ·  1 passed  ·  1 failed  ·  1 skipped  ·  1 error  120ms",
		"1 probe: the server accepted a request it should have rejected",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestNegativeText_AllPassed(t *testing.T) {
	out := reporting.NegativeText(&conformance.NegativeResult{Probes: []conformance.NegativeProbeResult{{Description: "p", StatusCode: 400}}})
	if !strings.Contains(out, "PASSED  1 probe") || strings.Contains(out, "should have rejected") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestDiffJUnit(t *testing.T) {
	res := &openapi.DiffResult{
		HasBreaking: true,
		Changes: []openapi.DiffChange{
			{ID: "endpoint-removed", Text: "endpoint removed", Method: "GET", Path: "/a", Breaking: true, Level: "ERR"},
			{ID: "info-changed", Text: "title changed", Level: "INFO"},
		},
	}
	data, err := reporting.DiffJUnit(res)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("missing XML header")
	}
	var parsed struct {
		Tests    int `xml:"tests,attr"`
		Failures int `xml:"failures,attr"`
		Suites   []struct {
			Cases []struct {
				Name    string `xml:"name,attr"`
				Failure *struct {
					Message string `xml:"message,attr"`
				} `xml:"failure"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, data)
	}
	cases := parsed.Suites[0].Cases
	if parsed.Tests != 2 || parsed.Failures != 1 || len(cases) != 2 {
		t.Fatalf("counts: %+v", parsed)
	}
	if cases[0].Name != "[ERR] endpoint-removed  GET /a" || cases[0].Failure == nil || cases[0].Failure.Message != "endpoint removed  GET /a" {
		t.Errorf("breaking change should be a failure with location: %+v", cases[0])
	}
	if cases[1].Name != "[INFO] info-changed" || cases[1].Failure != nil {
		t.Errorf("non-breaking change must not fail: %+v", cases[1])
	}
}

func TestDiffJUnit_NoChanges(t *testing.T) {
	data, err := reporting.DiffJUnit(&openapi.DiffResult{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "No breaking changes") || strings.Contains(string(data), "<failure") {
		t.Errorf("unexpected report:\n%s", data)
	}
}

func TestText_AssertionLabelsForEveryType(t *testing.T) {
	mk := func(a domain.Assertion) string {
		res := &scenario.Result{
			Scenario: &domain.Scenario{Title: "t"}, Passed: true,
			Steps: []scenario.StepResult{{
				Label: "s", Passed: true, Request: &domain.Request{Method: domain.MethodGET, URL: "http://x"},
				Response:   &domain.Response{StatusCode: 200},
				Assertions: []domain.AssertionResult{{Assertion: a, Passed: true}},
			}},
		}
		return reporting.Text(res)
	}
	cases := []struct {
		a    domain.Assertion
		want string
	}{
		{domain.Assertion{Type: domain.AssertionStatus, Expected: 200}, "status eq 200"},
		{domain.Assertion{Type: domain.AssertionHeader, Path: "X-A", Operator: domain.OperatorContains, Expected: "b"}, "header X-A contains b"},
		{domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.id", Operator: domain.OperatorGt, Expected: 1}, "jsonpath $.id gt 1"},
		{domain.Assertion{Type: domain.AssertionSchema, Schema: "getUser"}, "schema"},
		{domain.Assertion{Type: "custom", Name: "spec conformance"}, "spec conformance"},
		{domain.Assertion{Type: "custom"}, "custom"},
	}
	for _, c := range cases {
		if out := mk(c.a); !strings.Contains(out, c.want) {
			t.Errorf("%+v: missing %q in\n%s", c.a, c.want, out)
		}
	}
}

// F-029: when every probe errored the headline says FAILED, as the exit code does.
func TestNegativeText_ErroredProbesAreNotPassed(t *testing.T) {
	out := reporting.NegativeText(&conformance.NegativeResult{Probes: []conformance.NegativeProbeResult{
		{Description: "p1", Error: "connection refused"}, {Description: "p2", Error: "connection refused"},
	}})
	if !strings.Contains(out, "FAILED  2 probes  ·  0 passed  ·  0 failed  ·  0 skipped  ·  2 errors") || strings.Contains(out, "PASSED") {
		t.Errorf("an all-errored run must not say PASSED:\n%s", out)
	}
	if !strings.Contains(out, "2 probe(s) got no answer") {
		t.Errorf("the headline says why:\n%s", out)
	}
}
