package assertions_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/assertions"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const specYAML = `
openapi: "3.0.3"
info:
  title: t
  version: "1"
paths:
  /users/1:
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id:
                    type: integer
`

func jsonResp(body string) *domain.Response {
	return &domain.Response{
		StatusCode: 200,
		Headers:    []domain.Header{{Name: "Content-Type", Value: "application/json"}},
		Body:       []byte(body),
	}
}

func schemaAssertion(s string) domain.Assertion {
	return domain.Assertion{Type: domain.AssertionSchema, Schema: s}
}

func TestSchema_Validation(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(specYAML), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	withSpec := assertions.NewWithSpec(spec)

	tests := []struct {
		name     string
		ev       *assertions.Evaluator
		schema   string
		resp     *domain.Response
		wantPass bool
		wantMsg  string
	}{
		{"missing schema", assertions.New(), "", jsonResp(`{}`), false, "requires a schema"},
		{"empty body", assertions.New(), "getUser", jsonResp(""), false, "body is empty"},
		{"operationId without spec", assertions.New(), "getUser", jsonResp(`{"id":1}`), false, "no specFile"},
		{"operationId pass", withSpec, "getUser", jsonResp(`{"id":1}`), true, ""},
		{"operationId fail", withSpec, "getUser", jsonResp(`{"name":"x"}`), false, ""},
		{"inline JSON pass", assertions.New(), `{"type":"object","required":["id"]}`, jsonResp(`{"id":1}`), true, ""},
		{"inline JSON fail", assertions.New(), `{"type":"object","required":["id"]}`, jsonResp(`{}`), false, ""},
		{"inline YAML pass", assertions.New(), "type: object\nrequired:\n  - id\n", jsonResp(`{"id":1}`), true, ""},
		{"inline YAML fail", assertions.New(), "type: object\nrequired:\n  - id\n", jsonResp(`{"x":1}`), false, ""},
		{"invalid inline schema", assertions.New(), `{"type": `, jsonResp(`{}`), false, "invalid schema"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.ev.Evaluate(schemaAssertion(tc.schema), tc.resp)
			if res.Passed != tc.wantPass {
				t.Fatalf("passed=%v want %v; msg=%q", res.Passed, tc.wantPass, res.Message)
			}
			if !tc.wantPass && res.Message == "" {
				t.Error("failing result should carry a message")
			}
			if tc.wantMsg != "" && !strings.Contains(res.Message, tc.wantMsg) {
				t.Errorf("message %q should contain %q", res.Message, tc.wantMsg)
			}
		})
	}
}

func TestEvaluate_UnknownType(t *testing.T) {
	res := eval.Evaluate(domain.Assertion{Type: "bogus"}, resp(200, nil, ""))
	if res.Passed || !strings.Contains(res.Message, "unknown assertion type") {
		t.Errorf("unexpected result %+v", res)
	}
}

func TestCompare_Operators(t *testing.T) {
	tests := []struct {
		name     string
		op       domain.Operator
		expected any
		status   int
		wantPass bool
		wantMsg  string
	}{
		{"ne pass", domain.OperatorNe, "404", 200, true, ""},
		{"ne fail", domain.OperatorNe, "200", 200, false, "!="},
		{"contains fail", domain.OperatorContains, "99", 200, false, "contain"},
		{"matches pass", domain.OperatorMatches, `^2\d\d$`, 200, true, ""},
		{"matches fail", domain.OperatorMatches, `^4`, 200, false, "match"},
		{"matches bad regex", domain.OperatorMatches, `(`, 200, false, "invalid regex"},
		{"gt fail", domain.OperatorGt, "300", 200, false, ">"},
		{"gt non-numeric expected", domain.OperatorGt, "abc", 200, false, "cannot compare"},
		{"lt pass", domain.OperatorLt, "300", 200, true, ""},
		{"lt fail", domain.OperatorLt, "100", 200, false, "<"},
		{"lt non-numeric expected", domain.OperatorLt, "abc", 200, false, "cannot compare"},
		{"unknown operator", "weird", "1", 200, false, "unknown operator"},
		{"nil expected", domain.OperatorEq, nil, 200, false, "expected"},
		{"empty operator defaults to eq", "", "200", 200, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := domain.Assertion{Type: domain.AssertionStatus, Operator: tc.op, Expected: tc.expected}
			res := eval.Evaluate(a, resp(tc.status, nil, ""))
			if res.Passed != tc.wantPass {
				t.Fatalf("passed=%v want %v; msg=%q", res.Passed, tc.wantPass, res.Message)
			}
			if tc.wantMsg != "" && !strings.Contains(res.Message, tc.wantMsg) {
				t.Errorf("message %q should contain %q", res.Message, tc.wantMsg)
			}
		})
	}
}

func TestGt_NonNumericActual(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionHeader, Path: "X-N", Operator: domain.OperatorGt, Expected: "1"}
	res := eval.Evaluate(a, resp(200, map[string]string{"X-N": "abc"}, ""))
	if res.Passed || !strings.Contains(res.Message, "cannot compare") {
		t.Errorf("unexpected result %+v", res)
	}
}

func TestEvaluateAll_MixedResults(t *testing.T) {
	as := []domain.Assertion{
		{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
		{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "500"},
	}
	res := eval.EvaluateAll(as, resp(200, nil, ""))
	if len(res) != 2 || !res[0].Passed || res[1].Passed {
		t.Errorf("unexpected results %+v", res)
	}
}
