package bundle_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/bundle"
)

func validateText(t *testing.T, text string) []bundle.Issue {
	t.Helper()
	b, err := bundle.ParseText(text)
	if err != nil {
		t.Fatal(err)
	}
	return bundle.Validate(b, bundle.Options{})
}

const dynamicBundle = `schemaVersion: "1"
title: Dyn
environments:
  - name: local
    variables:
      - {name: base, value: "http://x"}
scenarios:
  - title: One
    variables:
      - {name: id, value: "id-{{$randomString 8}}"}
    steps:
      - label: a
        request: {method: GET, url: "{{base}}/items/{{id}}?n={{$randomInt 1 9}}&t={{$timestamp}}"}
`

func TestScenarioVariablesAndFunctionsAreNotReportedAsUndefined(t *testing.T) {
	for _, iss := range validateText(t, dynamicBundle) {
		t.Errorf("a valid dynamic bundle should have no findings, got %s: %s", iss.Severity, iss.Message)
	}
}

func TestUndefinedVariablesAreStillReported(t *testing.T) {
	text := strings.Replace(dynamicBundle, "{{id}}", "{{nope}}", 1)
	found := false
	for _, iss := range validateText(t, text) {
		if strings.Contains(iss.Message, "{{nope}} is not defined") {
			found = true
		}
	}
	if !found {
		t.Error("a genuinely undefined variable must still be reported")
	}
}

func TestDynamicMistakesAreErrorsInBundles(t *testing.T) {
	text := strings.Replace(dynamicBundle, "{{$randomString 8}}", "{{$randomStringg 8}}", 1)
	found := false
	for _, iss := range validateText(t, text) {
		if iss.Severity == bundle.SeverityError && strings.Contains(iss.Message, "unknown function $randomStringg") {
			found = true
		}
	}
	if !found {
		t.Error("an unknown function in a bundle blocks the import")
	}
}
