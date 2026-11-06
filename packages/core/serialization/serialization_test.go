package serialization_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

const yamlRequest = `
schemaVersion: "1"
title: Get User
method: GET
url: "{{baseUrl}}/api/users/{{userId}}"
headers:
  - name: Accept
    value: application/json
queryParams:
  - name: format
    value: json
auth:
  type: bearer
  token: "{{authToken}}"
timeout: 30s
`

const jsonRequest = `{
  "schemaVersion": "1",
  "title": "Create Item",
  "method": "POST",
  "url": "https://api.example.com/items",
  "body": {"type": "json", "content": "{\"name\": \"test\"}"}
}`

func TestParseRequest_YAML(t *testing.T) {
	req, err := serialization.ParseRequest([]byte(yamlRequest), "yaml")
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Title != "Get User" {
		t.Errorf("title: got %q", req.Title)
	}
	if req.Method != domain.MethodGET {
		t.Errorf("method: got %q", req.Method)
	}
	if req.URL != "{{baseUrl}}/api/users/{{userId}}" {
		t.Errorf("url: got %q", req.URL)
	}
	if len(req.Headers) != 1 || req.Headers[0].Name != "Accept" {
		t.Errorf("headers: got %+v", req.Headers)
	}
	if len(req.QueryParams) != 1 || req.QueryParams[0].Name != "format" {
		t.Errorf("queryParams: got %+v", req.QueryParams)
	}
	if req.Auth == nil || req.Auth.Type != domain.AuthBearer {
		t.Errorf("auth: got %+v", req.Auth)
	}
	if req.Timeout.Seconds() != 30 {
		t.Errorf("timeout: got %v", req.Timeout)
	}
}

func TestParseRequest_JSON(t *testing.T) {
	req, err := serialization.ParseRequest([]byte(jsonRequest), "json")
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Method != domain.MethodPOST {
		t.Errorf("method: got %q", req.Method)
	}
	if req.Body == nil || req.Body.Type != domain.BodyJSON {
		t.Errorf("body: got %+v", req.Body)
	}
}

func TestParseRequestFile_Extension(t *testing.T) {
	req, err := serialization.ParseRequestFile([]byte(yamlRequest), "get-user.yaml")
	if err != nil {
		t.Fatalf("ParseRequestFile: %v", err)
	}
	if req.Title != "Get User" {
		t.Errorf("title: got %q", req.Title)
	}
}

func TestWriteRequest_RoundTrip(t *testing.T) {
	original := &domain.Request{
		Title:  "Round Trip",
		Method: domain.MethodPUT,
		URL:    "https://example.com/test",
		Headers: []domain.Header{
			{Name: "X-Custom", Value: "value"},
		},
		Body: &domain.RequestBody{Type: domain.BodyJSON, Content: `{"ok":true}`},
		Auth: &domain.Auth{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuth{Token: "tok"},
		},
	}

	data, err := serialization.WriteRequest(original, "yaml")
	if err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}

	restored, err := serialization.ParseRequest(data, "yaml")
	if err != nil {
		t.Fatalf("ParseRequest after write: %v", err)
	}
	if restored.Method != original.Method {
		t.Errorf("method: %q != %q", restored.Method, original.Method)
	}
	if restored.URL != original.URL {
		t.Errorf("url: %q != %q", restored.URL, original.URL)
	}
	if len(restored.Headers) != 1 || restored.Headers[0].Name != "X-Custom" {
		t.Errorf("headers: %+v", restored.Headers)
	}
	if restored.Auth == nil || restored.Auth.Bearer == nil || restored.Auth.Bearer.Token != "tok" {
		t.Errorf("auth: %+v", restored.Auth)
	}
}

const yamlScenario = `
schemaVersion: "1"
title: User flow
description: Create and retrieve a user
steps:
  - title: Create user
    request:
      method: POST
      url: "{{baseUrl}}/users"
      body:
        type: json
        content: '{"name":"Alice"}'
    assertions:
      - type: status
        operator: eq
        expected: "201"
    extractions:
      - name: userId
        source: jsonpath
        path: "$.id"
  - title: Get user
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
    assertions:
      - type: status
        operator: eq
        expected: "200"
      - type: jsonpath
        path: "$.name"
        operator: eq
        expected: "Alice"
    failPolicy: continue
`

