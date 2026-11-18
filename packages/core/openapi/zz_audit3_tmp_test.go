package openapi

import (
	"strings"
	"testing"
)

func TestAuditStatusSemantics(t *testing.T) {
	spec := strings.Replace(auditSpec, "servers:\n  - url: https://api.example.com/v1\n", "", 1)
	s, err := ParseBytes([]byte(spec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	p := "/users/5"
	t.Logf("AUDIT undeclared 500 text/plain (ByPath): %v", ValidateResponseByPath(s, "GET", p, 500, resp(500, `oops`, "text/plain")))
	t.Logf("AUDIT undeclared 500 (ByOperation)      : %v", ValidateResponse(s, "getUser", resp(500, `oops`, "text/plain")))
	t.Logf("AUDIT empty body on 200 (ByPath)        : %v", ValidateResponseByPath(s, "GET", p, 200, resp(200, ``, "")))
	t.Logf("AUDIT empty body on 200 (ByOperation)   : %v", ValidateResponse(s, "getUser", resp(200, ``, "")))
	t.Logf("AUDIT html on 200 (ByPath)              : %v", ValidateResponseByPath(s, "GET", p, 200, resp(200, `<html>`, "text/html")))
	t.Logf("AUDIT html on 200 (ByOperation)         : %v", ValidateResponse(s, "getUser", resp(200, `<html>`, "text/html")))
	t.Logf("AUDIT missing required id (ByPath)      : %v", ValidateResponseByPath(s, "GET", p, 200, resp(200, `{}`, "application/json")))
	t.Logf("AUDIT extra undeclared field (ByPath)   : %v", ValidateResponseByPath(s, "GET", p, 200, resp(200, `{"id":1,"secret":"x"}`, "application/json")))
	t.Logf("AUDIT wrong content-type json body (ByPath): %v", ValidateResponseByPath(s, "GET", p, 200, resp(200, `{"id":1}`, "application/xml")))
	t.Logf("AUDIT 404 with body when none declared  : %v", ValidateResponseByPath(s, "GET", p, 404, resp(404, `{"error":"x"}`, "application/json")))
}
