package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const blockedSpec = `openapi: "3.0.3"
info: {title: blocked, version: "1"}
paths:
  /health:
    get:
      operationId: health
      responses: {"200": {description: ok}}
  /items/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: integer}}
    get:
      operationId: getItem
      responses:
        "200": {description: ok}
        "404": {description: not found}
  /secure:
    get:
      operationId: secure
      security: [{bearer: []}]
      responses:
        "200": {description: ok}
        "401": {description: unauthorized}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
`

func runBlocked(t *testing.T, env *domain.Environment) (*conformance.ConformResult, map[string]int) {
	t.Helper()
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		// 404 is a declared answer for /items/{id}: it must never be mistaken for a match.
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(blockedSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	res := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, env, conformance.Options{})
	return res, hits
}

func byID(res *conformance.ConformResult, id string) conformance.OperationResult {
	for _, op := range res.Operations {
		if op.OperationID == id {
			return op
		}
	}
	return conformance.OperationResult{}
}

func TestRunner_DoesNotSendARequestWithAMissingValue(t *testing.T) {
	res, hits := runBlocked(t, &domain.Environment{Name: "e", Variables: []domain.EnvironmentVariable{
		{Name: "unusedThing", Value: "x"}, // defined but never used: irrelevant
	}})

	item := byID(res, "getItem")
	if item.Blocked == "" || len(item.Missing) != 1 || item.Missing[0] != "id" || item.Passed() {
		t.Fatalf("getItem must be blocked on {{id}}: %+v", item)
	}
	if hits["/items/{{id}}"] != 0 || hits["/items/%7B%7Bid%7D%7D"] != 0 {
		t.Fatalf("a blocked request must not reach the server: %v", hits)
	}
	if len(hits) != 1 || hits["/health"] != 1 {
		t.Fatalf("only /health had everything it needs, hits = %v", hits)
	}
	secure := byID(res, "secure")
	if secure.Blocked == "" || secure.Missing[0] != "token" {
		t.Fatalf("a secured operation without a token must be blocked on {{token}}: %+v", secure)
	}
	if !strings.Contains(item.Blocked, "{{id}}") || !strings.HasPrefix(item.Blocked, "not sent") {
		t.Fatalf("reason = %q", item.Blocked)
	}

	passed, failed, skipped := res.Counts()
	if passed != 1 || failed != 2 || skipped != 0 {
		t.Fatalf("counts passed=%d failed=%d skipped=%d: a blocked operation counts as failed", passed, failed, skipped)
	}
	cov := conformance.ComputeCoverage(res)
	if cov.Covered != 1 || cov.NotCovered != 2 {
		t.Fatalf("a blocked operation is not covered: %+v", cov)
	}
}

func TestRunner_SendsOnceTheValuesAreThere(t *testing.T) {
	res, hits := runBlocked(t, &domain.Environment{Name: "e", Variables: []domain.EnvironmentVariable{
		{Name: "id", Value: "1"}, {Name: "token", Value: "abc-secret"},
	}})
	for _, op := range res.Operations {
		if op.Blocked != "" {
			t.Fatalf("%s should not be blocked: %+v", op.OperationID, op)
		}
	}
	if hits["/items/1"] != 1 || hits["/secure"] != 1 {
		t.Fatalf("hits = %v", hits)
	}
}

func TestRunner_ABlockedOperationStopsAFailFastRun(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(blockedSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	var sent atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sent.Add(1); w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	res := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, nil, conformance.Options{FailFast: true})
	if len(res.Operations) != 2 || res.Operations[0].OperationID != "health" || res.Operations[1].Blocked == "" {
		t.Fatalf("fail-fast must stop at the first blocked operation (after /health): %+v", res.Operations)
	}
	if sent.Load() != 1 {
		t.Fatalf("only /health was sent, sent = %d", sent.Load())
	}
}

func TestUnresolvedPlaceholders(t *testing.T) {
	req := &domain.Request{
		URL:         "http://x/items/{{id}}/{{ id }}",
		QueryParams: []domain.QueryParam{{Name: "q", Value: "{{search}}"}},
		Headers:     []domain.Header{{Name: "X-Trace", Value: "{{trace}}"}, {Name: "X-Fine", Value: "plain"}},
		Body:        &domain.RequestBody{Content: `{"name":"{{name}}","n":"{{$randomInt}}"}`},
		Auth:        &domain.Auth{Bearer: &domain.BearerAuth{Token: "{{token}}"}},
	}
	got := strings.Join(conformance.UnresolvedPlaceholders(req), ",")
	if got != "id,search,trace,name,token" {
		t.Fatalf("got %q (each name once, in order, dynamic functions left out)", got)
	}
	if n := conformance.UnresolvedPlaceholders(&domain.Request{URL: "http://x/ok", Body: &domain.RequestBody{Content: "{}"}}); len(n) != 0 {
		t.Fatalf("nothing to report: %v", n)
	}
}

func TestDiagnose_Blocked(t *testing.T) {
	f := conformance.Diagnose(conformance.OperationResult{Blocked: "not sent: no value for {{id}}, {{name}}", Missing: []string{"id", "name"}})
	if len(f) != 1 || f[0].Kind != conformance.FindingMissingValue {
		t.Fatalf("%+v", f)
	}
	if f[0].Summary != "No value for {{id}}, {{name}}" || !strings.Contains(f[0].Fix, "id=…, name=…") || !strings.Contains(f[0].Why, "not sent") {
		t.Fatalf("%+v", f[0])
	}
	dyn := conformance.Diagnose(conformance.OperationResult{Blocked: "not sent: unknown function $nope"})
	if len(dyn) != 1 || dyn[0].Kind != conformance.FindingMissingValue || !strings.Contains(dyn[0].Got, "unknown function") {
		t.Fatalf("%+v", dyn)
	}
}
