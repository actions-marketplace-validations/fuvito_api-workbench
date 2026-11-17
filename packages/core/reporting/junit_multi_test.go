package reporting_test

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestJUnitMulti(t *testing.T) {
	mk := func(title string, passed bool) *scenario.Result {
		return &scenario.Result{
			Scenario: &domain.Scenario{Title: title}, Passed: passed, Duration: 1500 * time.Millisecond,
			Steps: []scenario.StepResult{
				{Label: "ok", Passed: true},
				{Label: "bad", Passed: passed, Assertions: []domain.AssertionResult{{Assertion: domain.Assertion{Type: domain.AssertionStatus, Expected: 200}, Passed: passed, Message: "expected 200"}}},
			},
		}
	}
	data, err := reporting.JUnitMulti("Suite", []*scenario.Result{mk("First", true), mk("Second", false)})
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Name     string `xml:"name,attr"`
		Tests    int    `xml:"tests,attr"`
		Failures int    `xml:"failures,attr"`
		Time     string `xml:"time,attr"`
		Suites   []struct {
			Name string `xml:"name,attr"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, data)
	}
	if parsed.Name != "Suite" || parsed.Tests != 4 || parsed.Failures != 1 || len(parsed.Suites) != 2 || parsed.Suites[1].Name != "Second" {
		t.Errorf("combined report: %+v", parsed)
	}
	if parsed.Time != "3.000" {
		t.Errorf("total time: %q", parsed.Time)
	}
	if empty, err := reporting.JUnitMulti("None", nil); err != nil || !strings.Contains(string(empty), `tests="0"`) {
		t.Errorf("empty: %v %s", err, empty)
	}
}
