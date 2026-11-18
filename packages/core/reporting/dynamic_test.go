package reporting_test

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func dynamicResult() *scenario.Result {
	r := passedResult("Orders")
	r.Seed = 8123
	r.UsedRandomness = true
	r.Variables = []scenario.ResolvedVariable{
		{Name: "email", Value: "u-abc@example.com"},
		{Name: "apiToken", Value: "tok-supersecret-1", Secret: true},
	}
	r.Secrets = []string{"tok-supersecret-1"}
	return r
}

func TestDynamicRunShowsSeedAndVariablesInEveryFormat(t *testing.T) {
	r := dynamicResult()

	text := reporting.Text(r)
	for _, want := range []string{"seed: 8123", "var email = u-abc@example.com", "var apiToken = ***"} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q:\n%s", want, text)
		}
	}

	b, err := reporting.JSON(r)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Seed      *int64 `json:"seed"`
		Variables []struct{ Name, Value string }
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.Seed == nil || *doc.Seed != 8123 || len(doc.Variables) != 2 || doc.Variables[1].Value != "***" {
		t.Errorf("json = %+v (%v)\n%s", doc, err, b)
	}

	j, err := reporting.JUnit(r)
	if err != nil {
		t.Fatal(err)
	}
	var ju struct {
		Suites []struct {
			Props []struct {
				Name  string `xml:"name,attr"`
				Value string `xml:"value,attr"`
			} `xml:"properties>property"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(j, &ju); err != nil || len(ju.Suites) != 1 || len(ju.Suites[0].Props) != 1 || ju.Suites[0].Props[0].Name != "seed" || ju.Suites[0].Props[0].Value != "8123" {
		t.Errorf("junit seed property = %+v (%v)\n%s", ju, err, j)
	}

	h, err := reporting.HTML(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	md := string(reporting.Markdown(r, nil))
	for name, out := range map[string]string{"html": string(h), "markdown": md} {
		for _, want := range []string{"seed 8123", "variable email", "u-abc@example.com"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s missing %q", name, want)
			}
		}
		if strings.Contains(out, "tok-supersecret-1") {
			t.Errorf("%s leaked a secret scenario variable", name)
		}
	}
}

func TestRunWithoutRandomnessPrintsNoSeed(t *testing.T) {
	r := passedResult("Static")
	r.Seed = 99 // set, but nothing was random
	for name, out := range map[string]string{
		"text":     reporting.Text(r),
		"markdown": string(reporting.Markdown(r, nil)),
	} {
		if strings.Contains(strings.ToLower(out), "seed") {
			t.Errorf("%s: a run that drew no random values needs no seed:\n%s", name, out)
		}
	}
	b, _ := reporting.JSON(r)
	if strings.Contains(string(b), "seed") || strings.Contains(string(b), "variables") {
		t.Errorf("json should be unchanged for a static run:\n%s", b)
	}
}

func TestSetupErrorIsReportedEverywhere(t *testing.T) {
	r := failedResult("Broken")
	r.SetupError = "variable x: {{$randomIntt 1 2}}: unknown function $randomIntt"
	r.Passed = false

	if text := reporting.Text(r); !strings.Contains(text, "error: variable x") {
		t.Errorf("text:\n%s", text)
	}
	b, _ := reporting.JSON(r)
	if !strings.Contains(string(b), `"error": "variable x`) {
		t.Errorf("json:\n%s", b)
	}
	j, _ := reporting.JUnit(r)
	var ju struct {
		Errors int `xml:"errors,attr"`
		Suites []struct {
			Cases []struct {
				Name  string `xml:"name,attr"`
				Error *struct {
					Message string `xml:"message,attr"`
				} `xml:"error"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(j, &ju); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range ju.Suites[0].Cases {
		if c.Name == "scenario variables" && c.Error != nil && strings.Contains(c.Error.Message, "unknown function") {
			found = true
		}
	}
	if !found {
		t.Errorf("junit should carry the setup error as an errored test case:\n%s", j)
	}
	h, _ := reporting.HTML(r, nil)
	md := string(reporting.Markdown(r, nil))
	if !strings.Contains(string(h), "scenario could not start") || !strings.Contains(md, "scenario could not start") {
		t.Error("html and markdown should say the scenario could not start")
	}
}

func TestCombinedReportsShowOneSeed(t *testing.T) {
	a, b := dynamicResult(), dynamicResult()
	b.Scenario = makeScenario("Second")
	out := reporting.TextMulti("suite", []*scenario.Result{a, b}, nil, time.Millisecond)
	if !strings.Contains(out, "Run: suite  (2 scenarios)  seed 8123") || strings.Contains(out, "          seed: 8123") {
		t.Errorf("the seed belongs to the run and is printed once in the header:\n%s", out)
	}
	j, err := reporting.JSONMulti("suite", []*scenario.Result{a, b}, nil, time.Millisecond)
	if err != nil || !strings.Contains(string(j), `"seed": 8123`) {
		t.Errorf("json: %v\n%s", err, j)
	}
	md := string(reporting.MarkdownMulti("suite", []*scenario.Result{a, b}, time.Millisecond))
	if !strings.Contains(md, "seed 8123") {
		t.Errorf("markdown summary:\n%s", md)
	}

	static, _ := reporting.JSONMulti("suite", []*scenario.Result{passedResult("A")}, nil, time.Millisecond)
	if strings.Contains(string(static), "seed") {
		t.Error("a run without random values has no seed")
	}
}

func TestRetriedStepShowsItsAttempts(t *testing.T) {
	r := passedResult("Flaky")
	r.Steps[0].Attempts = 3
	if !strings.Contains(reporting.Text(r), "(3 attempts)") {
		t.Errorf("text:\n%s", reporting.Text(r))
	}
	b, _ := reporting.JSON(r)
	if !strings.Contains(string(b), `"attempts": 3`) {
		t.Errorf("json:\n%s", b)
	}
	h, _ := reporting.HTML(r, nil)
	if !strings.Contains(string(h), "3 attempts") || !strings.Contains(string(reporting.Markdown(r, nil)), "3 attempts") {
		t.Error("html and markdown show the attempts")
	}
	once := passedResult("Once")
	once.Steps[0].Attempts = 1
	b, _ = reporting.JSON(once)
	if strings.Contains(string(b), "attempts") || strings.Contains(reporting.Text(once), "attempts") {
		t.Error("a step sent once shows no attempt count")
	}
}
