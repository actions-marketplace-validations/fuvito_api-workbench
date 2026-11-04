package variables_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

func testEnv() *domain.Environment {
	env := domain.NewEnvironment("test")
	env.Variables = []domain.EnvironmentVariable{
		{Name: "baseUrl", Value: "https://api.example.com"},
		{Name: "version", Value: "v1"},
		{Name: "token", Value: "secret", Secret: true},
	}
	return env
}

func TestInterpolate_Simple(t *testing.T) {
	r := variables.NewResolver(testEnv())

	cases := []struct{ in, want string }{
		{"{{baseUrl}}/items", "https://api.example.com/items"},
		{"{{baseUrl}}/{{version}}/items", "https://api.example.com/v1/items"},
		{"no variables here", "no variables here"},
		{"{{missing}}", "{{missing}}"},        // unresolved left unchanged
		{"{{ baseUrl }}", "https://api.example.com"}, // whitespace trimmed
	}
	for _, c := range cases {
		if got := r.Interpolate(c.in); got != c.want {
			t.Errorf("Interpolate(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestInterpolate_RuntimeOverride(t *testing.T) {
	r := variables.NewResolver(testEnv())
	r.Set("baseUrl", "https://staging.example.com")

	got := r.Interpolate("{{baseUrl}}/items")
	want := "https://staging.example.com/items"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestInterpolateRequest(t *testing.T) {
	r := variables.NewResolver(testEnv())

	req := &domain.Request{
		Method: domain.MethodGET,
		URL:    "{{baseUrl}}/{{version}}/users",
		Headers: []domain.Header{
			{Name: "Accept", Value: "application/json"},
			{Name: "X-Version", Value: "{{version}}"},
		},
		QueryParams: []domain.QueryParam{
			{Name: "api", Value: "{{version}}"},
		},
		Auth: &domain.Auth{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuth{Token: "{{token}}"},
		},
	}

	resolved := r.InterpolateRequest(req)

	if resolved.URL != "https://api.example.com/v1/users" {
		t.Errorf("URL = %q; want https://api.example.com/v1/users", resolved.URL)
	}
	if resolved.Headers[1].Value != "v1" {
		t.Errorf("X-Version = %q; want v1", resolved.Headers[1].Value)
	}
	if resolved.QueryParams[0].Value != "v1" {
		t.Errorf("query api = %q; want v1", resolved.QueryParams[0].Value)
	}
	if resolved.Auth.Bearer.Token != "secret" {
		t.Errorf("bearer token = %q; want secret", resolved.Auth.Bearer.Token)
	}

	// original must not be mutated
	if req.URL != "{{baseUrl}}/{{version}}/users" {
		t.Error("original request URL was mutated")
	}
}

func TestInterpolateRequest_NilEnv(t *testing.T) {
	r := variables.NewResolver(nil)
	req := &domain.Request{
		Method: domain.MethodGET,
		URL:    "{{baseUrl}}/items",
	}
	resolved := r.InterpolateRequest(req)
	if resolved.URL != "{{baseUrl}}/items" {
		t.Errorf("unresolved ref should remain: got %q", resolved.URL)
	}
}
