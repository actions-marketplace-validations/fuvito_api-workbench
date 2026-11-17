package scenario_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

const conformSpec = `
openapi: "3.0.3"
info: {title: t, version: "1"}
paths:
  /users/1:
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id: {type: integer}
`

func jsonServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Token", "tok-1")
		fmt.Fprint(w, body)
	}))
}

func getStep(label, url string, opts ...func(*domain.ScenarioStep)) domain.ScenarioStep {
	s := domain.ScenarioStep{Label: label, Request: &domain.Request{Method: domain.MethodGET, URL: url}}
	for _, o := range opts {
		o(&s)
	}
	return s
}

func TestRunner_SpecConformance(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(conformSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	r := scenario.NewWithSpec(executor.New(executor.Options{}), spec)

	bad := jsonServer(`{"name":"no id"}`)
	defer bad.Close()
	good := jsonServer(`{"id":1}`)
	defer good.Close()
	off := false

	s := &domain.Scenario{Steps: []domain.ScenarioStep{
		getStep("violates", bad.URL+"/users/1", func(s *domain.ScenarioStep) { s.FailPolicy = domain.FailPolicyContinue }),
		getStep("opted out", bad.URL+"/users/1", func(s *domain.ScenarioStep) { s.ValidateSpec = &off }),
	}}
	res := r.Run(context.Background(), s, nil)

	if res.Passed {
		t.Fatal("scenario should fail on a spec violation")
	}
	st := res.Steps[0]
	if st.Passed || len(st.Assertions) != 1 || st.Assertions[0].Assertion.Name != "spec conformance" {
		t.Errorf("expected one spec conformance failure, got %+v", st.Assertions)
	}
	if !res.Steps[1].Passed || len(res.Steps[1].Assertions) != 0 {
		t.Errorf("opted-out step should pass without validation: %+v", res.Steps[1])
	}

	ok := r.Run(context.Background(), &domain.Scenario{Steps: []domain.ScenarioStep{
		getStep("conforms", good.URL+"/users/1", func(s *domain.ScenarioStep) {
			s.Assertions = []domain.Assertion{{Type: domain.AssertionSchema, Schema: "getUser"}}
		}),
	}}, nil)
	if !ok.Passed {
		t.Errorf("conforming response should pass: %+v", ok.Steps[0])
	}

	bare := r.Run(context.Background(), &domain.Scenario{Steps: []domain.ScenarioStep{getStep("bare", good.URL)}}, nil)
	if len(bare.Steps) != 1 {
		t.Fatal("expected one step")
	}
}

func TestRunner_Extractions(t *testing.T) {
	srv := jsonServer(`{"id":7}`)
	defer srv.Close()

	var seenAuth string
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("X-Auth")
	}))
	defer echo.Close()

	s := &domain.Scenario{Steps: []domain.ScenarioStep{
		getStep("extract", srv.URL, func(s *domain.ScenarioStep) {
			s.Extractions = []domain.Extraction{
				{Name: "code", Source: domain.ExtractStatus},
				{Name: "tok", Source: domain.ExtractHeader, Path: "x-token"},
				{Name: "id", Source: domain.ExtractJSONPath, Path: "$.id"},
			}
		}),
		{Label: "use", Request: &domain.Request{
			Method: domain.MethodGET, URL: echo.URL,
			Headers: []domain.Header{{Name: "X-Auth", Value: "{{tok}}-{{id}}-{{code}}"}},
		}},
	}}
	res := newRunner().Run(context.Background(), s, nil)
	if !res.Passed {
		t.Fatalf("unexpected failure: %+v", res.Steps)
	}
	if got := res.Steps[0].Extracted; got["code"] != "200" || got["tok"] != "tok-1" || got["id"] != "7" {
		t.Errorf("extracted: %v", got)
	}
	if seenAuth != "tok-1-7-200" {
		t.Errorf("chained variables not applied: %q", seenAuth)
	}
}

