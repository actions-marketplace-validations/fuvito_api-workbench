package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const weakScenario = `schemaVersion: "1"
title: Weak
steps:
  - label: Ping
    request: {method: GET, url: "{{baseUrl}}/ping/{{missing}}"}
  - label: Login
    request: {method: POST, url: "{{baseUrl}}/login"}
    extractions:
      - {name: token, source: jsonpath, path: $.token}
`

const goodScenario = `schemaVersion: "1"
title: Good
steps:
  - label: Login
    request: {method: POST, url: "{{baseUrl}}/login"}
    extractions:
      - {name: token, source: jsonpath, path: $.token}
  - label: Me
    request:
      method: GET
      url: "{{baseUrl}}/me"
      headers:
        - {name: Authorization, value: "Bearer {{token}}"}
    assertions:
      - {type: status, operator: eq, expected: 200}
`

const strictEnv = "schemaVersion: \"1\"\nname: ci\nvariables:\n  - {name: baseUrl, value: http://x}\n"

func TestValidateStrict_WarnsAboutWeakScenarios(t *testing.T) {
	project(t, map[string]string{"scenarios/weak.yaml": weakScenario, "scenarios/good.yaml": goodScenario, "environments/ci.yaml": strictEnv})

	out, err := exec(t, "validate", "scenarios/weak.yaml")
	if err != nil || !strings.Contains(out, "OK") || strings.Contains(out, "warning") {
		t.Fatalf("without --strict nothing changes: %v\n%s", err, out)
	}

	out, err = exec(t, "validate", "--strict", "scenarios/weak.yaml")
	if exitCodeOf(err) != 1 {
		t.Fatalf("--strict fails on warnings: %v\n%s", err, out)
	}
	for _, want := range []string{
		"warning: [step 1] Ping: has no assertions and extracts nothing",
		`warning: [step 2] Login: extracts "token", but no later step uses {{token}}`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "{{missing}}") {
		t.Errorf("names nothing defines need --env:\n%s", out)
	}

	out, err = exec(t, "validate", "--strict", "--env", "environments/ci.yaml", "scenarios/weak.yaml")
	if exitCodeOf(err) != 1 || !strings.Contains(out, "uses {{missing}}, which nothing defines") {
		t.Fatalf("with --env the undefined name is found: %v\n%s", err, out)
	}

	if out, err := exec(t, "validate", "--strict", "--env", "environments/ci.yaml", "scenarios/good.yaml"); err != nil || !strings.Contains(out, "OK") {
		t.Errorf("a good scenario passes --strict: %v\n%s", err, out)
	}
}

func TestValidateStrict_JSONAndFolders(t *testing.T) {
	project(t, map[string]string{"scenarios/weak.yaml": weakScenario, "scenarios/good.yaml": goodScenario})
	out, err := exec(t, "validate", "--strict", "--format", "json", "scenarios")
	if exitCodeOf(err) != 1 {
		t.Fatalf("%v\n%s", err, out)
	}
	var res []struct {
		Path     string `json:"path"`
		Valid    bool   `json:"valid"`
		Warnings []struct {
			Step    int    `json:"step"`
			Message string `json:"message"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	byName := map[string]int{}
	for _, r := range res {
		byName[strings.TrimSuffix(r.Path[strings.LastIndexAny(r.Path, `/\`)+1:], ".yaml")] = len(r.Warnings)
		if r.Valid != (len(r.Warnings) == 0) {
			t.Errorf("%s: valid=%v with %d warning(s): with --strict a warning makes the file not valid", r.Path, r.Valid, len(r.Warnings))
		}
	}
	if byName["weak"] != 2 || byName["good"] != 0 {
		t.Errorf("warnings per file: %v", byName)
	}

	text, _ := exec(t, "validate", "--strict", "scenarios")
	if !strings.Contains(text, "2 file(s) checked  1 valid  1 invalid") {
		t.Errorf("the summary counts a file with warnings as not valid:\n%s", text)
	}
}

func TestValidateStrict_UsesTheProjectFile(t *testing.T) {
	project(t, map[string]string{
		"scenarios/weak.yaml":  weakScenario,
		"environments/ci.yaml": strictEnv,
		"clrsig.yaml":          "schemaVersion: \"1\"\nenvironment: environments/ci.yaml\nrun:\n  scenarios: [scenarios]\n",
	})
	out, err := exec(t, "validate", "--strict")
	if exitCodeOf(err) != 1 || !strings.Contains(out, "uses {{missing}}") || !strings.Contains(out, "Using clrsig.yaml") {
		t.Fatalf("the scenarios and the environment come from the file: %v\n%s", err, out)
	}
}
