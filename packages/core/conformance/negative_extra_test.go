package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const probeSpecYAML = `
openapi: "3.0.3"
info: {title: Probe API, version: "2"}
security:
  - key: []
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
paths:
  /items:
    get:
      operationId: listItems
      responses: {"200": {description: ok}}
  /items/{id}:
    get:
      operationId: getItem
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses: {"200": {description: ok}}
`

func loadProbeSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	s, err := openapi.ParseBytes([]byte(probeSpecYAML), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNegativeRunner_SkipsProbesWithUnresolvedPathParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer srv.Close()

	res := conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), loadProbeSpec(t), srv.URL, nil, conformance.Options{})
	passed, failed, skipped, errored := res.Counts()
	if skipped != 1 || passed != 1 || failed != 0 || errored != 0 {
		t.Fatalf("counts p=%d f=%d s=%d e=%d probes=%+v", passed, failed, skipped, errored, res.Probes)
	}
	for _, p := range res.Probes {
		if p.OperationID == "getItem" && !p.Skipped {
			t.Error("probe with {{id}} left in the URL must be skipped, not sent")
		}
	}
}

func TestNegativeRunner_ResolvesPathParamsFromEnvironment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer srv.Close()
	env := domain.NewEnvironment("e")
	env.Variables = []domain.EnvironmentVariable{{Name: "id", Value: "42"}}

	res := conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), loadProbeSpec(t), srv.URL, env, conformance.Options{})
	if _, _, skipped, _ := res.Counts(); skipped != 0 {
		t.Errorf("nothing should be skipped when the variable is defined: %+v", res.Probes)
	}
}

func TestNegativeRunner_RecordsTransportErrorsAndHonoursFailFast(t *testing.T) {
	spec := loadProbeSpec(t)
	env := domain.NewEnvironment("e")
	env.Variables = []domain.EnvironmentVariable{{Name: "id", Value: "1"}}
	r := conformance.NewNegative(executor.New(executor.Options{}))

	all := r.Run(context.Background(), spec, "http://127.0.0.1:1", env, conformance.Options{RequestTimeoutMs: 300})
	if _, _, _, errored := all.Counts(); errored != 2 {
		t.Fatalf("both probes should error, got %+v", all.Probes)
	}

	ff := r.Run(context.Background(), spec, "http://127.0.0.1:1", env, conformance.Options{RequestTimeoutMs: 300, FailFast: true})
	if len(ff.Probes) != 1 || ff.Probes[0].Error == "" {
		t.Errorf("fail-fast should stop after the first error: %+v", ff.Probes)
	}
}

func TestNegativeRunner_PerRequestTimeoutApplies(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-make(chan struct{}):
		}
	}))
	defer slow.Close()
	env := domain.NewEnvironment("e")
	env.Variables = []domain.EnvironmentVariable{{Name: "id", Value: "1"}}
	res := conformance.NewNegative(executor.New(executor.Options{})).Run(context.Background(), loadProbeSpec(t), slow.URL, env, conformance.Options{RequestTimeoutMs: 100})
	if _, _, _, errored := res.Counts(); errored == 0 {
		t.Errorf("slow server should hit the timeout: %+v", res.Probes)
	}
}
