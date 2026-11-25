package reporting_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

func runAgainst(t *testing.T, specYAML string, status int, body string) *conformance.ConformResult {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(specYAML), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, nil, conformance.Options{})
}

// Pins CS-GAP-016: an error status the spec declares counts as conforming, so the operation passes
// although it did not do what it is for.
func TestConform_ADeclaredErrorAnswerPasses_KnownGap(t *testing.T) {
	result := runAgainst(t, `
openapi: "3.0.3"
info: {title: NotFound, version: "1"}
paths:
  /thing:
    get:
      responses:
        "200": {description: ok}
        "404": {description: not found}
`, http.StatusNotFound, "")
	if len(result.Operations) != 1 || !result.OpPassed(result.Operations[0]) {
		t.Errorf("a declared 404 no longer passes: update CS-GAP-016 (%+v)", result.Operations)
	}
}

// Pins CS-GAP-017: the rules are about names and shapes. A secret in a field with an ordinary name,
// too short to look like a token, is quoted when it is itself the value that broke the spec.
func TestConform_AnUnrecognisableSecretThatBreaksTheSpecIsQuoted_KnownGap(t *testing.T) {
	result := runAgainst(t, `
openapi: "3.0.3"
info: {title: Plain, version: "1"}
paths:
  /thing:
    get:
      responses:
        "200":
          description: ok
          content: {application/json: {schema: {type: object, properties: {reference: {type: integer}}}}}
`, http.StatusOK, `{"reference":"hunter2pw"}`)
	if out := reporting.ConformText(result, nil); !strings.Contains(out, "hunter2pw") {
		t.Errorf("the offending value is no longer quoted: update CS-GAP-017\n%s", out)
	}
}
