package openapi

import (
	"fmt"
	"strings"
	"testing"
)

func TestAuditServersMatrix(t *testing.T) {
	cases := []struct{ name, servers string }{
		{"no servers block", ""},
		{"relative /v1", "servers:\n  - url: /v1\n"},
		{"host only http://localhost:8080", "servers:\n  - url: http://localhost:8080\n"},
		{"host+path https://api.example.com/v1", "servers:\n  - url: https://api.example.com/v1\n"},
		{"host only https://api.example.com", "servers:\n  - url: https://api.example.com\n"},
		{"variable server", "servers:\n  - url: https://{env}.example.com/v1\n    variables: {env: {default: prod}}\n"},
	}
	bad := resp(200, `{"id":"not-int"}`, "application/json")
	for _, c := range cases {
		spec := strings.Replace(auditSpec, "servers:\n  - url: https://api.example.com/v1\n", c.servers, 1)
		s, err := ParseBytes([]byte(spec), "yaml")
		if err != nil {
			t.Logf("AUDIT [%s] parse error: %v", c.name, err)
			continue
		}
		out := ""
		for _, p := range []string{"/users/5", "/v1/users/5"} {
			v := ValidateResponseByPath(s, "GET", p, 200, bad)
			out += fmt.Sprintf(" %s=>%d", p, len(v))
		}
		t.Logf("AUDIT [%-38s] bad body detected (violations):%s", c.name, out)
	}
}
