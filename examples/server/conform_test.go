package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
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

// TestSpecDiff demonstrates breaking-change detection between two spec versions.
//
// Run with:
//
//	go test -v -run TestSpecDiff ./examples/server
//
// from the repository root to see a formatted diff report.
func TestSpecDiff(t *testing.T) {
	const v1 = `
openapi: "3.0.3"
info:
  title: Users API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      summary: List users
      responses:
        "200":
          description: A list of users
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  required: [id, name, email]
                  properties:
                    id:    { type: integer }
                    name:  { type: string }
                    email: { type: string }
    post:
      operationId: createUser
      summary: Create a user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, email]
              properties:
                name:  { type: string }
                email: { type: string }
      responses:
        "201":
          description: User created
`

	const v2 = `
openapi: "3.0.3"
info:
  title: Users API
  version: "2.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      summary: List users
      responses:
        "200":
          description: A list of users
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  required: [id, name, email]
                  properties:
                    id:    { type: integer }
                    name:  { type: string }
                    email: { type: string }
  /users/{id}:
    get:
      operationId: getUser
      summary: Get a user by ID
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer }
      responses:
        "200":
          description: The requested user
          content:
            application/json:
              schema:
                type: object
                required: [id, name, email]
                properties:
                  id:    { type: integer }
                  name:  { type: string }
                  email: { type: string }
        "404":
          description: User not found
`

	base, err := openapi.ParseBytes([]byte(v1), "yaml")
	if err != nil {
		t.Fatalf("parsing base spec: %v", err)
	}
	revision, err := openapi.ParseBytes([]byte(v2), "yaml")
	if err != nil {
		t.Fatalf("parsing revision spec: %v", err)
	}

	result, err := openapi.Diff(base, revision)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	// Format and log a readable report.
	var sb strings.Builder
	fmt.Fprintf(&sb, "\nSpec diff: Users API v1.0.0 → v2.0.0\n")
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("─", 50))
	for _, c := range result.Changes {
		marker := "  "
		if c.Breaking {
			marker = "✗ "
		} else {
			marker = "✓ "
		}
		fmt.Fprintf(&sb, "%s[%s] %s\n", marker, c.Level, c.Text)
	}
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("─", 50))
	if result.HasBreaking {
		fmt.Fprintf(&sb, "Result: BREAKING CHANGES DETECTED (%d total changes)\n", len(result.Changes))
	} else {
		fmt.Fprintf(&sb, "Result: no breaking changes (%d total changes)\n", len(result.Changes))
	}
	t.Log(sb.String())

	if !result.HasBreaking {
		t.Error("expected breaking changes (POST /users was removed)")
	}

	var foundRemoved bool
	for _, c := range result.Changes {
		if c.Breaking && strings.Contains(strings.ToLower(c.Text), "removed") {
			foundRemoved = true
			break
		}
	}
	if !foundRemoved {
		t.Error("expected a breaking change describing removal of POST /users")
	}
}
