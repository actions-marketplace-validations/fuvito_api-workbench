package variables_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

func resolver() *variables.Resolver {
	env := domain.NewEnvironment("e")
	env.Variables = []domain.EnvironmentVariable{
		{Name: "tok", Value: "T"}, {Name: "user", Value: "U"}, {Name: "pass", Value: "P"}, {Name: "key", Value: "K"},
	}
	return variables.NewResolver(env)
}

func TestInterpolateRequest_Auth(t *testing.T) {
	r := resolver()
	cases := map[string]struct {
		in    *domain.Auth
		check func(*domain.Auth) bool
	}{
		"bearer": {&domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "{{tok}}"}}, func(a *domain.Auth) bool { return a.Bearer.Token == "T" }},
		"basic": {&domain.Auth{Type: domain.AuthBasic, Basic: &domain.BasicAuth{Username: "{{user}}", Password: "{{pass}}"}}, func(a *domain.Auth) bool {
			return a.Basic.Username == "U" && a.Basic.Password == "P"
		}},
		"apikey": {&domain.Auth{Type: domain.AuthAPIKey, APIKey: &domain.APIKeyAuth{Key: "{{key}}", Value: "{{key}}", Location: domain.APIKeyInQuery}}, func(a *domain.Auth) bool {
			return a.APIKey.Value == "K" && a.APIKey.Key == "{{key}}" && a.APIKey.Location == domain.APIKeyInQuery
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			req := &domain.Request{Method: domain.MethodGET, URL: "http://x", Auth: c.in}
			got := r.InterpolateRequest(req)
			if !c.check(got.Auth) {
				t.Errorf("not interpolated: %+v", got.Auth)
			}
			switch name {
			case "bearer":
				if req.Auth.Bearer.Token != "{{tok}}" {
					t.Error("original auth mutated")
				}
			case "basic":
				if req.Auth.Basic.Username != "{{user}}" {
					t.Error("original auth mutated")
				}
			case "apikey":
				if req.Auth.APIKey.Value != "{{key}}" {
					t.Error("original auth mutated")
				}
			}
		})
	}
}

func TestInterpolateRequest_AuthEdgeCases(t *testing.T) {
	r := resolver()
	for name, a := range map[string]*domain.Auth{
		"bearer without payload": {Type: domain.AuthBearer},
		"basic without payload":  {Type: domain.AuthBasic},
		"apikey without payload": {Type: domain.AuthAPIKey},
		"none":                   {Type: domain.AuthNone},
	} {
		t.Run(name, func(t *testing.T) {
			got := r.InterpolateRequest(&domain.Request{Method: domain.MethodGET, URL: "http://x", Auth: a})
			if got.Auth == nil || got.Auth.Type != a.Type {
				t.Errorf("auth should be preserved: %+v", got.Auth)
			}
		})
	}
}

func TestInterpolateRequest_BodyAndNilParts(t *testing.T) {
	r := resolver()
	req := &domain.Request{Method: domain.MethodPOST, URL: "http://x", Body: &domain.RequestBody{Type: domain.BodyJSON, Content: `{"t":"{{tok}}"}`}}
	got := r.InterpolateRequest(req)
	if got.Body.Content != `{"t":"T"}` || req.Body.Content != `{"t":"{{tok}}"}` {
		t.Errorf("body: %q (original %q)", got.Body.Content, req.Body.Content)
	}
	if got.Headers != nil || got.QueryParams != nil || got.Auth != nil {
		t.Error("absent parts must stay absent")
	}
}
