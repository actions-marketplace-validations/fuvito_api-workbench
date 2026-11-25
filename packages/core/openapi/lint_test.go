package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const lintHealthy = `openapi: "3.0.3"
info: {title: ok, version: "1"}
paths:
  /users/{id}:
    parameters: [{name: id, in: path, required: true, schema: {type: integer}}]
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {$ref: "#/components/schemas/User"}
components:
  schemas:
    User:
      type: object
      required: [id]
      properties: {id: {type: integer}, name: {type: string}}
`

const lintBroken = `openapi: "3.0.3"
info: {title: broken, version: "1"}
paths:
  /users/{id}:
    parameters: [{name: id, in: path, required: true, schema: {type: integer}}]
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {$ref: "#/components/schemas/User"}
  /things:
    get:
      responses: {"200": {description: ok}}
components:
  schemas:
    User:
      type: object
      required: [id, email]
      properties:
        id: {type: integer}
        profile:
          type: object
          required: [nick]
          properties: {name: {type: string}}
`

func lintIDs(issues []openapi.SpecIssue) string {
	var out []string
	for _, i := range issues {
		out = append(out, i.Check)
	}
	return strings.Join(out, ",")
}

func TestLint_AHealthySpecHasNothingToSay(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(lintHealthy), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := spec.Lint(); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestLint_FindsWhatIsWrongWithTheDocument(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(lintBroken), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	issues := spec.Lint()
	ids := lintIDs(issues)
	for _, want := range []string{"CS-SPEC-001", "CS-SPEC-002"} {
		if !strings.Contains(ids, want) {
			t.Errorf("missing %s in %s", want, ids)
		}
	}
	byCheck := map[string][]openapi.SpecIssue{}
	for _, i := range issues {
		byCheck[i.Check] = append(byCheck[i.Check], i)
		if i.What == "" || i.Why == "" || i.Fix == "" {
			t.Errorf("every issue says what, why and how to fix it: %+v", i)
		}
	}
	if got := byCheck["CS-SPEC-001"]; len(got) != 1 || got[0].Operation != "GET /things" || got[0].Severity != openapi.SpecWarning {
		t.Errorf("a missing operationId is a warning on its operation: %+v", got)
	}
	required := byCheck["CS-SPEC-002"]
	if len(required) != 2 {
		t.Fatalf("both undeclared required names are found, nested too: %+v", required)
	}
	if !strings.Contains(required[0].What, `"email"`) && !strings.Contains(required[1].What, `"email"`) {
		t.Errorf("%+v", required)
	}
}

func TestLint_AnExampleThatBreaksItsFormatIsReported(t *testing.T) {
	doc := strings.Replace(lintHealthy, "name: {type: string}", "uid: {type: string, format: uuid, example: \"123\"}", 1)
	spec, err := openapi.ParseBytes([]byte(doc), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := spec.Lint(); len(got) != 1 || got[0].Check != "CS-SPEC-003" || got[0].Severity != openapi.SpecWarning {
		t.Errorf("%+v", got)
	}
}

func TestLint_ASchemaThatComposesIsNotJudged(t *testing.T) {
	doc := strings.Replace(lintHealthy, "required: [id]\n      properties", "required: [id, other]\n      allOf: [{type: object, properties: {other: {type: string}}}]\n      properties", 1)
	spec, err := openapi.ParseBytes([]byte(doc), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := spec.Lint(); len(got) != 0 {
		t.Errorf("a required name may come from an allOf member: %+v", got)
	}
}
