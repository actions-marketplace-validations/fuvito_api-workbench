package scenario_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestStepLines(t *testing.T) {
	text := "schemaVersion: \"1\"\ntitle: T\nsteps:\n  - label: One\n    request: {method: GET, url: x}\n  - label: Two\n    request: {method: GET, url: y}\n"
	got := scenario.StepLines([]byte(text))
	if len(got) != 2 || got[0] != 4 || got[1] != 6 {
		t.Errorf("%v", got)
	}
	if got := scenario.StepLines([]byte("{\n  \"steps\": [\n    {\"label\": \"a\"}\n  ]\n}\n")); len(got) != 1 || got[0] != 3 {
		t.Errorf("json: %v", got)
	}
	if scenario.StepLines([]byte("not: [valid")) != nil || scenario.StepLines([]byte("title: x\n")) != nil {
		t.Error("an unreadable file or one without steps has no lines")
	}
}
