package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

const probeSpec = `openapi: "3.1.0"
info: {title: p, version: "1"}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
paths:
  /orders/{orderId}:
    parameters:
      - {name: orderId, in: path, required: true, schema: {type: string}}
    put:
      operationId: putOrder
      tags: [orders]
      parameters:
        - {name: If-Match, in: header, required: true, schema: {type: string}, example: v1}
      responses: {"200": {description: ok}, "428": {description: precondition required}}
  /payments:
    post:
      operationId: createPayment
      tags: [payments]
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - {type: object, required: [kind, cardLast4], properties: {kind: {type: string}, cardLast4: {type: string}}}
                - {type: object, required: [kind, iban], properties: {kind: {type: string}, iban: {type: string}}}
      responses: {"201": {description: ok}}
  /public:
    get:
      operationId: getPublic
      security: [{}, {bearerAuth: []}]
      responses: {"200": {description: ok}}
`

func probeDescriptions(t *testing.T) []string {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(probeSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	probes, err := openapi.GenerateNegativeProbes(spec, "http://x")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range probes {
		out = append(out, p.Description)
	}
	return out
}

// F-006: a required header is dropped in a probe.
func TestProbes_MissingRequiredHeader(t *testing.T) {
	if !strings.Contains(strings.Join(probeDescriptions(t), "\n"), "PUT /orders/{orderId} — missing required header 'If-Match'") {
		t.Errorf("no probe for the required header: %v", probeDescriptions(t))
	}
}

// F-004: each oneOf branch is probed with its own required fields.
func TestProbes_EveryOneOfBranchIsProbed(t *testing.T) {
	all := strings.Join(probeDescriptions(t), "\n")
	for _, want := range []string{"missing required field 'cardLast4' of branch 1", "missing required field 'iban' of branch 2"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing probe %q in:\n%s", want, all)
		}
	}
}

// Optional security (an empty alternative) is not probed with "no credentials".
func TestProbes_OptionalSecurityHasNoMissingAuthProbe(t *testing.T) {
	if strings.Contains(strings.Join(probeDescriptions(t), "\n"), "GET /public — no credentials") {
		t.Error("an operation whose authentication is optional must not get a no-credentials probe")
	}
}

func rejectingServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) }))
}

// F-013: a path or tag filter limits the probes, not only the operations.
func TestNegativeRunner_FiltersLimitTheProbes(t *testing.T) {
	srv := rejectingServer()
	defer srv.Close()
	spec, _ := openapi.ParseBytes([]byte(probeSpec), "yaml")
	run := func(opts conformance.Options) *conformance.NegativeResult {
		return conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, nil, opts)
	}
	all := run(conformance.Options{})
	byPath := run(conformance.Options{Paths: []string{"/orders"}})
	byTag := run(conformance.Options{Tags: []string{"payments"}})
	if len(byPath.Probes) == 0 || len(byPath.Probes) >= len(all.Probes) {
		t.Fatalf("path filter kept %d of %d probes", len(byPath.Probes), len(all.Probes))
	}
	for _, p := range byPath.Probes {
		if !strings.HasPrefix(p.Path, "/orders") {
			t.Errorf("probe for %s survived the /orders filter", p.Path)
		}
	}
	for _, p := range byTag.Probes {
		if p.Path != "/payments" {
			t.Errorf("probe for %s survived the payments tag filter", p.Path)
		}
	}
	if len(byTag.Probes) == 0 {
		t.Error("tag filter removed every probe")
	}
}

// F-014: probes are in the JSON and JUnit reports, and a failed probe fails them.
func TestReports_CarryTheNegativeProbes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	spec, _ := openapi.ParseBytes([]byte(probeSpec), "yaml")
	exec := executor.New(executor.Options{})
	res := conformance.New(exec).Run(context.Background(), spec, srv.URL, nil, conformance.Options{Tags: []string{"none"}})
	neg := conformance.NewNegative(exec).Run(context.Background(), spec, srv.URL, nil, conformance.Options{})
	if _, failed, _, _ := neg.Counts(); failed == 0 {
		t.Fatal("test setup: a server that accepts everything must fail probes")
	}

	js, err := reporting.ConformJSON(res, nil, neg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `"probes"`) || !strings.Contains(string(js), "missing required header 'If-Match'") {
		t.Errorf("JSON has no probes:\n%s", js)
	}
	if !strings.Contains(string(js), `"passed": false`) {
		t.Errorf("a failed probe must make the JSON report not passed:\n%s", js)
	}
	x, err := reporting.ConformJUnit(res, neg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(x), "probe: PUT /orders/{orderId}") || strings.Contains(string(x), `failures="0"`) {
		t.Errorf("JUnit has no probe cases or hides their failures:\n%s", x)
	}
	plain, _ := reporting.ConformJSON(res, nil, nil)
	if strings.Contains(string(plain), `"probes"`) {
		t.Error("no probes expected without --negative")
	}
}
