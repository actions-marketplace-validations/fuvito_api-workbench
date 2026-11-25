package main

import (
	"strings"
	"testing"
)

const lintSpecClean = `openapi: "3.0.3"
info: {title: clean, version: "1"}
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
`

const lintSpecNoID = `openapi: "3.0.3"
info: {title: noid, version: "1"}
paths:
  /a:
    get:
      responses: {"200": {description: ok}}
`

const lintSpecBroken = `openapi: "3.0.3"
info: {title: broken, version: "1"}
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id, email], properties: {id: {type: integer}}}
`

func TestValidateSpec_ReportsProblemsInTheDocument(t *testing.T) {
	project(t, map[string]string{"clean.yaml": lintSpecClean, "noid.yaml": lintSpecNoID, "broken.yaml": lintSpecBroken, "s.yaml": goodScenario})

	if out, err := exec(t, "validate", "--spec", "clean.yaml"); err != nil || !strings.Contains(out, "clean.yaml  OK") {
		t.Fatalf("a clean spec passes, with no scenario given: %v\n%s", err, out)
	}

	out, err := exec(t, "validate", "--spec", "broken.yaml")
	if exitCodeOf(err) != 1 || !strings.Contains(out, "[CS-SPEC-002]") || !strings.Contains(out, `"email"`) || !strings.Contains(out, "Fix:") {
		t.Fatalf("an undeclared required property is an error with its check, why and fix: %v\n%s", err, out)
	}

	out, err = exec(t, "validate", "--spec", "noid.yaml")
	if err != nil || !strings.Contains(out, "note: [CS-SPEC-001]") {
		t.Fatalf("a missing operationId is a note and does not fail: %v\n%s", err, out)
	}
	if out, err := exec(t, "validate", "--spec", "noid.yaml", "--strict"); exitCodeOf(err) != 1 || !strings.Contains(out, "warning: [CS-SPEC-001]") {
		t.Fatalf("--strict makes it a failing warning: %v\n%s", err, out)
	}
}

func TestValidateSpec_WorksWithScenariosJSONAndAProjectFile(t *testing.T) {
	project(t, map[string]string{
		"noid.yaml":   lintSpecNoID,
		"s.yaml":      goodScenario,
		"clrsig.yaml": "schemaVersion: \"1\"\n",
	})
	out, err := exec(t, "validate", "--spec", "noid.yaml", "s.yaml", "--format", "json")
	if err != nil || !strings.Contains(out, `"notes"`) || !strings.Contains(out, "CS-SPEC-001") || !strings.Contains(out, `"valid": true`) {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := exec(t, "validate", "--spec", "missing.yaml"); exitCodeOf(err) != 1 {
		t.Errorf("a spec that cannot be read fails: %v", err)
	}
	if _, err := exec(t, "validate"); err == nil || !strings.Contains(err.Error(), "--spec") {
		t.Errorf("with nothing to check the error mentions --spec: %v", err)
	}
}
