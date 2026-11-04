package variables

import (
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// varPattern matches {{variableName}} references, allowing internal whitespace.
var varPattern = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

// Resolver substitutes {{variable}} references from a merged set of values.
// Environment variables are loaded first; runtime variables (from scenario
// extractions) can be added with Set and take precedence.
type Resolver struct {
	vars map[string]string
}

func NewResolver(env *domain.Environment) *Resolver {
	r := &Resolver{vars: make(map[string]string)}
	if env != nil {
		for _, v := range env.Variables {
			r.vars[v.Name] = v.Value
		}
	}
	return r
}

// Set adds or overrides a runtime variable (e.g. extracted from a scenario step).
func (r *Resolver) Set(name, value string) {
	r.vars[name] = value
}

// Interpolate replaces all {{name}} references in s with their resolved values.
// Unresolved references are left unchanged.
func (r *Resolver) Interpolate(s string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := strings.TrimSpace(match[2 : len(match)-2])
		if val, ok := r.vars[name]; ok {
			return val
		}
		return match
	})
}

// InterpolateRequest returns a shallow copy of req with all string fields interpolated.
// The original request is not modified.
func (r *Resolver) InterpolateRequest(req *domain.Request) *domain.Request {
	resolved := *req

	resolved.URL = r.Interpolate(req.URL)

	if len(req.Headers) > 0 {
		resolved.Headers = make([]domain.Header, len(req.Headers))
		for i, h := range req.Headers {
			resolved.Headers[i] = domain.Header{
				Name:  h.Name,
				Value: r.Interpolate(h.Value),
			}
		}
	}

	if len(req.QueryParams) > 0 {
		resolved.QueryParams = make([]domain.QueryParam, len(req.QueryParams))
		for i, p := range req.QueryParams {
			resolved.QueryParams[i] = domain.QueryParam{
				Name:  p.Name,
				Value: r.Interpolate(p.Value),
			}
		}
	}

	if req.Body != nil {
		body := *req.Body
		body.Content = r.Interpolate(req.Body.Content)
		resolved.Body = &body
	}

	if req.Auth != nil {
		resolved.Auth = r.interpolateAuth(req.Auth)
	}

	return &resolved
}

func (r *Resolver) interpolateAuth(auth *domain.Auth) *domain.Auth {
	out := *auth
	switch auth.Type {
	case domain.AuthBearer:
		if auth.Bearer != nil {
			b := *auth.Bearer
			b.Token = r.Interpolate(auth.Bearer.Token)
			out.Bearer = &b
		}
	case domain.AuthBasic:
		if auth.Basic != nil {
			b := *auth.Basic
			b.Username = r.Interpolate(auth.Basic.Username)
			b.Password = r.Interpolate(auth.Basic.Password)
			out.Basic = &b
		}
	case domain.AuthAPIKey:
		if auth.APIKey != nil {
			a := *auth.APIKey
			a.Value = r.Interpolate(auth.APIKey.Value)
			out.APIKey = &a
		}
	}
	return &out
}
