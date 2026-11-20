package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

const stubSpec = `
openapi: "3.0.3"
info: {title: s, version: "1"}
paths:
  /typed:
    post:
      operationId: typed
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [title, price, inStock, tags, size, label]
              properties:
                id: {type: integer, readOnly: true}
                title: {type: string}
                price: {type: number}
                inStock: {type: boolean}
                tags: {type: array, items: {type: string}}
                size: {type: object, properties: {w: {type: integer}}}
                label: {type: string, example: Lamp}
                kind: {type: string, enum: [a, b]}
                sku: {type: string, default: A-100}
                note: {type: string}
      responses: {"201": {description: ok}}
  /composed:
    post:
      operationId: composed
      requestBody:
        content:
          application/json:
            schema:
              allOf:
                - {type: object, required: [id], properties: {id: {type: integer}}}
                - {type: object, required: [label], properties: {label: {type: string}, extra: {type: string}}}
      responses: {"201": {description: ok}}
  /either:
    post:
      operationId: either
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - {type: object, required: [card], properties: {card: {type: string}}}
                - {type: object, required: [iban], properties: {iban: {type: string}}}
      responses: {"201": {description: ok}}
  /many:
    post:
      operationId: many
      requestBody:
        content:
          application/json:
            schema:
              type: array
              items: {type: object, required: [qty], properties: {qty: {type: integer}}}
      responses: {"201": {description: ok}}
  /nothing:
    post:
      operationId: nothing
      requestBody:
        content:
          application/json:
            schema: {type: object, properties: {a: {type: string}, b: {type: integer}}}
      responses: {"201": {description: ok}}
`

func stubOf(t *testing.T, spec *openapi.Spec, op string) string {
	t.Helper()
	reqs, err := openapi.GenerateRequests(spec, "http://x")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reqs {
		if r.OperationID == op {
			if r.Request.Body == nil {
				t.Fatalf("%s has no body", op)
			}
			return r.Request.Body.Content
		}
	}
	t.Fatalf("operation %s not found", op)
	return ""
}

func TestBodyStub_TypedRequiredOnlyWithLiterals(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(stubSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "title": "{{title}}",
  "price": {{price}},
  "inStock": {{inStock}},
  "tags": {{tags}},
  "size": {{size}},
  "label": "Lamp",
  "kind": "a",
  "sku": "A-100"
}`
	if got := stubOf(t, spec, "typed"); got != want {
		t.Fatalf("stub =\n%s\nwant\n%s", got, want)
	}
}

func TestBodyStub_CompositionAndShapes(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(stubSpec), "yaml")
	if got := stubOf(t, spec, "composed"); got != "{\n  \"id\": {{id}},\n  \"label\": \"{{label}}\"\n}" {
		t.Errorf("allOf members must be merged (id and label are required, extra is not):\n%s", got)
	}
	if got := stubOf(t, spec, "either"); got != "{\n  \"card\": \"{{card}}\"\n}" {
		t.Errorf("oneOf uses the first branch:\n%s", got)
	}
	if got := stubOf(t, spec, "many"); got != "[\n  {\n    \"qty\": {{qty}}\n  }\n]" {
		t.Errorf("an array body is an array of the item stub:\n%s", got)
	}
	if got := stubOf(t, spec, "nothing"); got != "{}" {
		t.Errorf("a schema with no required properties asks for nothing:\n%s", got)
	}
}

// The point of typed placeholders: with plain values supplied for them, the generated request is a
// valid request for the spec. With the old quoted-string stubs this was impossible for any
// number, boolean, array or object property.
func TestBodyStub_FilledInRequestIsValid(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(stubSpec), "yaml")
	reqs, _ := openapi.GenerateRequests(spec, "http://x")
	env := &domain.Environment{Name: "e", Variables: []domain.EnvironmentVariable{
		{Name: "title", Value: "Desk"}, {Name: "price", Value: "9.99"}, {Name: "inStock", Value: "true"},
		{Name: "tags", Value: `["a","b"]`}, {Name: "size", Value: `{"w":3}`},
		{Name: "id", Value: "1"}, {Name: "label", Value: "x"}, {Name: "card", Value: "4111"}, {Name: "qty", Value: "2"},
	}}
	for _, r := range reqs {
		filled := variables.NewResolver(env).InterpolateRequest(r.Request)
		if strings.Contains(filled.Body.Content, "{{") {
			t.Errorf("%s: placeholders left after filling in: %s", r.OperationID, filled.Body.Content)
		}
		if v := openapi.ValidateRequest(spec, filled); len(v) != 0 {
			t.Errorf("%s: the filled-in request must be valid, got %v\nbody: %s", r.OperationID, v, filled.Body.Content)
		}
	}
}

const sw2Stub = `{
  "swagger": "2.0",
  "info": {"title": "s", "version": "1"},
  "definitions": {
    "Base": {"type": "object", "required": ["id"], "properties": {"id": {"type": "integer"}}},
    "Item": {
      "allOf": [
        {"$ref": "#/definitions/Base"},
        {"type": "object", "required": ["price", "ok"], "properties": {
          "price": {"type": "number"}, "ok": {"type": "boolean"}, "label": {"type": "string", "example": "Lamp"}, "note": {"type": "string"}}}
      ]
    }
  },
  "paths": {
    "/items": {"post": {
      "operationId": "createItem",
      "parameters": [{"in": "body", "name": "body", "schema": {"$ref": "#/definitions/Item"}}],
      "responses": {"201": {"description": "ok"}}
    }},
    "/batch": {"post": {
      "operationId": "createBatch",
      "parameters": [{"in": "body", "name": "body", "schema": {"type": "array", "items": {"$ref": "#/definitions/Base"}}}],
      "responses": {"201": {"description": "ok"}}
    }}
  }
}`

func TestBodyStub_Swagger2UsesTheSameRules(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2Stub), "json", "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range reqs {
		got[r.OperationID] = r.Request.Body.Content
	}
	if want := "{\n  \"id\": {{id}},\n  \"price\": {{price}},\n  \"ok\": {{ok}},\n  \"label\": \"Lamp\"\n}"; got["createItem"] != want {
		t.Errorf("createItem (allOf through $ref, typed, required plus example):\n%s\nwant\n%s", got["createItem"], want)
	}
	if want := "[\n  {\n    \"id\": {{id}}\n  }\n]"; got["createBatch"] != want {
		t.Errorf("createBatch (array of a $ref):\n%s", got["createBatch"])
	}
}
