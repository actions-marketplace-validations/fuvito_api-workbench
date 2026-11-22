package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const xmlExampleSpec = `openapi: "3.0.3"
info: {title: xml, version: "1"}
paths:
  /note:
    post:
      requestBody:
        content:
          application/xml:
            schema: {type: object, properties: {text: {type: string}}, xml: {name: note}}
            example: "<note><text>hello</text></note>"
      responses:
        "200":
          description: ok
          content:
            application/xml:
              schema: {type: object, properties: {text: {type: string}}}
              example: "<note><text>hello</text></note>"
`

// F-034: an XML example string is not checked against the JSON schema, so the document loads.
func TestParse_AcceptsAnExampleForANonJSONMediaType(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(xmlExampleSpec), "yaml")
	if err != nil {
		t.Fatalf("a valid document with an XML example must load: %v", err)
	}
	if reqs, err := openapi.GenerateRequests(spec, "http://x"); err != nil || len(reqs) != 1 {
		t.Errorf("requests=%d err=%v", len(reqs), err)
	}
}

// A wrong example for a JSON media type is still rejected.
func TestParse_StillRejectsAWrongJSONExample(t *testing.T) {
	bad := strings.Replace(xmlExampleSpec, "application/xml", "application/json", 1)
	bad = strings.Replace(bad, `example: "<note><text>hello</text></note>"`, `example: "not an object"`, 1)
	if _, err := openapi.ParseBytes([]byte(bad), "yaml"); err == nil {
		t.Error("a JSON example that does not fit its schema is still an invalid document")
	}
}

const cookieAuthSpec = `openapi: "3.0.3"
info: {title: c, version: "1"}
components:
  securitySchemes:
    sess: {type: apiKey, in: cookie, name: session}
paths:
  /me:
    get:
      security: [{sess: []}]
      parameters:
        - {name: theme, in: cookie, required: true, schema: {type: string}}
      responses: {"200": {description: ok}}
`

// F-035: the no-credentials probe drops an API key sent in a cookie, and keeps other cookies.
func TestProbes_NoCredentialsDropsACookieAPIKey(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(cookieAuthSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	probes, _ := openapi.GenerateNegativeProbes(spec, "http://x")
	for _, p := range probes {
		if p.Kind != openapi.ProbeMissingAuth {
			continue
		}
		var cookie string
		for _, h := range p.Request.Headers {
			if strings.EqualFold(h.Name, "Cookie") {
				cookie = h.Value
			}
		}
		if strings.Contains(cookie, "session=") || !strings.Contains(cookie, "theme=") {
			t.Errorf("the probe must drop the session cookie and keep the others: %q", cookie)
		}
		return
	}
	t.Fatal("no missing-credentials probe")
}