func TestParseScenario_YAML(t *testing.T) {
	s, err := serialization.ParseScenario([]byte(yamlScenario), "yaml")
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	if s.Title != "User flow" {
		t.Errorf("title: got %q", s.Title)
	}
	if s.Description != "Create and retrieve a user" {
		t.Errorf("description: got %q", s.Description)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps: got %d", len(s.Steps))
	}

	step0 := s.Steps[0]
	if step0.Title != "Create user" {
		t.Errorf("step0 title: %q", step0.Title)
	}
	if step0.Request == nil || step0.Request.Method != domain.MethodPOST {
		t.Errorf("step0 request method: %+v", step0.Request)
	}
	if len(step0.Assertions) != 1 || step0.Assertions[0].Type != domain.AssertionStatus {
		t.Errorf("step0 assertions: %+v", step0.Assertions)
	}
	if len(step0.Extractions) != 1 || step0.Extractions[0].Name != "userId" {
		t.Errorf("step0 extractions: %+v", step0.Extractions)
	}

	step1 := s.Steps[1]
	if step1.FailPolicy != domain.FailPolicyContinue {
		t.Errorf("step1 failPolicy: %q", step1.FailPolicy)
	}
	if len(step1.Assertions) != 2 {
		t.Errorf("step1 assertions count: %d", len(step1.Assertions))
	}
}

func TestWriteScenario_RoundTrip(t *testing.T) {
	original := &domain.Scenario{
		SchemaVersion: "1",
		Title:         "Round trip",
		Steps: []domain.ScenarioStep{
			{
				Title: "Step A",
				Request: &domain.Request{
					Method: domain.MethodGET,
					URL:    "https://example.com",
				},
				Assertions: []domain.Assertion{
					{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"},
				},
				FailPolicy: domain.FailPolicyContinue,
			},
		},
	}

	data, err := serialization.WriteScenario(original, "yaml")
	if err != nil {
		t.Fatalf("WriteScenario: %v", err)
	}

	restored, err := serialization.ParseScenario(data, "yaml")
	if err != nil {
		t.Fatalf("ParseScenario after write: %v", err)
	}
	if restored.Title != original.Title {
		t.Errorf("title: %q != %q", restored.Title, original.Title)
	}
	if len(restored.Steps) != 1 {
		t.Fatalf("steps: %d", len(restored.Steps))
	}
	if restored.Steps[0].FailPolicy != domain.FailPolicyContinue {
		t.Errorf("failPolicy: %q", restored.Steps[0].FailPolicy)
	}
	if restored.Steps[0].Request == nil || restored.Steps[0].Request.Method != domain.MethodGET {
		t.Errorf("request: %+v", restored.Steps[0].Request)
	}
}

func TestParseScenario_WithRequestRef(t *testing.T) {
	yaml := `
schemaVersion: "1"
name: Ref scenario
steps:
  - name: Reuse request
    requestRef: "requests/create-user.yaml"
    assertions:
      - type: status
        operator: eq
        expected: "201"
`
	s, err := serialization.ParseScenario([]byte(yaml), "yaml")
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	if s.Steps[0].RequestRef != "requests/create-user.yaml" {
		t.Errorf("requestRef: got %q", s.Steps[0].RequestRef)
	}
	if s.Steps[0].Request != nil {
		t.Errorf("expected no inline request when requestRef is set")
	}
}

const yamlEnv = `
schemaVersion: "1"
name: Local Dev
variables:
  - name: baseUrl
    value: http://localhost:8080
  - name: apiKey
    value: ""
    secret: true
`

func TestParseEnvironment_YAML(t *testing.T) {
	env, err := serialization.ParseEnvironment([]byte(yamlEnv), "yaml")
	if err != nil {
		t.Fatalf("ParseEnvironment: %v", err)
	}
	if env.Name != "Local Dev" {
		t.Errorf("name: got %q", env.Name)
	}
	if len(env.Variables) != 2 {
		t.Fatalf("variables: got %d", len(env.Variables))
	}
	if !env.Variables[1].Secret {
		t.Errorf("expected apiKey to be secret")
	}
}

func TestWriteEnvironment_RoundTrip(t *testing.T) {
	original := domain.NewEnvironment("Staging")
	original.Variables = []domain.EnvironmentVariable{
		{Name: "baseUrl", Value: "https://staging.example.com"},
		{Name: "token", Value: "", Secret: true},
	}

	data, err := serialization.WriteEnvironment(original, "yaml")
	if err != nil {
		t.Fatalf("WriteEnvironment: %v", err)
	}

	restored, err := serialization.ParseEnvironment(data, "yaml")
	if err != nil {
		t.Fatalf("ParseEnvironment: %v", err)
	}
	if restored.Name != original.Name {
		t.Errorf("name: %q != %q", restored.Name, original.Name)
	}
	if len(restored.Variables) != 2 {
		t.Fatalf("variables: %d", len(restored.Variables))
	}
	if restored.Variables[0].Value != "https://staging.example.com" {
		t.Errorf("value: %q", restored.Variables[0].Value)
	}
	if !restored.Variables[1].Secret {
		t.Errorf("expected token to be secret")
	}
}
