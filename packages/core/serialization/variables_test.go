package serialization_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/serialization"
)

const scenarioWithVariables = `schemaVersion: "1"
title: Orders
variables:
  - name: email
    value: "user-{{$randomString 8}}@example.com"
  - name: dueDate
    value: "{{$date +7d}}"
  - name: apiKey
    value: "${ORDERS_KEY}"
    secret: true
steps:
  - label: Create
    request:
      method: POST
      url: "http://x/orders"
      body:
        type: json
        content: '{"email":"{{email}}","due":"{{dueDate}}"}'
    assertions:
      - {type: jsonpath, path: $.email, operator: eq, expected: "{{email}}"}
`

func TestScenarioVariablesRoundTripInYAMLAndJSON(t *testing.T) {
	sc, err := serialization.ParseScenario([]byte(scenarioWithVariables), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.Variables) != 3 || sc.Variables[0].Name != "email" || sc.Variables[2].Value != "${ORDERS_KEY}" || !sc.Variables[2].Secret {
		t.Fatalf("variables = %+v", sc.Variables)
	}

	for _, format := range []string{"yaml", "json"} {
		data, err := serialization.WriteScenario(sc, format)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "variables") || !strings.Contains(string(data), "{{$randomString 8}}") {
			t.Errorf("%s output lost the variables:\n%s", format, data)
		}
		back, err := serialization.ParseScenario(data, format)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if len(back.Variables) != 3 || back.Variables[1].Value != "{{$date +7d}}" || !back.Variables[2].Secret {
			t.Errorf("%s round trip: %+v", format, back.Variables)
		}
		if back.Steps[0].Assertions[0].Expected != "{{email}}" {
			t.Errorf("%s: the assertion value must be kept", format)
		}
	}
}

func TestScenarioWithoutVariablesWritesNoVariablesKey(t *testing.T) {
	sc, err := serialization.ParseScenario([]byte("schemaVersion: \"1\"\ntitle: T\nsteps:\n  - {label: a, request: {method: GET, url: http://x}}\n"), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := serialization.WriteScenario(sc, "yaml")
	if strings.Contains(string(data), "variables") {
		t.Errorf("existing scenarios must stay unchanged on disk:\n%s", data)
	}
}

func TestBundleScenariosCarryVariables(t *testing.T) {
	bundle := `schemaVersion: "1"
title: B
scenarios:
  - title: One
    variables:
      - {name: id, value: "{{$guid}}"}
    steps:
      - {label: a, request: {method: GET, url: "http://x/{{id}}"}}
`
	b, err := serialization.ParseBundle([]byte(bundle), "yaml")
	if err != nil {
		t.Fatalf("a bundle scenario may declare variables: %v", err)
	}
	if len(b.Scenarios[0].Scenario.Variables) != 1 || b.Scenarios[0].Scenario.Variables[0].Value != "{{$guid}}" {
		t.Fatalf("variables = %+v", b.Scenarios[0].Scenario.Variables)
	}
	out, err := serialization.WriteBundle(b, "yaml")
	if err != nil || !strings.Contains(string(out), "{{$guid}}") {
		t.Errorf("writing a bundle keeps scenario variables: %v\n%s", err, out)
	}
}
