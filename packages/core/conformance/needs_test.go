package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const needsSpec = `
openapi: "3.0.3"
info: {title: Needs, version: "1"}
paths:
  /health:
    get:
      tags: [system]
      responses: {"200": {description: ok}}
  /items/{id}:
    get:
      tags: [items]
      security: [{bearer: []}]
      parameters: [{name: id, in: path, required: true, schema: {type: string}}]
      responses: {"200": {description: ok}}
    delete:
      tags: [items]
      parameters: [{name: id, in: path, required: true, schema: {type: string}}]
      responses: {"204": {description: ok}}
  /session:
    get:
      tags: [system]
      security: [{cookie: []}]
      parameters: [{name: trace, in: query, required: true, schema: {type: string}}]
      responses: {"200": {description: ok}}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
    cookie: {type: apiKey, in: cookie, name: sid}
`

func parseNeedsSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(needsSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

func envOf(pairs ...string) *domain.Environment {
	env := &domain.Environment{}
	for i := 0; i < len(pairs); i += 2 {
		env.Variables = append(env.Variables, domain.EnvironmentVariable{Name: pairs[i], Value: pairs[i+1]})
	}
	return env
}

func needsOf(t *testing.T, l *conformance.NeedsList, op string) conformance.OperationNeeds {
	t.Helper()
	for _, o := range l.Operations {
		if o.Method+" "+o.SpecPath == op {
			return o
		}
	}
	t.Fatalf("%s is not in the list", op)
	return conformance.OperationNeeds{}
}

func TestListNeeds_NamesWhatEachOperationTakes(t *testing.T) {
	list, err := conformance.ListNeeds(parseNeedsSpec(t), "http://x", nil, conformance.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if h := needsOf(t, list, "GET /health"); len(h.Needs) != 0 || h.Blocked != "" {
		t.Errorf("health needs nothing: %+v", h)
	}
	item := needsOf(t, list, "GET /items/{id}")
	if !slices.Contains(item.Needs, "id") || len(item.Needs) != 2 || len(item.Credentials) != 1 || item.Credentials[0] == "id" || len(item.Missing) != 2 || !strings.HasPrefix(item.Blocked, "not sent: no value for ") {
		t.Errorf("%+v", item)
	}
	session := needsOf(t, list, "GET /session")
	if !slices.Contains(session.Needs, "trace") || len(session.Credentials) != 1 || session.Credentials[0] == "trace" {
		t.Errorf("a cookie API key is a credential, a query parameter is not: %+v", session)
	}
}

// What the list calls missing is exactly what a run does not send, whatever the filters.
func TestListNeeds_AgreesWithARun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	spec := parseNeedsSpec(t)
	for name, c := range map[string]struct {
		env  *domain.Environment
		opts conformance.Options
	}{
		"nothing":        {nil, conformance.Options{}},
		"only the id":    {envOf("id", "1"), conformance.Options{}},
		"empty id":       {envOf("id", ""), conformance.Options{}},
		"read only":      {nil, conformance.Options{ReadOnly: true}},
		"by tag":         {nil, conformance.Options{Tags: []string{"system"}}},
		"by path":        {envOf("id", "1"), conformance.Options{Paths: []string{"/items"}}},
		"failed dynamic": {envOf("id", "{{$noSuchFunction}}"), conformance.Options{}},
	} {
		list, err := conformance.ListNeeds(spec, srv.URL, c.env, c.opts)
		if err != nil {
			t.Fatal(err)
		}
		run := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, c.env, c.opts)
		if len(run.Operations) != len(list.Operations) || run.FilterMatched != list.FilterMatched {
			t.Fatalf("%s: run %d operations (%d matched), list %d (%d)", name, len(run.Operations), run.FilterMatched, len(list.Operations), list.FilterMatched)
		}
		for i, op := range run.Operations {
			l := list.Operations[i]
			if op.Method != l.Method || op.SpecPath != l.SpecPath || op.Skipped != l.Skipped || op.Blocked != l.Blocked || !slices.Equal(op.Missing, l.Missing) {
				t.Errorf("%s: run %s %s skipped=%v blocked=%q; list %s %s skipped=%v blocked=%q", name, op.Method, op.SpecPath, op.Skipped, op.Blocked, l.Method, l.SpecPath, l.Skipped, l.Blocked)
			}
		}
	}
}

// Pins CS-GAP-013: outside the path and the credentials, a variable that is defined but empty
// counts as a value, so a required query parameter is sent empty. Change this with the catalog.
func TestRun_AnEmptyValueElsewhereIsSentAsEmpty_KnownGap(t *testing.T) {
	var mu sync.Mutex
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/session" {
			queries = append(queries, r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	spec := parseNeedsSpec(t)
	list, err := conformance.ListNeeds(spec, srv.URL, nil, conformance.Options{})
	if err != nil {
		t.Fatal(err)
	}
	env := envOf("trace", "", needsOf(t, list, "GET /session").Credentials[0], "key")
	conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, env, conformance.Options{ReadOnly: true, Paths: []string{"/session"}})
	if !slices.Equal(queries, []string{"trace="}) {
		t.Fatalf("an empty query value is no longer sent as empty (%v): update CS-GAP-013", queries)
	}
}
