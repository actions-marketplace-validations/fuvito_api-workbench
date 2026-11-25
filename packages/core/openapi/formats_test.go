package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const formatSpec = `openapi: "3.0.3"
info: {title: formats, version: "1"}
paths:
  /f:
    get:
      operationId: f
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  mail: {type: string, format: email}
                  uid: {type: string, format: uuid}
                  v4: {type: string, format: ipv4}
                  v6: {type: string, format: ipv6}
                  day: {type: string, format: date}
                  at: {type: string, format: date-time}
                  other: {type: string, format: made-up}
`

func TestFormats_EachSupportedFormat(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(formatSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		field, value string
		ok           bool
	}{
		{"mail", "ana@example.com", true},
		{"mail", "a.b+tag@sub.example.org", true},
		{"mail", "user@localhost", true},
		{"mail", "nope", false},
		{"mail", "two@@example.com", false},
		{"mail", "Ana <ana@example.com>", false},
		{"mail", "ana @example.com", false},
		{"mail", "", false},
		{"uid", "0b6f6b2e-8d2a-4c53-9a53-5c1f0a3d9e11", true},
		{"uid", "00000000-0000-0000-0000-000000000000", true},
		{"uid", "0B6F6B2E-8D2A-4C53-9A53-5C1F0A3D9E11", true},
		{"uid", "123", false},
		{"uid", "0b6f6b2e8d2a4c539a535c1f0a3d9e11", false},
		{"v4", "192.168.0.1", true},
		{"v4", "256.1.1.1", false},
		{"v4", "::1", false},
		{"v6", "::1", true},
		{"v6", "2001:db8::ff00:42:8329", true},
		{"v6", "192.168.0.1", false},
		{"day", "2026-11-24", true},
		{"day", "24/11/2026", false},
		{"at", "2026-11-24T10:00:00Z", true},
		{"at", "yesterday", false},
		{"other", "anything at all", true},
	}
	for _, c := range cases {
		body := `{"` + c.field + `":"` + c.value + `"}`
		got := byPath(spec, "/f", 200, ctype("application/json"), body)
		if (got == 0) != c.ok {
			t.Errorf("%s = %q: violations=%d, want ok=%v", c.field, c.value, got, c.ok)
		}
	}
}

func TestFormats_MessageNamesTheFormat(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(formatSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	errs := openapi.ValidateResponseByPath(spec, "GET", "/f", 200, &domain.Response{StatusCode: 200, Headers: ctype("application/json"), Body: []byte(`{"mail":"nope"}`)})
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), `format "email"`) || !strings.Contains(errs[0].Error(), "/mail") {
		t.Fatalf("%v", errs)
	}
}

// A spec whose example breaks one of the formats checked since 0.9.0 must keep loading; one whose
// example has the wrong type is still refused, as before.
func TestFormats_ABadFormatExampleDoesNotRefuseTheSpec(t *testing.T) {
	const doc = `openapi: "3.0.3"
info: {title: e, version: "1"}
paths:
  /u:
    get:
      operationId: u
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string, format: uuid, example: "123"}
`
	spec, err := openapi.ParseBytes([]byte(doc), "yaml")
	if err != nil {
		t.Fatalf("a spec with a badly formatted example must load: %v", err)
	}
	if p := spec.IgnoredExampleProblem(); !strings.Contains(p, `format "uuid"`) {
		t.Errorf("the problem is kept for the caller: %q", p)
	}
	if got := byPath(spec, "/u", 200, ctype("application/json"), `{"id":"123"}`); got == 0 {
		t.Error("the format is still checked in responses")
	}

	wrong := strings.Replace(doc, `example: "123"`, `example: 5`, 1)
	if _, err := openapi.ParseBytes([]byte(wrong), "yaml"); err == nil {
		t.Error("an example of the wrong type is still refused")
	}
	if clean, err := openapi.ParseBytes([]byte(formatSpec), "yaml"); err != nil || clean.IgnoredExampleProblem() != "" {
		t.Errorf("no problem when the examples are fine: %v", err)
	}
}

func TestFormats_SupportedListMatchesTheCode(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(formatSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range openapi.SupportedFormats {
		if name == "byte" {
			continue // base64 is checked by the library; it has no case in formatSpec
		}
		doc := strings.Replace(formatSpec, "format: made-up", "format: "+name, 1)
		if _, err := openapi.ParseBytes([]byte(doc), "yaml"); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	_ = spec
}
