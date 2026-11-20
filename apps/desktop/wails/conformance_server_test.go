package main

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
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

func TestChoosePort(t *testing.T) {
	// Automatic: the first free port from the start.
	auto, err := choosePort(0, 8300, "x")
	if err != nil || auto < 8300 || auto >= 8330 {
		t.Fatalf("auto = %d, %v", auto, err)
	}

	// Configured and free: exactly that port.
	free, err := findFreePort(8400)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := choosePort(free, 8300, "x"); err != nil || got != free {
		t.Fatalf("configured free port: got %d, %v", got, err)
	}

	// Configured and taken: an error that says where to change it, never another port.
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", free))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, err = choosePort(free, 8300, "the Conformance Server port")
	if err == nil || !strings.Contains(err.Error(), "already in use") || !strings.Contains(err.Error(), "the Conformance Server port") {
		t.Fatalf("taken port: %v", err)
	}
}

func TestConformanceServerStartUsesTheConfiguredPort(t *testing.T) {
	free, err := findFreePort(8500)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewConformanceServerService()
	svc.portFn = func() int { return free }
	res := svc.Start()
	defer svc.Stop()
	if res.Error != "" || res.Port != free {
		t.Fatalf("start: %+v, want port %d", res, free)
	}

	other := NewConformanceServerService()
	other.portFn = func() int { return free }
	if res := other.Start(); res.Error == "" || res.Port != 0 {
		t.Fatalf("a taken configured port must fail: %+v", res)
	}
}

func TestSanitizePort(t *testing.T) {
	for in, want := range map[int]int{0: 0, 80: 0, 1023: 0, 1024: 1024, 8090: 8090, 65535: 65535, 65536: 0, -5: 0} {
		if got := sanitizePort(in); got != want {
			t.Errorf("sanitizePort(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestConformanceServerServesItsSpecAndEnvironment(t *testing.T) {
	h := newConformanceMux()
	for path, want := range map[string]string{"/openapi.yaml": "Clear Signal Conformance Server", "/environment.yaml": "name: id"} {
		code, _, body := doDemo(t, h, demoCall{method: "GET", path: path})
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("%s: %d, missing %q", path, code, want)
		}
	}
}
