package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

// Pins CS-GAP-014: --paths filters operations on the generated path (/items/{{id}}) and negative
// probes on the spec path (/items/{id}), so a prefix with a parameter selects only one side. Change
// this test and the catalog together when both match the path as the spec writes it.
func TestPathsFilter_WithAParameterSelectsOnlyOneSide_KnownGap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	spec := parseNeedsSpec(t) // /items/{id} has a GET with security, so it has a probe
	count := func(prefix string) (operations, probes int) {
		opts := conformance.Options{Paths: []string{prefix}}
		run := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, envOf("id", "1"), opts)
		neg := conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, envOf("id", "1"), opts)
		return run.FilterMatched, len(neg.Probes)
	}
	if ops, probes := count("/items/{id}"); ops != 0 || probes == 0 {
		t.Errorf("spec spelling: %d operations, %d probes; the gap has changed: update CS-GAP-014", ops, probes)
	}
	if ops, probes := count("/items/{{id}}"); ops == 0 || probes != 0 {
		t.Errorf("generated spelling: %d operations, %d probes; the gap has changed: update CS-GAP-014", ops, probes)
	}
	if ops, probes := count("/items/"); ops == 0 || probes == 0 {
		t.Errorf("a prefix without a parameter selects both: %d operations, %d probes", ops, probes)
	}
}
