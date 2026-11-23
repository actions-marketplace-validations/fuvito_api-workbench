package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const generationSpec = `openapi: "3.1.0"
info: {title: g, version: "1"}
servers: [{url: "http://x"}]
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
    basicAuth: {type: http, scheme: basic}
    apiKeyHeader: {type: apiKey, in: header, name: X-API-Key}
    apiKeyQuery: {type: apiKey, in: query, name: api_key}
paths:
  /me:
    get:
      security: [{bearerAuth: []}]
      responses: {"200": {description: ok}}
  /audit:
    get:
      security: [{basicAuth: []}]
      responses: {"200": {description: ok}}
  /flags:
    get:
      security: [{apiKeyHeader: []}]
      responses: {"200": {description: ok}}
  /partner:
    get:
      security: [{apiKeyQuery: []}]
      responses: {"200": {description: ok}}
  /optional:
    get:
      security: [{}, {bearerAuth: []}]
      responses: {"200": {description: ok}}
  /prefs:
    get:
      parameters:
        - {name: theme, in: cookie, required: true, schema: {type: string}, example: dark}
      responses: {"200": {description: ok}}
  /lookup:
    get:
      parameters:
        - {name: ids, in: query, required: true, style: form, explode: false, schema: {type: array, items: {type: integer}}, example: [1, 2, 3]}
        - {name: tags, in: query, required: true, schema: {type: array, items: {type: string}}, example: [a, b]}
      responses: {"200": {description: ok}}
  /payments:
    post:
      requestBody:
        content:
          application/json:
            schema:
              example: {kind: card, amount: 5, cardLast4: "4242"}
              oneOf:
                - {type: object, required: [kind, amount, cardLast4], properties: {kind: {type: string}, amount: {type: number}, cardLast4: {type: string}}}
                - {type: object, required: [kind, iban], properties: {kind: {type: string}, iban: {type: string}}}
      responses: {"201": {description: ok}}
  /customers:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [email, age]
              properties:
                email: {type: string, examples: [a@b.c]}
                age: {type: integer, examples: [30]}
      responses: {"201": {description: ok}}
  /form:
    post:
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {type: object, required: [name, qty], properties: {name: {type: string, example: pen}, qty: {type: integer}}}
      responses: {"201": {description: ok}}
  /upload:
    post:
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema: {type: object, required: [title], properties: {title: {type: string, example: hi}}}
      responses: {"201": {description: ok}}
`

func generated(t *testing.T) map[string]*domain.Request {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(generationSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	reqs, err := openapi.GenerateRequests(spec, "")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*domain.Request{}
	for _, r := range reqs {
		out[r.Path] = r.Request
	}
	return out
}

// CS-REQ-003: credentials are applied for the scheme the operation declares, not only for bearer.
func TestGenerate_AppliesEachSecurityScheme(t *testing.T) {
	g := generated(t)
	if a := g["/me"].Auth; a == nil || a.Type != domain.AuthBearer || a.Bearer.Token != "{{token}}" {
		t.Errorf("bearer: %+v", a)
	}
	if a := g["/audit"].Auth; a == nil || a.Type != domain.AuthBasic || a.Basic.Username != "{{basicUser}}" || a.Basic.Password != "{{basicPass}}" {
		t.Errorf("basic: %+v", a)
	}
	if a := g["/flags"].Auth; a == nil || a.Type != domain.AuthAPIKey || a.APIKey.Key != "X-API-Key" || a.APIKey.Location != domain.APIKeyInHeader || a.APIKey.Value != "{{apiKey}}" {
		t.Errorf("api key in header: %+v", a)
	}
	if a := g["/partner"].Auth; a == nil || a.Type != domain.AuthAPIKey || a.APIKey.Key != "api_key" || a.APIKey.Location != domain.APIKeyInQuery {
		t.Errorf("api key in query: %+v", a)
	}
	if g["/optional"].Auth != nil {
		t.Errorf("optional security must add no credentials: %+v", g["/optional"].Auth)
	}
}

// CS-REQ-003: a required cookie parameter is sent as a Cookie header.
func TestGenerate_SendsRequiredCookieParameters(t *testing.T) {
	var got string
	for _, h := range generated(t)["/prefs"].Headers {
		if h.Name == "Cookie" {
			got = h.Value
		}
	}
	if got != "theme=dark" {
		t.Errorf("Cookie = %q, want theme=dark", got)
	}
}

// CS-REQ-003: an array parameter is written by its style and explode, not as Go slice text.
func TestGenerate_SerializesArrayQueryParameters(t *testing.T) {
	var ids []string
	var tags []string
	for _, q := range generated(t)["/lookup"].QueryParams {
		switch q.Name {
		case "ids":
			ids = append(ids, q.Value)
		case "tags":
			tags = append(tags, q.Value)
		}
	}
	if strings.Join(ids, "|") != "1,2,3" {
		t.Errorf("ids (explode false) = %v, want one value 1,2,3", ids)
	}
	if strings.Join(tags, "|") != "a|b" {
		t.Errorf("tags (explode default) = %v, want a and b as separate values", tags)
	}
}

// CS-REQ-003: an example given on the schema of a oneOf body, or as an OpenAPI 3.1 `examples` list,
// is used for the body.
func TestGenerate_UsesBodyExamples(t *testing.T) {
	g := generated(t)
	pay := g["/payments"].Body.Content
	for _, want := range []string{`"kind": "card"`, `"amount": 5`, `"cardLast4": "4242"`} {
		if !strings.Contains(pay, want) {
			t.Errorf("oneOf body should use the schema example; missing %s in %s", want, pay)
		}
	}
	if strings.Contains(pay, "{{") {
		t.Errorf("no placeholder expected: %s", pay)
	}
	cust := g["/customers"].Body.Content
	if !strings.Contains(cust, `"email": "a@b.c"`) || !strings.Contains(cust, `"age": 30`) || strings.Contains(cust, "{{") {
		t.Errorf("3.1 examples should fill the body: %s", cust)
	}
}

// CS-REQ-003 and CS-GAP-006: a form or multipart body is generated from its schema and the request
// check can read it, so a correct operation does not fail on its own generated request.
func TestGenerate_FormAndMultipartBodies(t *testing.T) {
	g := generated(t)
	form := g["/form"].Body
	if form == nil || form.Type != domain.BodyFormURLEncoded || form.Content != "name=pen&qty={{qty}}" {
		t.Errorf("form body: %+v", form)
	}
	up := g["/upload"].Body
	if up == nil || up.Type != domain.BodyMultipart || up.Content != "title=hi" {
		t.Errorf("multipart body: %+v", up)
	}
	spec, _ := openapi.ParseBytes([]byte(generationSpec), "yaml")
	for _, path := range []string{"/form", "/upload"} {
		req := *g[path]
		b := *req.Body
		b.Content = strings.ReplaceAll(b.Content, "{{qty}}", "3")
		req.Body = &b
		if v := openapi.ValidateRequest(spec, &req); len(v) != 0 {
			t.Errorf("%s: generated request does not fit the spec: %+v", path, v)
		}
	}
}
