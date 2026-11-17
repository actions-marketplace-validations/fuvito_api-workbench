package serialization_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/serialization"
)

const sampleBundle = `
schemaVersion: "1"
title: Suite
specFile: https://api.example.com/openapi.json
environments:
  - name: staging
    variables:
      - {name: baseUrl, value: "https://staging.example.com"}
      - {name: token, value: CHANGE_ME, secret: true}
requests:
  - file: check-health
    title: Check health
    method: GET
    url: "{{baseUrl}}/health"
    timeout: 5s
scenarios:
  - title: Smoke
    steps:
      - label: Health
        requestRef: check-health
        assertions:
          - {type: status, operator: eq, expected: 200}
`

func TestParseBundle_YAML(t *testing.T) {
	b, err := serialization.ParseBundle([]byte(sampleBundle), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "Suite" || b.SpecFile != "https://api.example.com/openapi.json" || b.SchemaVersion != "1" {
		t.Errorf("header: %+v", b)
	}
	if len(b.Environments) != 1 || b.Environments[0].Environment.Name != "staging" || !b.Environments[0].Environment.Variables[1].Secret {
		t.Errorf("environments: %+v", b.Environments)
	}
	if len(b.Requests) != 1 || b.Requests[0].File != "check-health" || b.Requests[0].Request.Timeout.Seconds() != 5 {
		t.Errorf("requests: %+v", b.Requests)
	}
	if len(b.Scenarios) != 1 || b.Scenarios[0].Scenario.Steps[0].RequestRef != "check-health" {
		t.Errorf("scenarios: %+v", b.Scenarios)
	}
}

func TestParseBundle_JSON(t *testing.T) {
	data := `{"schemaVersion":"1","scenarios":[{"title":"S","steps":[{"label":"a","request":{"method":"GET","url":"http://x"}}]}]}`
	b, err := serialization.ParseBundle([]byte(data), "json")
	if err != nil || len(b.Scenarios) != 1 || b.Scenarios[0].Scenario.Steps[0].Request.URL != "http://x" {
		t.Fatalf("%+v %v", b, err)
	}
}

func TestParseBundle_RejectsUnknownFields(t *testing.T) {
	cases := map[string]string{
		"top level":       "schemaVersion: \"1\"\nscenarioz: []\n",
		"request timeout": "requests:\n  - file: a\n    method: GET\n    url: http://x\n    timeoutMs: 5000\n",
		"step field":      "scenarios:\n  - title: S\n    steps:\n      - label: a\n        retries: 2\n",
		"assertion field": "scenarios:\n  - title: S\n    steps:\n      - label: a\n        assertions:\n          - {type: status, expectd: 200}\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := serialization.ParseBundle([]byte(doc), "yaml")
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("expected an unknown-field error, got %v", err)
			}
			if strings.Contains(err.Error(), "serialization.") {
				t.Errorf("Go type names should not leak into messages: %v", err)
			}
		})
	}
	if _, err := serialization.ParseBundle([]byte(`{"nope":1}`), "json"); err == nil {
		t.Error("json should be strict too")
	}
}

func TestParseBundle_Errors(t *testing.T) {
	if _, err := serialization.ParseBundle([]byte(""), "yaml"); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty: %v", err)
	}
	if _, err := serialization.ParseBundle([]byte("a: ["), "yaml"); err == nil {
		t.Error("bad yaml")
	}
	if _, err := serialization.ParseBundle([]byte("{"), "json"); err == nil {
		t.Error("bad json")
	}
	if _, err := serialization.ParseBundle([]byte("{}"), "toml"); err == nil {
		t.Error("unknown format")
	}
	_, err := serialization.ParseBundle([]byte("requests:\n  - file: a\n    method: GET\n"), "yaml")
	if err == nil || !strings.Contains(err.Error(), `requests[0] "a"`) || !strings.Contains(err.Error(), "url is required") {
		t.Errorf("request errors should be located: %v", err)
	}
	_, err = serialization.ParseBundle([]byte("requests:\n  - file: a\n    method: GET\n    url: http://x\n    timeout: soon\n"), "yaml")
	if err == nil || !strings.Contains(err.Error(), "invalid timeout") {
		t.Errorf("bad timeout: %v", err)
	}
	_, err = serialization.ParseBundle([]byte("scenarios:\n  - title: S\n    steps:\n      - label: a\n        request: {method: GET}\n"), "yaml")
	if err == nil || !strings.Contains(err.Error(), `scenarios[0] "S"`) {
		t.Errorf("scenario errors should be located: %v", err)
	}
}
