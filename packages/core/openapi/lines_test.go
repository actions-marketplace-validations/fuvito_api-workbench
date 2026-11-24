package openapi_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

func TestOperationLines(t *testing.T) {
	yamlSpec := "openapi: \"3.0.3\"\ninfo: {title: t, version: \"1\"}\npaths:\n  /a:\n    parameters: []\n    get:\n      operationId: a\n    post:\n      operationId: b\n  /b/{id}:\n    delete:\n      operationId: c\n"
	got := openapi.OperationLines([]byte(yamlSpec))
	want := map[string]int{"GET /a": 6, "POST /a": 8, "DELETE /b/{id}": 11}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: line %d, want %d", k, got[k], v)
		}
	}

	jsonSpec := "{\n  \"paths\": {\n    \"/x\": {\n      \"get\": {}\n    }\n  }\n}\n"
	if got := openapi.OperationLines([]byte(jsonSpec)); got["GET /x"] != 4 {
		t.Errorf("json: %v", got)
	}
	if got := openapi.OperationLines([]byte("not: [valid")); len(got) != 0 {
		t.Errorf("an unreadable document has no lines: %v", got)
	}
}
