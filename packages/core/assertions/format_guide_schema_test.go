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

// Pins CS-GAP-020: a schema assertion does not look at the body of a status it does not know.
func TestSchemaAssertion_AnotherStatusPassesUnchecked_KnownGap(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionSchema, Schema: `{"type":"object","required":["neverThere"]}`}
	for _, status := range []int{201, 404, 500} {
		r := assertions.New().Evaluate(a, &domain.Response{StatusCode: status, Body: []byte(`{}`)})
		if !r.Passed {
			t.Errorf("%d is now checked (%s): update CS-GAP-020, the format guide, and close B-135", status, r.Message)
		}
	}
}
