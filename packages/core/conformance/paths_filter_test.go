package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const filterSpec = `
openapi: "3.0.3"
info: {title: Filter, version: "1"}
paths:
  /items:
    get: {security: [{bearer: []}], responses: {"200": {description: ok}}}
  /items/{id}:
    get:
      security: [{bearer: []}]
      parameters: [{name: id, in: path, required: true, schema: {type: string}}]
      responses: {"200": {description: ok}}
  /items/{id}/parts/{part}:
    get:
      security: [{bearer: []}]
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
        - {name: part, in: path, required: true, schema: {type: string}}
      responses: {"200": {description: ok}}
  /other:
    get: {security: [{bearer: []}], responses: {"200": {description: ok}}}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
`

// --paths and the per-path limits take the path as the spec writes it ({id}) and, as before, as
// the generated request has it ({{id}}). Operations, probes and the --list listing agree.
func TestPathsFilter_BothSpellingsSelectTheSameOperationsAndProbes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(filterSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	env := envOf("id", "1", "part", "2", "token", "t")
	one := []string{"/items/{id}", "/items/{id}/parts/{part}"}
	all := []string{"/items", "/items/{id}", "/items/{id}/parts/{part}"}
	for prefix, want := range map[string][]string{
		"/items/{id}":                one,
		"/items/{{id}}":              one,
		"/items/{i":                  one, // an unfinished name still is the start of the spec's path
		"/items/{{i":                 one, // and of the generated one
		"/items/{id}/parts/{{part}}": {"/items/{id}/parts/{part}"},
		"/items/{{id}}/parts/{part}": {"/items/{id}/parts/{part}"},
		"/items":                     all,
		"/items/":                    one,
		"/items/{name}":              nil, // another parameter name is another path
		"/items/1":                   nil, // a value is not a path
		"/nowhere":                   nil,
	} {
		opts := conformance.Options{Paths: []string{prefix}}
		var ran, probed, listed []string
		for _, op := range conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, env, opts).Operations {
			if !op.Skipped {
				ran = append(ran, op.SpecPath)
			}
		}
		for _, p := range conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, env, opts).Probes {
			if !slices.Contains(probed, p.Path) {
				probed = append(probed, p.Path)
			}
		}
		list, err := conformance.ListNeeds(spec, srv.URL, env, opts)
		if err != nil {
			t.Fatal(err)
		}
		for _, op := range list.Operations {
			if !op.Skipped {
				listed = append(listed, op.SpecPath)
			}
		}
		for what, got := range map[string][]string{"operations": ran, "probes": probed, "list": listed} {
			slices.Sort(got)
			if !slices.Equal(got, want) {
				t.Errorf("--paths %s: %s %v, want %v", prefix, what, got, want)
			}
		}
	}
}

func TestPathLimits_BothSpellingsApplyAndTheLongestPathWins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(filterSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	env := envOf("id", "1", "part", "2", "token", "t")
	slow := func(limits map[string]int) []string {
		var out []string
		run := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, srv.URL, env, conformance.Options{PathLimits: limits})
		for _, op := range run.Operations {
			if len(op.Violations) > 0 {
				out = append(out, op.SpecPath)
			}
		}
		slices.Sort(out)
		return out
	}
	both := []string{"/items/{id}", "/items/{id}/parts/{part}"}
	for _, spelling := range []string{"/items/{id}", "/items/{{id}}"} {
		if got := slow(map[string]int{spelling: 1}); !slices.Equal(got, both) {
			t.Errorf("limit %s=1: %v, want %v", spelling, got, both)
		}
	}
	// The longer path wins however it is spelled: extra braces do not make a prefix longer.
	if got := slow(map[string]int{"/items/{{id}}": 1, "/items/{id}/p": 0}); !slices.Equal(got, []string{"/items/{id}"}) {
		t.Errorf("the longer path turns the limit off below it: %v", got)
	}
	// One path written both ways with two limits: the {{id}} one wins, as it did when it was the only
	// spelling that matched, and the answer is the same every time.
	for i := 0; i < 15; i++ {
		if got := slow(map[string]int{"/items/{id}": 0, "/items/{{id}}": 1}); !slices.Equal(got, both) {
			t.Fatalf("round %d: %v, want %v", i, got, both)
		}
		if got := slow(map[string]int{"/items/{id}": 1, "/items/{{id}}": 0}); len(got) != 0 {
			t.Fatalf("round %d: %v, want none", i, got)
		}
	}
}
