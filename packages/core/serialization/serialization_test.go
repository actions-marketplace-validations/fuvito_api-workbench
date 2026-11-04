package serialization_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

const yamlRequest = `
schemaVersion: "1"
name: Get User
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
  "name": "Create Item",
  "method": "POST",
  "url": "https://api.example.com/items",
  "body": {"type": "json", "content": "{\"name\": \"test\"}"}
}`

func TestParseRequest_YAML(t *testing.T) {
	req, err := serialization.ParseRequest([]byte(yamlRequest), "yaml")
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Name != "Get User" {
		t.Errorf("name: got %q", req.Name)
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
	if req.Name != "Get User" {
		t.Errorf("name: got %q", req.Name)
	}
}

func TestWriteRequest_RoundTrip(t *testing.T) {
	original := &domain.Request{
		Name:   "Round Trip",
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
