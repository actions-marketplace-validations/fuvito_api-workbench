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

const emptySpec = `
openapi: "3.0.3"
info: {title: Empty, version: "1"}
paths:
  /files/{name}.json:
    get:
      parameters: [{name: name, in: path, required: true, schema: {type: string}}]
      responses: {"200": {description: ok}}
  /bearer:
    get: {security: [{bearer: []}], responses: {"200": {description: ok}}}
  /basic:
    get: {security: [{basic: []}], responses: {"200": {description: ok}}}
  /key-header:
    get: {security: [{keyHeader: []}], responses: {"200": {description: ok}}}
  /key-query:
    get: {security: [{keyQuery: []}], responses: {"200": {description: ok}}}
  /key-cookie:
    get: {security: [{keyCookie: []}], responses: {"200": {description: ok}}}
  /optional:
    get:
      parameters:
        - {name: q, in: query, required: true, schema: {type: string}}
        - {name: X-Trace, in: header, required: true, schema: {type: string}}
        - {name: pref, in: cookie, required: true, schema: {type: string}}
      responses: {"200": {description: ok}}
  /items/{id}:
    put:
      security: [{bearer: []}]
      parameters: [{name: id, in: path, required: true, schema: {type: string}}]
      requestBody:
        required: true
        content: {application/json: {schema: {type: object, required: [title], properties: {title: {type: string}}}}}
      responses: {"200": {description: ok}}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
    basic: {type: http, scheme: basic}
    keyHeader: {type: apiKey, in: header, name: X-Key}
    keyQuery: {type: apiKey, in: query, name: key}
    keyCookie: {type: apiKey, in: cookie, name: sid}
`

// sentTo runs against a server that records what arrives.
func sentTo(t *testing.T) (*openapi.Spec, string, func() []string) {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(emptySpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return spec, srv.URL, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(paths)
	}
}

// allEmpty defines every value the spec's requests take, each as an empty text.
func allEmpty(t *testing.T, spec *openapi.Spec, url string) *domain.Environment {
	t.Helper()
	list, err := conformance.ListNeeds(spec, url, nil, conformance.Options{})
	if err != nil {
		t.Fatal(err)
	}
	names, _ := list.Names()
	env := &domain.Environment{}
	for _, n := range names {
		env.Variables = append(env.Variables, domain.EnvironmentVariable{Name: n, Value: ""})
	}
	return env
}

// An empty value is no value in a path or in a credential, where empty is never what was meant: the
// request would ask for something else (/items/) or arrive unauthenticated, and a declared 400,
// 401 or 404 would show as passing. Elsewhere, and as a basic password, empty can be meant.
func TestRun_AnEmptyPathValueOrCredentialIsNotSent(t *testing.T) {
	spec, url, sent := sentTo(t)
	result := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, url, allEmpty(t, spec, url), conformance.Options{})

	want := map[string][]string{
		"GET /files/{name}.json": {"name"},
		"GET /bearer":            {"token"},
		"GET /basic":             {"basicUser"}, // not basicPass: APIs that take a key as the user name leave it empty
		"GET /key-header":        {"apiKey"},
		"GET /key-query":         {"apiKey"},
		"GET /key-cookie":        {"apiKey"},
		"GET /optional":          nil, // a query, header or cookie parameter may be meant to be empty
		"PUT /items/{id}":        {"id", "token"},
	}
	for _, op := range result.Operations {
		name := op.Method + " " + op.SpecPath
		missing := slices.Clone(op.Missing)
		slices.Sort(missing)
		if name == "PUT /items/{id}" {
			// the body's {{title}} may be empty
			if slices.Contains(missing, "title") {
				t.Errorf("%s: an empty body value is sent", name)
			}
		}
		if !slices.Equal(missing, want[name]) {
			t.Errorf("%s: missing %v, want %v (%s)", name, missing, want[name], op.Blocked)
		}
		if len(want[name]) > 0 && (!strings.Contains(op.Blocked, "not sent: no value for {{") || !strings.Contains(op.Blocked, "empty")) {
			t.Errorf("%s: the reason says the value is empty: %q", name, op.Blocked)
		}
	}
	if got := sent(); !slices.Equal(got, []string{"GET /optional"}) {
		t.Errorf("requests that reached the server: %v", got)
	}
}

// An unset CI secret commonly arrives as a process variable that is set and empty.
func TestRun_AnEmptyProcessVariableBehindACredentialIsNotSent(t *testing.T) {
	t.Setenv("CLRSIG_TEST_TOKEN", "")
	spec, url, sent := sentTo(t)
	result := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, url, envOf("token", "${CLRSIG_TEST_TOKEN}"), conformance.Options{Paths: []string{"/bearer"}})
	for _, op := range result.Operations {
		if op.SpecPath == "/bearer" && !slices.Equal(op.Missing, []string{"token"}) {
			t.Errorf("%+v", op)
		}
	}
	if len(sent()) != 0 {
		t.Errorf("sent: %v", sent())
	}
}

// Probes are built from the same requests: one with an empty path value or credential would be
// refused for that, and count as the server rejecting what the probe tests.
func TestNegativeRunner_SkipsProbesWithAnEmptyPathValueOrCredential(t *testing.T) {
	spec, url, sent := sentTo(t)
	result := conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), spec, url, allEmpty(t, spec, url), conformance.Options{Paths: []string{"/items"}})
	if len(result.Probes) == 0 {
		t.Fatal("no probes")
	}
	for _, p := range result.Probes {
		if !p.Skipped {
			t.Errorf("%s was sent with an empty id", p.Description)
		}
	}
	if len(sent()) != 0 {
		t.Errorf("sent: %v", sent())
	}
}
