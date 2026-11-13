package main

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// TestScenarioCoverage runs the user-lifecycle scenario against an in-process
// httptest server and reports which spec operations were exercised.
//
// Run with:
//
//	go test -v -run TestScenarioCoverage ./examples/server
//
// from the repository root to see the full scenario output and coverage summary.
func TestScenarioCoverage(t *testing.T) {
	srv := httptest.NewServer(newMux())
	t.Cleanup(srv.Close)

	specData, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	spec, err := openapi.ParseBytes(specData, "yaml")
	if err != nil {
		t.Fatalf("parsing spec: %v", err)
	}

	scenarioData, err := os.ReadFile("../scenarios/user-lifecycle.yaml")
	if err != nil {
		t.Fatalf("reading scenario: %v", err)
	}
	sc, err := serialization.ParseScenarioFile(scenarioData, "../scenarios/user-lifecycle.yaml")
	if err != nil {
		t.Fatalf("parsing scenario: %v", err)
	}

	env := &domain.Environment{
		Variables: []domain.EnvironmentVariable{
			{Name: "baseUrl", Value: srv.URL},
			{Name: "username", Value: "admin"},
			{Name: "password", Value: "secret"},
		},
	}

	exec := executor.New(executor.Options{})
	runner := scenario.NewWithSpec(exec, spec)
	result := runner.Run(context.Background(), sc, env)

	var executed []openapi.ExecutedRequest
	for _, sr := range result.Steps {
		if sr.Request != nil && sr.Response != nil {
			executed = append(executed, openapi.ExecutedRequest{
				Method: string(sr.Request.Method),
				URL:    sr.Request.URL,
			})
		}
	}
	cov := spec.ScenarioCoverage(executed)

	t.Log("\n" + reporting.TextWithCoverage(result, cov))

	if !result.Passed {
		t.Error("scenario failed — see output above for details")
	}
	if cov == nil {
		t.Fatal("ScenarioCoverage returned nil")
	}
	if cov.Covered == 0 {
		t.Error("no spec operations were covered")
	}
	// The user-lifecycle scenario exercises: POST /auth/login, POST /users,
	// GET /users/{id}, DELETE /users/{id} — expect exactly 4 of 8.
	if cov.Covered != 4 {
		t.Errorf("covered operations: got %d, want 4", cov.Covered)
	}
	if cov.Total != 8 {
		t.Errorf("total spec operations: got %d, want 8", cov.Total)
	}
}
