package scenario_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

const redirectSpec = `openapi: 3.0.3
info: {title: R, version: "1"}
paths:
  /old:
    get:
      operationId: oldPath
      responses:
        "302":
          description: moved
          headers:
            Location: {schema: {type: string}}
`

func TestRedirectDeclaredInTheSpecCanBeValidated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusFound)
			return
		}
		w.Write([]byte("final"))
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(redirectSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	step := func(label string, noFollow bool) domain.ScenarioStep {
		return domain.ScenarioStep{Label: label, FailPolicy: domain.FailPolicyContinue,
			Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL + "/old", NoFollowRedirects: noFollow}}
	}
	res := scenario.NewWithSpec(executor.New(executor.Options{}), spec).Run(context.Background(),
		&domain.Scenario{Steps: []domain.ScenarioStep{step("followed", false), step("not followed", true)}}, nil)

	if res.Steps[0].Response.StatusCode != 200 {
		t.Errorf("when redirects are followed the client only sees the final 200, got %d", res.Steps[0].Response.StatusCode)
	}
	if !res.Steps[1].Passed || res.Steps[1].Response.StatusCode != 302 {
		t.Errorf("the declared 302 is returned and validates: %+v / %v", res.Steps[1], res.Steps[1].Error)
	}
}
