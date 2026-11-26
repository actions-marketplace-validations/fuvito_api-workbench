package assertions_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/assertions"
	"github.com/fuvito/api-workbench/packages/core/domain"
)

// The format guide says a schema may be written in place, as JSON or as a YAML block, and needs no
// specFile.
func TestASchemaWrittenInPlaceNeedsNoSpecFile(t *testing.T) {
	good := &domain.Response{StatusCode: 200, Body: []byte(`{"id":1}`)}
	bad := &domain.Response{StatusCode: 200, Body: []byte(`{"id":"one"}`)}
	for name, schema := range map[string]string{
		"JSON":                           `{"type":"object","required":["id"],"properties":{"id":{"type":"integer"}}}`,
		"YAML block":                     "type: object\nrequired: [id]\nproperties:\n  id: {type: integer}\n",
		"indented block, nested further": "  type: object\n  properties:\n    id:\n      type: integer\n  required:\n    - id\n",
	} {
		a := domain.Assertion{Type: domain.AssertionSchema, Schema: schema}
		if r := assertions.New().Evaluate(a, good); !r.Passed {
			t.Errorf("%s, a fitting body: %s", name, r.Message)
		}
		if r := assertions.New().Evaluate(a, bad); r.Passed {
			t.Errorf("%s, a body that does not fit, passed", name)
		}
	}
}
