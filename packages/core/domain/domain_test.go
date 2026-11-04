package domain_test

import (
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func TestHTTPMethodIsValid(t *testing.T) {
	valid := []domain.HTTPMethod{
		domain.MethodGET, domain.MethodPOST, domain.MethodPUT,
		domain.MethodPATCH, domain.MethodDELETE, domain.MethodHEAD, domain.MethodOPTIONS,
	}
	for _, m := range valid {
		if !m.IsValid() {
			t.Errorf("expected %q to be valid", m)
		}
	}

	invalid := []domain.HTTPMethod{"", "get", "CONNECT", "TRACE"}
	for _, m := range invalid {
		if m.IsValid() {
			t.Errorf("expected %q to be invalid", m)
		}
	}
}

func TestRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.Request
		wantErr bool
	}{
		{
			name:    "valid GET",
			req:     domain.Request{Method: domain.MethodGET, URL: "https://example.com"},
			wantErr: false,
		},
		{
			name:    "missing method",
			req:     domain.Request{URL: "https://example.com"},
			wantErr: true,
		},
		{
			name:    "missing URL",
			req:     domain.Request{Method: domain.MethodGET},
			wantErr: true,
		},
		{
			name:    "invalid method",
			req:     domain.Request{Method: "INVALID", URL: "https://example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponseIsSuccess(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{200, true}, {201, true}, {204, true}, {299, true},
		{300, false}, {400, false}, {404, false}, {500, false},
	}
	for _, c := range cases {
		r := &domain.Response{StatusCode: c.code}
		if r.IsSuccess() != c.want {
			t.Errorf("StatusCode %d: IsSuccess() = %v, want %v", c.code, r.IsSuccess(), c.want)
		}
	}
}

func TestEnvironmentLookup(t *testing.T) {
	env := domain.NewEnvironment("test")
	env.Variables = []domain.EnvironmentVariable{
		{Name: "baseUrl", Value: "https://api.example.com"},
		{Name: "apiKey", Value: "secret", Secret: true},
	}

	val, ok := env.Lookup("baseUrl")
	if !ok || val != "https://api.example.com" {
		t.Errorf("Lookup(baseUrl) = %q, %v; want %q, true", val, ok, "https://api.example.com")
	}

	_, ok = env.Lookup("missing")
	if ok {
		t.Error("Lookup(missing) should return false")
	}
}

func TestRequestWithTimeout(t *testing.T) {
	req := domain.Request{
		Method:  domain.MethodGET,
		URL:     "https://example.com",
		Timeout: 30 * time.Second,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if req.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v; want 30s", req.Timeout)
	}
}
