package openapi_test

import (
	"os"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

func loadExampleSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	data, err := os.ReadFile("../../../examples/server/openapi.yaml")
	if err != nil {
		t.Fatalf("reading example spec: %v", err)
	}
	spec, err := openapi.ParseBytes(data, "yaml")
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	return spec
}

func TestParseBytes_ValidSpec(t *testing.T) {
	spec := loadExampleSpec(t)
	title, version := spec.Info()
	if title == "" {
		t.Error("expected non-empty title")
	}
	if version == "" {
		t.Error("expected non-empty version")
	}
}

func TestParseBytes_InvalidSpec(t *testing.T) {
	_, err := openapi.ParseBytes([]byte("not yaml at all: {{{"), "yaml")
	if err == nil {
		t.Error("expected error for invalid spec")
	}
}

func TestGenerateRequests_Count(t *testing.T) {
	spec := loadExampleSpec(t)
	reqs, err := openapi.GenerateRequests(spec, "")
	if err != nil {
		t.Fatalf("GenerateRequests: %v", err)
	}
	// spec has: GET /health, POST /auth/login, GET /users, POST /users,
	//           GET /users/{id}, PUT /users/{id}, DELETE /users/{id}, GET /protected = 8 ops
	if len(reqs) != 8 {
		t.Errorf("expected 8 requests, got %d", len(reqs))
	}
}

func TestGenerateRequests_BaseURL(t *testing.T) {
	spec := loadExampleSpec(t)
	reqs, err := openapi.GenerateRequests(spec, "https://staging.example.com")
	if err != nil {
		t.Fatalf("GenerateRequests: %v", err)
	}
	for _, r := range reqs {
		if r.Request.URL[:len("https://staging.example.com")] != "https://staging.example.com" {
			t.Errorf("URL %q does not start with override base URL", r.Request.URL)
		}
	}
}

func TestGenerateRequests_PathParams(t *testing.T) {
	spec := loadExampleSpec(t)
	reqs, err := openapi.GenerateRequests(spec, "")
	if err != nil {
		t.Fatalf("GenerateRequests: %v", err)
	}
	for _, r := range reqs {
		if r.Request.Method == domain.MethodGET && r.Request.URL == "http://localhost:8080/users/{{id}}" {
			return
		}
	}
	t.Error("expected /users/{{id}} URL for GET /users/{id}")
}

func TestGenerateRequests_JSONBody(t *testing.T) {
	spec := loadExampleSpec(t)
	reqs, err := openapi.GenerateRequests(spec, "")
	if err != nil {
		t.Fatalf("GenerateRequests: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "createUser" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Error("expected JSON body for createUser")
			}
			return
		}
	}
	t.Error("createUser operation not found")
}

