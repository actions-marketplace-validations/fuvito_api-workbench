package openapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const probeSpec = `
openapi: "3.0.3"
info:
  title: probes
  version: "1"
servers:
  - url: https://api.example.com/
security:
  - bearerAuth: []
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /items:
    get:
      operationId: listItems
      security: []
      parameters:
        - name: page
          in: query
          required: true
          schema: {type: integer}
        - name: filter
          in: query
          required: false
          schema: {type: string}
      responses:
        "200": {description: ok}
    post:
      operationId: createItem
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name, count, flag, tags, meta, untyped]
              properties:
                name: {type: string}
                count: {type: integer}
                flag: {type: boolean}
                tags: {type: array, items: {type: string}}
                meta: {type: object}
                untyped: {}
                optional: {type: string}
      responses:
        "201": {description: created}
  /plain:
    delete:
      responses:
        "204": {description: gone}
`

func probesByKind(probes []openapi.NegativeProbe) map[openapi.ProbeKind][]openapi.NegativeProbe {
	m := map[openapi.ProbeKind][]openapi.NegativeProbe{}
	for _, p := range probes {
		m[p.Kind] = append(m[p.Kind], p)
	}
	return m
}

func TestGenerateNegativeProbes(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(probeSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	probes, err := openapi.GenerateNegativeProbes(spec, "")
	if err != nil {
		t.Fatal(err)
	}
	by := probesByKind(probes)

	// listItems opts out of global security; createItem and DELETE /plain inherit it.
	auth := by[openapi.ProbeMissingAuth]
	if len(auth) != 2 {
		t.Fatalf("expected 2 missing-auth probes, got %d", len(auth))
	}
	for _, p := range auth {
		if p.Request.Auth != nil {
			t.Errorf("%s %s: auth should be cleared", p.Method, p.Path)
		}
		if p.OperationID == "listItems" {
			t.Error("listItems has security: [] and must not get an auth probe")
		}
	}

	if got := len(by[openapi.ProbeDropRequiredField]); got != 6 {
		t.Fatalf("expected 6 drop-field probes, got %d", got)
	}
	for _, p := range by[openapi.ProbeDropRequiredField] {
		var body map[string]any
		if err := json.Unmarshal([]byte(p.Request.Body.Content), &body); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if len(body) != 6 { // 7 properties minus the dropped one
			t.Errorf("expected 6 remaining properties, got %v", body)
		}
	}

	// "untyped" has no schema type so it gets no wrong-type probe.
	wrong := by[openapi.ProbeWrongFieldType]
	if len(wrong) != 5 {
		t.Fatalf("expected 5 wrong-type probes, got %d", len(wrong))
	}
	for _, p := range wrong {
		if strings.Contains(p.Description, "'untyped'") {
			t.Error("untyped field must not get a wrong-type probe")
		}
	}

	params := by[openapi.ProbeMissingRequiredParam]
	if len(params) != 1 {
		t.Fatalf("expected 1 missing-param probe, got %d", len(params))
	}
	for _, q := range params[0].Request.QueryParams {
		if strings.EqualFold(q.Name, "page") {
			t.Error("required param should have been removed")
		}
	}
	if !strings.HasPrefix(params[0].Request.URL, "https://api.example.com") {
		t.Errorf("base URL should come from servers: %q", params[0].Request.URL)
	}
}

func TestGenerateNegativeProbes_ExplicitBaseURL(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(probeSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	probes, err := openapi.GenerateNegativeProbes(spec, "http://localhost:9000")
	if err != nil {
		t.Fatal(err)
	}
	if len(probes) == 0 {
		t.Fatal("expected probes")
	}
	for _, p := range probes {
		if !strings.HasPrefix(p.Request.URL, "http://localhost:9000") {
			t.Errorf("unexpected URL %q", p.Request.URL)
		}
	}
}

func TestGenerateNegativeProbes_ExampleSpec(t *testing.T) {
	probes, err := openapi.GenerateNegativeProbes(loadExampleSpec(t), "http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range probes {
		if p.Description == "" || p.Method == "" || p.Path == "" || p.Request == nil {
			t.Errorf("incomplete probe: %+v", p)
		}
	}
}
