package scenario_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func newRunner() *scenario.Runner {
	return scenario.New(executor.New(executor.Options{}))
}

func TestRun_AllPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	s := &domain.Scenario{
		Name: "Simple pass",
		Steps: []domain.ScenarioStep{
			{
				Name:    "Health check",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL + "/health"},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
					{Type: domain.AssertionJSONPath, Path: "$.status", Operator: domain.OperatorEq, Expected: "ok"},
				},
			},
		},
	}

	result := newRunner().Run(context.Background(), s, nil)
	if !result.Passed {
		t.Errorf("expected passed; step error: %s", result.Steps[0].Error)
	}
}

func TestRun_VariableExtraction(t *testing.T) {
	var capturedID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(201)
			fmt.Fprint(w, `{"id":"42","name":"Alice"}`)
			return
		}
		// GET /users/42
		if r.URL.Path == "/users/42" {
			capturedID = "42"
			w.WriteHeader(200)
			fmt.Fprint(w, `{"id":"42","name":"Alice"}`)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := &domain.Scenario{
		Name: "Create then fetch",
		Steps: []domain.ScenarioStep{
			{
				Name:    "Create user",
				Request: &domain.Request{Method: domain.MethodPOST, URL: srv.URL + "/users"},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "201"},
				},
				Extractions: []domain.Extraction{
					{Name: "userId", Source: domain.ExtractJSONPath, Path: "$.id"},
				},
			},
			{
				Name:    "Get user",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL + "/users/{{userId}}"},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
					{Type: domain.AssertionJSONPath, Path: "$.name", Operator: domain.OperatorEq, Expected: "Alice"},
				},
			},
		},
	}

	result := newRunner().Run(context.Background(), s, nil)
	if !result.Passed {
		for i, sr := range result.Steps {
			t.Logf("step %d %q: passed=%v error=%q", i, sr.Name, sr.Passed, sr.Error)
			for _, ar := range sr.Assertions {
				t.Logf("  assertion: passed=%v msg=%q", ar.Passed, ar.Message)
			}
		}
		t.Fail()
	}
	if capturedID != "42" {
		t.Errorf("variable extraction did not propagate: capturedID=%q", capturedID)
	}
}

func TestRun_FailPolicyStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := &domain.Scenario{
		Name: "Fail stop",
		Steps: []domain.ScenarioStep{
			{
				Name:    "Failing step",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
				},
				FailPolicy: domain.FailPolicyStop,
			},
			{
				Name:    "Should be skipped",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
			},
		},
	}

	result := newRunner().Run(context.Background(), s, nil)
	if result.Passed {
		t.Error("expected failed result")
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(result.Steps))
	}
	if !result.Steps[1].Skipped {
		t.Error("expected second step to be skipped")
	}
}

func TestRun_WithEnvironment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Token") == "secret" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(401)
	}))
	defer srv.Close()

	env := domain.NewEnvironment("test")
	env.Variables = []domain.EnvironmentVariable{
		{Name: "token", Value: "secret"},
		{Name: "baseUrl", Value: srv.URL},
	}

	s := &domain.Scenario{
		Name: "With env",
		Steps: []domain.ScenarioStep{
			{
				Name: "Auth request",
				Request: &domain.Request{
					Method:  domain.MethodGET,
					URL:     "{{baseUrl}}/protected",
					Headers: []domain.Header{{Name: "X-Token", Value: "{{token}}"}},
				},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
				},
			},
		},
	}

	result := newRunner().Run(context.Background(), s, env)
	if !result.Passed {
		t.Errorf("expected pass with env; msg: %v", result.Steps[0].Error)
	}
}

func TestRun_HeaderExtraction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-abc-123")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	var capturedReqID string
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = r.Header.Get("X-Trace")
		w.WriteHeader(200)
	}))
	defer srv2.Close()

	s := &domain.Scenario{
		Name: "Header extraction",
		Steps: []domain.ScenarioStep{
			{
				Name:    "Step 1",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
				Extractions: []domain.Extraction{
					{Name: "reqID", Source: domain.ExtractHeader, Path: "X-Request-ID"},
				},
			},
			{
				Name: "Step 2",
				Request: &domain.Request{
					Method:  domain.MethodGET,
					URL:     srv2.URL,
					Headers: []domain.Header{{Name: "X-Trace", Value: "{{reqID}}"}},
				},
			},
		},
	}

	result := newRunner().Run(context.Background(), s, nil)
	if !result.Passed {
		t.Errorf("unexpected failure: %v", result.Steps[0].Error)
	}
	if capturedReqID != "req-abc-123" {
		t.Errorf("header extraction not propagated: got %q", capturedReqID)
	}
}

func TestResult_Counts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	s := &domain.Scenario{
		Name: "Counts",
		Steps: []domain.ScenarioStep{
			{
				Name:    "Pass step",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
				},
				FailPolicy: domain.FailPolicyContinue,
			},
			{
				Name:    "Fail step",
				Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "201"},
				},
				FailPolicy: domain.FailPolicyContinue,
			},
		},
	}

	result := newRunner().Run(context.Background(), s, nil)
	passed, failed, skipped := result.Counts()
	if passed != 1 || failed != 1 || skipped != 0 {
		t.Errorf("counts: passed=%d failed=%d skipped=%d", passed, failed, skipped)
	}
}
