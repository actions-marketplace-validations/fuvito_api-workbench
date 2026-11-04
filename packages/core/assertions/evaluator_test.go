package assertions_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/assertions"
	"github.com/fuvito/api-workbench/packages/core/domain"
)

func resp(statusCode int, headers map[string]string, body string) *domain.Response {
	r := &domain.Response{StatusCode: statusCode, Status: "200 OK"}
	for k, v := range headers {
		r.Headers = append(r.Headers, domain.Header{Name: k, Value: v})
	}
	r.Body = []byte(body)
	return r
}

var eval = assertions.New()

func TestStatus_Eq_Pass(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"}
	res := eval.Evaluate(a, resp(200, nil, ""))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestStatus_Eq_Fail(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"}
	res := eval.Evaluate(a, resp(404, nil, ""))
	if res.Passed {
		t.Error("expected fail")
	}
	if res.Actual != "404" {
		t.Errorf("actual: got %q", res.Actual)
	}
}

func TestStatus_NumericExpected(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 201}
	res := eval.Evaluate(a, resp(201, nil, ""))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestStatus_Gt(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionStatus, Operator: domain.OperatorGt, Expected: "199"}
	res := eval.Evaluate(a, resp(200, nil, ""))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestHeader_Contains_Pass(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionHeader, Path: "Content-Type", Operator: domain.OperatorContains, Expected: "application/json"}
	res := eval.Evaluate(a, resp(200, map[string]string{"Content-Type": "application/json; charset=utf-8"}, ""))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestHeader_CaseInsensitive(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionHeader, Path: "content-type", Operator: domain.OperatorEq, Expected: "text/plain"}
	res := eval.Evaluate(a, resp(200, map[string]string{"Content-Type": "text/plain"}, ""))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestHeader_Missing_Fail(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionHeader, Path: "X-Custom", Operator: domain.OperatorEq, Expected: "value"}
	res := eval.Evaluate(a, resp(200, nil, ""))
	if res.Passed {
		t.Error("expected fail for missing header")
	}
}

func TestJSONPath_Eq_Pass(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.name", Operator: domain.OperatorEq, Expected: "Alice"}
	res := eval.Evaluate(a, resp(200, nil, `{"name":"Alice","age":30}`))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestJSONPath_Nested(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.user.id", Operator: domain.OperatorEq, Expected: "42"}
	res := eval.Evaluate(a, resp(200, nil, `{"user":{"id":42}}`))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestJSONPath_Contains(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.message", Operator: domain.OperatorContains, Expected: "success"}
	res := eval.Evaluate(a, resp(200, nil, `{"message":"operation success"}`))
	if !res.Passed {
		t.Errorf("expected pass; message: %s", res.Message)
	}
}

func TestJSONPath_BadPath(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.nonexistent", Operator: domain.OperatorEq, Expected: "x"}
	res := eval.Evaluate(a, resp(200, nil, `{"name":"Alice"}`))
	if res.Passed {
		t.Error("expected fail for missing path")
	}
}

func TestJSONPath_NonJSON_Body(t *testing.T) {
	a := domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.key", Operator: domain.OperatorEq, Expected: "val"}
	res := eval.Evaluate(a, resp(200, nil, "not json"))
	if res.Passed {
		t.Error("expected fail for non-JSON body")
	}
}

func TestEvaluateAll(t *testing.T) {
	r := resp(201, map[string]string{"Content-Type": "application/json"}, `{"id":1}`)
	results := eval.EvaluateAll([]domain.Assertion{
		{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "201"},
		{Type: domain.AssertionHeader, Path: "Content-Type", Operator: domain.OperatorContains, Expected: "json"},
		{Type: domain.AssertionJSONPath, Path: "$.id", Operator: domain.OperatorEq, Expected: "1"},
	}, r)
	for i, res := range results {
		if !res.Passed {
			t.Errorf("assertion %d failed: %s", i, res.Message)
		}
	}
}
