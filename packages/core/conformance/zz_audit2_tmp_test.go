package conformance

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const reqSpec = `openapi: 3.0.3
info: {title: t, version: "1"}
SERVERS
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, required: [name], properties: {name: {type: string}}}
      responses:
        "201": {description: created}
`

func TestAuditRequestValidation(t *testing.T) {
	for _, servers := range []string{"", "servers:\n  - url: http://localhost:8080\n", "servers:\n  - url: https://api.example.com/v1\n"} {
		s, err := openapi.ParseBytes([]byte(strings.Replace(reqSpec, "SERVERS\n", servers, 1)), "yaml")
		if err != nil {
			t.Fatal(err)
		}
		for _, url := range []string{"http://localhost:8080/users", "http://127.0.0.1:9999/users", "https://api.example.com/v1/users"} {
			v := openapi.ValidateRequest(s, &domain.Request{Method: domain.MethodPOST, URL: url,
				Body: &domain.RequestBody{Type: domain.BodyJSON, Content: `{}`}})
			t.Logf("AUDIT servers=%-42q url=%-36s invalid body flagged=%d", strings.TrimSpace(servers), url, len(v))
		}
	}
}
