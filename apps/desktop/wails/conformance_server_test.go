package main

import (
	"net/http"
	"sort"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// Each operation is called once and validated against the embedded spec, the
// same way scenario steps are. Exactly the documented operations must fail.
func TestConformanceServerViolationsAreExactlyTheKnownOnes(t *testing.T) {
	spec, err := openapi.ParseBytes(conformanceOpenAPIYAML, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	calls := []struct {
		op, method, path, body string
	}{
		{"getHealth", "GET", "/health", ""},
		{"listProducts", "GET", "/products", ""},
		{"createProduct", "POST", "/products", `{"name":"Lamp","price":12.5}`},
		{"getProduct", "GET", "/products/1", ""},
		{"deleteProduct", "DELETE", "/products/2", ""},
		{"listCategories", "GET", "/categories", ""},
		{"listOrders", "GET", "/orders", ""},
	}
	var failing []string
	for _, c := range calls {
		code, hdr, body := doDemo(t, newConformanceMux(), demoCall{
			method: c.method, path: c.path, body: c.body,
			headers: map[string]string{"Content-Type": "application/json"},
		})
		resp := &domain.Response{StatusCode: code, Body: []byte(body), Headers: toDomainHeaders(hdr)}
		if v := openapi.ValidateResponseByPath(spec, c.method, c.path, code, resp); len(v) > 0 {
			failing = append(failing, c.op)
		}
	}
	want := append([]string(nil), conformanceKnownViolations...)
	sort.Strings(want)
	sort.Strings(failing)
	if len(failing) != len(want) {
		t.Fatalf("failing = %v, want %v", failing, want)
	}
	for i := range want {
		if failing[i] != want[i] {
			t.Fatalf("failing = %v, want %v", failing, want)
		}
	}
}

func toDomainHeaders(h http.Header) []domain.Header {
	var out []domain.Header
	for k, vs := range h {
		for _, v := range vs {
			out = append(out, domain.Header{Name: k, Value: v})
		}
	}
	return out
}
