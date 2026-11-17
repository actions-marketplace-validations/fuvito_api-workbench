package reporting_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func op(m, p string) openapi.CoveredOperation { return openapi.CoveredOperation{Method: m, Path: p} }

func TestMergeCoverage_UnionOfExercised(t *testing.T) {
	a := &openapi.ScenarioCoverageResult{
		Total: 4, Covered: 1,
		Exercised: []openapi.CoveredOperation{op("GET", "/a")},
		Missed:    []openapi.CoveredOperation{op("GET", "/b"), op("POST", "/b"), op("GET", "/c")},
	}
	b := &openapi.ScenarioCoverageResult{
		Total: 4, Covered: 2,
		Exercised: []openapi.CoveredOperation{op("GET", "/b"), op("GET", "/a")},
		Missed:    []openapi.CoveredOperation{op("POST", "/b"), op("GET", "/c")},
	}
	m := reporting.MergeCoverage([]*openapi.ScenarioCoverageResult{a, nil, b})
	if m.Total != 4 || m.Covered != 2 || m.Percent != 50 {
		t.Errorf("merged = %+v", m)
	}
	var ex, miss []string
	for _, o := range m.Exercised {
		ex = append(ex, o.String())
	}
	for _, o := range m.Missed {
		miss = append(miss, o.String())
	}
	if strings.Join(ex, ",") != "GET /a,GET /b" || strings.Join(miss, ",") != "POST /b,GET /c" {
		t.Errorf("exercised=%v missed=%v", ex, miss)
	}
	if reporting.MergeCoverage(nil) != nil || reporting.MergeCoverage([]*openapi.ScenarioCoverageResult{nil}) != nil {
		t.Error("nothing to merge gives nil")
	}
}

func TestCoverageText_LabelOptional(t *testing.T) {
	plain := reporting.CoverageText(makeCovResult(), "")
	if !strings.HasPrefix(plain, "Spec coverage: 2 / 3 operations (67%)") {
		t.Errorf("plain heading: %q", plain)
	}
	labelled := reporting.CoverageText(makeCovResult(), "users.yaml")
	if !strings.HasPrefix(labelled, "Spec coverage (users.yaml): 2 / 3") {
		t.Errorf("labelled heading: %q", labelled)
	}
}

func TestTextMulti(t *testing.T) {
	results := []*scenario.Result{passedResult("One"), failedResult("Two")}
	cov := []reporting.SpecCoverage{{Spec: "api.yaml", Result: makeCovResult()}}
	out := reporting.TextMulti("scenarios", results, cov, 90*time.Millisecond)
	for _, want := range []string{
		"Run: scenarios  (2 scenarios)", "Scenario: One", "Scenario: Two",
		"Run FAILED  1/2 scenarios passed", "90ms", "Spec coverage: 2 / 3 operations",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	one := reporting.TextMulti("x", results[:1], nil, time.Millisecond)
	if !strings.Contains(one, "(1 scenario)") || !strings.Contains(one, "Run PASSED  1/1 scenarios passed") {
		t.Errorf("single-scenario wording:\n%s", one)
	}

	two := reporting.TextMulti("x", results, []reporting.SpecCoverage{
		{Spec: "a.yaml", Result: makeCovResult()}, {Spec: "b.yaml", Result: makeCovResult()},
	}, time.Millisecond)
	if !strings.Contains(two, "Spec coverage (a.yaml)") || !strings.Contains(two, "Spec coverage (b.yaml)") {
		t.Errorf("several specs are labelled:\n%s", two)
	}
}

func TestJSONMulti(t *testing.T) {
	results := []*scenario.Result{passedResult("One"), failedResult("Two")}
	b, err := reporting.JSONMulti("scenarios", results, []reporting.SpecCoverage{{Spec: "api.yaml", Result: makeCovResult()}}, 90*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Run        string `json:"run"`
		Passed     bool   `json:"passed"`
		DurationMs int64  `json:"durationMs"`
		Scenarios  []struct {
			Scenario string `json:"scenario"`
			Passed   bool   `json:"passed"`
		} `json:"scenarios"`
		Coverage []struct {
			Spec    string  `json:"spec"`
			Total   int     `json:"total"`
			Covered int     `json:"covered"`
			Percent float64 `json:"percent"`
		} `json:"coverage"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, b)
	}
	if doc.Run != "scenarios" || doc.Passed || doc.DurationMs != 90 || len(doc.Scenarios) != 2 {
		t.Errorf("doc = %+v", doc)
	}
	if doc.Scenarios[0].Scenario != "One" || !doc.Scenarios[0].Passed || doc.Scenarios[1].Passed {
		t.Errorf("scenarios = %+v", doc.Scenarios)
	}
	if len(doc.Coverage) != 1 || doc.Coverage[0].Spec != "api.yaml" || doc.Coverage[0].Total != 3 || doc.Coverage[0].Covered != 2 {
		t.Errorf("coverage = %+v", doc.Coverage)
	}

	b, _ = reporting.JSONMulti("x", results[:1], nil, 0)
	if strings.Contains(string(b), `"coverage"`) {
		t.Error("no coverage key without coverage")
	}
	if !strings.HasSuffix(string(b), "\n") {
		t.Error("report should end with a newline")
	}
}

func TestHTMLMulti_WithCoverage(t *testing.T) {
	results := []*scenario.Result{passedResult("One"), passedResult("Two")}
	b, err := reporting.HTMLMulti("scenarios", results, time.Millisecond,
		reporting.SpecCoverage{Spec: "a.yaml", Result: makeCovResult()},
		reporting.SpecCoverage{Spec: "b.yaml", Result: makeCovResult()})
	s := mustHTML(t, b, err)
	for _, want := range []string{"Spec coverage (a.yaml)", "Spec coverage (b.yaml)", "GET /widgets"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	b, err = reporting.HTMLMulti("scenarios", results, time.Millisecond, reporting.SpecCoverage{Spec: "a.yaml", Result: makeCovResult()})
	s = mustHTML(t, b, err)
	if !strings.Contains(s, "Spec coverage") || strings.Contains(s, "(a.yaml)") {
		t.Error("a single spec uses the plain heading")
	}
}