func TestValidateResponse_Pass(t *testing.T) {
	spec := loadExampleSpec(t)
	resp := &domain.Response{
		StatusCode: 200,
		Headers:    []domain.Header{{Name: "Content-Type", Value: "application/json"}},
		Body:       []byte(`{"status":"ok","time":"2026-01-01T00:00:00Z"}`),
	}
	errs := openapi.ValidateResponse(spec, "getHealth", resp)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateResponse_Fail_MissingField(t *testing.T) {
	spec := loadExampleSpec(t)
	// Missing required "time" field
	resp := &domain.Response{
		StatusCode: 200,
		Headers:    []domain.Header{{Name: "Content-Type", Value: "application/json"}},
		Body:       []byte(`{"status":"ok"}`),
	}
	errs := openapi.ValidateResponse(spec, "getHealth", resp)
	if len(errs) == 0 {
		t.Error("expected schema validation error for missing required field")
	}
}

func TestValidateResponse_UnknownOperation(t *testing.T) {
	spec := loadExampleSpec(t)
	resp := &domain.Response{StatusCode: 200, Body: []byte(`{}`)}
	errs := openapi.ValidateResponse(spec, "doesNotExist", resp)
	if len(errs) == 0 {
		t.Error("expected error for unknown operationID")
	}
}

// ---- Swagger 2.0 tests ----

const swagger2Fixture = `{
  "swagger": "2.0",
  "info": {"title": "Petstore", "version": "1.0.6"},
  "host": "petstore.swagger.io",
  "basePath": "/v2",
  "schemes": ["https", "http"],
  "securityDefinitions": {
    "api_key": {"type": "apiKey", "name": "api_key", "in": "header"},
    "petstore_auth": {"type": "oauth2"}
  },
  "paths": {
    "/pet/{petId}": {
      "get": {
        "operationId": "getPetById",
        "summary": "Find pet by ID",
        "tags": ["pet"],
        "parameters": [
          {"name": "petId", "in": "path", "required": true, "type": "integer"}
        ],
        "security": [{"api_key": []}]
      },
      "delete": {
        "operationId": "deletePet",
        "summary": "Deletes a pet",
        "tags": ["pet"],
        "parameters": [
          {"name": "api_key", "in": "header", "required": false},
          {"name": "petId", "in": "path", "required": true, "type": "integer"}
        ]
      }
    },
    "/pet": {
      "post": {
        "operationId": "addPet",
        "summary": "Add a new pet",
        "tags": ["pet"],
        "parameters": [
          {"name": "body", "in": "body", "required": true,
            "schema": {
              "type": "object",
              "required": ["name"],
              "properties": {"name": {}, "status": {}}
            }
          }
        ],
        "security": [{"petstore_auth": ["write:pets"]}]
      }
    }
  }
}`

func TestIsSwagger2_JSON(t *testing.T) {
	if !openapi.IsSwagger2([]byte(swagger2Fixture)) {
		t.Error("expected IsSwagger2=true for Swagger 2.0 JSON")
	}
}

func TestIsSwagger2_OpenAPI3(t *testing.T) {
	if openapi.IsSwagger2([]byte(`{"openapi":"3.0.0","info":{"title":"T","version":"1"},"paths":{}}`)) {
		t.Error("expected IsSwagger2=false for OpenAPI 3.0")
	}
}

func TestParseSwagger2Bytes_Info(t *testing.T) {
	title, version, reqs, err := openapi.ParseSwagger2Bytes([]byte(swagger2Fixture), "json", "")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes: %v", err)
	}
	if title != "Petstore" {
		t.Errorf("title: got %q, want %q", title, "Petstore")
	}
	if version != "1.0.6" {
		t.Errorf("version: got %q, want %q", version, "1.0.6")
	}
	if len(reqs) != 3 {
		t.Errorf("expected 3 requests, got %d", len(reqs))
	}
}

func TestParseSwagger2Bytes_PathParams(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(swagger2Fixture), "json", "")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "getPetById" {
			if !strings.Contains(r.Request.URL, "{{petId}}") {
				t.Errorf("expected {{petId}} placeholder in URL, got %q", r.Request.URL)
			}
			return
		}
	}
	t.Error("getPetById not found")
}

func TestParseSwagger2Bytes_BaseURLOverride(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(swagger2Fixture), "json", "https://staging.example.com")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes: %v", err)
	}
	for _, r := range reqs {
		if !strings.HasPrefix(r.Request.URL, "https://staging.example.com") {
			t.Errorf("URL %q does not start with override base URL", r.Request.URL)
		}
	}
}

func TestParseSwagger2Bytes_BodyStub(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(swagger2Fixture), "json", "")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "addPet" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Error("expected JSON body for addPet")
			}
			if !strings.Contains(r.Request.Body.Content, `"name"`) {
				t.Errorf("body stub missing name field: %s", r.Request.Body.Content)
			}
			return
		}
	}
	t.Error("addPet not found")
}

func TestParseSwagger2Bytes_APIKeyAuth(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(swagger2Fixture), "json", "")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "getPetById" {
			if r.Request.Auth == nil || r.Request.Auth.Type != domain.AuthAPIKey {
				t.Errorf("expected apikey auth for getPetById, got %v", r.Request.Auth)
			}
			return
		}
	}
	t.Error("getPetById not found")
}

func TestParseSwagger2Bytes_OAuthBearer(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(swagger2Fixture), "json", "")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "addPet" {
			if r.Request.Auth == nil || r.Request.Auth.Type != domain.AuthBearer {
				t.Errorf("expected bearer auth for addPet (oauth2), got %v", r.Request.Auth)
			}
			return
		}
	}
	t.Error("addPet not found")
}
