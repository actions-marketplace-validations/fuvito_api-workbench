package openapi_test

import (
	"os"
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
