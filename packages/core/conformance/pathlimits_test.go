package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const pathLimitSpec = `openapi: "3.0.3"
info: {title: s, version: "1"}
paths:
  /fast:
    get: {responses: {"200": {description: ok}}}
  /reports:
    get: {responses: {"200": {description: ok}}}
  /reports/export:
    get: {responses: {"200": {description: ok}}}
`

func slowOps(t *testing.T, opts conformance.Options) map[string]bool {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(pathLimitSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	res := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, domain.NewEnvironment("e"), opts)
	out := map[string]bool{}
	for _, op := range res.Operations {
		out[op.Path] = !op.Passed()
	}
	return out
}

// A limit for a path prefix replaces the general limit there; the longest prefix wins; 0 turns the limit off.
func TestMaxResponseTimeFor_LongestPrefixWins(t *testing.T) {
	got := slowOps(t, conformance.Options{
		MaxResponseTimeMs: 20,
		PathLimits:        map[string]int{"/reports": 5000, "/reports/export": 20},
	})
	want := map[string]bool{"/fast": true, "/reports": false, "/reports/export": true}
	for p, w := range want {
		if got[p] != w {
			t.Errorf("%s failed=%v, want %v", p, got[p], w)
		}
	}
	off := slowOps(t, conformance.Options{MaxResponseTimeMs: 20, PathLimits: map[string]int{"/reports": 0}})
	if off["/reports"] || !off["/fast"] {
		t.Errorf("0 under a prefix switches the limit off there only: %v", off)
	}
	only := slowOps(t, conformance.Options{PathLimits: map[string]int{"/fast": 20}})
	if !only["/fast"] || only["/reports"] {
		t.Errorf("a path limit works without a general limit: %v", only)
	}
}

func TestParsePathLimits(t *testing.T) {
	m, err := conformance.ParsePathLimits([]string{"/reports=2000", "/a/b=0"})
	if err != nil || m["/reports"] != 2000 || m["/a/b"] != 0 || len(m) != 2 {
		t.Fatalf("%v %v", m, err)
	}
	for _, bad := range []string{"reports=1", "/x", "/x=abc", "/x=-1", "=5"} {
		if _, err := conformance.ParsePathLimits([]string{bad}); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}
