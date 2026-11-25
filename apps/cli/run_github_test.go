package main

import (
	"strings"
	"testing"
)

func failingScenario(url string) string {
	return `schemaVersion: "1"
title: Checkout
steps:
  - label: Health
    request: {method: GET, url: "` + url + `"}
    assertions:
      - {type: status, operator: eq, expected: 200}
  - label: Expect missing
    request: {method: GET, url: "` + url + `"}
    assertions:
      - {type: status, operator: eq, expected: 404}
      - {type: status, operator: eq, expected: 200}
`
}

func TestRunFormatGitHub_AnnotatesTheFailedStepOnItsLine(t *testing.T) {
	srv := pingServer(t)
	project(t, map[string]string{"scenarios/checkout.yaml": failingScenario(srv.URL), "scenarios/ok.yaml": scenarioYAML(srv.URL)})

	out, err := exec(t, "run", "scenarios/checkout.yaml", "--format", "github")
	if exitCodeOf(err) != 1 {
		t.Fatalf("the exit code is the same as for any other format: %v\n%s", err, out)
	}
	var errs []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasPrefix(l, "::") {
			t.Errorf("only workflow commands are printed, got %q", l)
		}
		if strings.HasPrefix(l, "::error ") {
			errs = append(errs, l)
		}
	}
	if len(errs) != 1 || !strings.HasPrefix(errs[0], "::error file=scenarios/checkout.yaml,line=8,title=Checkout%3A Expect missing::") || !strings.Contains(errs[0], "404") {
		t.Fatalf("one failed assertion, on the line of its step:\n%s", out)
	}
	if !strings.Contains(out, "::notice title=Clear Signal run::1 scenarios: 0 passed, 1 failed.") {
		t.Errorf("a summary notice:\n%s", out)
	}

	both, _ := exec(t, "run", "scenarios", "--format", "github")
	if !strings.Contains(both, "2 scenarios: 1 passed, 1 failed.") || strings.Count(both, "::error ") != 1 {
		t.Errorf("a folder run annotates only what failed:\n%s", both)
	}
}