func TestRunner_ExtractionErrors(t *testing.T) {
	srv := jsonServer(`{"id":7}`)
	defer srv.Close()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer empty.Close()
	text := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "plain") }))
	defer text.Close()

	tests := []struct {
		name string
		url  string
		ex   domain.Extraction
		want string
	}{
		{"missing header", srv.URL, domain.Extraction{Name: "x", Source: domain.ExtractHeader, Path: "X-Nope"}, "not found"},
		{"empty body", empty.URL, domain.Extraction{Name: "x", Source: domain.ExtractJSONPath, Path: "$.id"}, "empty"},
		{"non-json body", text.URL, domain.Extraction{Name: "x", Source: domain.ExtractJSONPath, Path: "$.id"}, "not valid JSON"},
		{"bad path", srv.URL, domain.Extraction{Name: "x", Source: domain.ExtractJSONPath, Path: "$.missing"}, "jsonpath"},
		{"unknown source", srv.URL, domain.Extraction{Name: "x", Source: "bogus"}, "unknown extraction source"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &domain.Scenario{Steps: []domain.ScenarioStep{
				getStep("s", tc.url, func(s *domain.ScenarioStep) { s.Extractions = []domain.Extraction{tc.ex} }),
			}}
			res := newRunner().Run(context.Background(), s, nil)
			if e := res.Steps[0].Error; !strings.Contains(e, tc.want) {
				t.Errorf("error %q should contain %q", e, tc.want)
			}
			if _, failed, _ := res.Counts(); failed != 1 {
				t.Errorf("a step with an extraction error counts as failed")
			}
		})
	}
}

func TestRunner_StepFlowControl(t *testing.T) {
	ok := jsonServer(`{}`)
	defer ok.Close()
	failing := func(s *domain.ScenarioStep) {
		s.Assertions = []domain.Assertion{{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "500"}}
	}

	t.Run("RunFrom and disabled", func(t *testing.T) {
		s := &domain.Scenario{Steps: []domain.ScenarioStep{
			getStep("a", ok.URL),
			getStep("b", ok.URL, func(s *domain.ScenarioStep) { s.Disabled = true }),
			getStep("c", ok.URL),
		}}
		res := newRunner().RunFrom(context.Background(), s, nil, 1)
		p, f, sk := res.Counts()
		if p != 1 || f != 0 || sk != 2 || !res.Steps[0].Skipped || !res.Steps[1].Skipped {
			t.Errorf("counts %d/%d/%d steps=%+v", p, f, sk, res.Steps)
		}
		if !res.AllPassed() {
			t.Error("skips are not failures")
		}
	})

	t.Run("skip policy skips after failure", func(t *testing.T) {
		s := &domain.Scenario{Steps: []domain.ScenarioStep{
			getStep("fail", ok.URL, failing, func(s *domain.ScenarioStep) { s.FailPolicy = domain.FailPolicyContinue }),
			getStep("skipme", ok.URL, func(s *domain.ScenarioStep) { s.FailPolicy = domain.FailPolicySkip }),
			getStep("after", ok.URL),
		}}
		res := newRunner().Run(context.Background(), s, nil)
		if res.Passed || !res.Steps[1].Skipped || !res.Steps[2].Passed {
			t.Errorf("unexpected flow: %+v", res.Steps)
		}
	})

	t.Run("stop policy skips the rest", func(t *testing.T) {
		s := &domain.Scenario{Steps: []domain.ScenarioStep{
			getStep("fail", ok.URL, failing),
			getStep("never", ok.URL),
		}}
		res := newRunner().Run(context.Background(), s, nil)
		if !res.Steps[1].Skipped {
			t.Error("remaining step should be skipped")
		}
		ap, af := res.AssertionCounts()
		if ap != 0 || af != 1 {
			t.Errorf("assertion counts %d/%d", ap, af)
		}
	})

	t.Run("missing request and transport error", func(t *testing.T) {
		s := &domain.Scenario{Steps: []domain.ScenarioStep{
			{Label: "norequest", RequestRef: "x.yaml", FailPolicy: domain.FailPolicyContinue},
			getStep("unreachable", "http://127.0.0.1:1"),
		}}
		res := newRunner().Run(context.Background(), s, nil)
		if !strings.Contains(res.Steps[0].Error, "no request") || res.Steps[1].Error == "" || res.Passed {
			t.Errorf("unexpected: %+v", res.Steps)
		}
	})
}
