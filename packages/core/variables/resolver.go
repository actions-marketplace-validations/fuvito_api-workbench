package variables

import (
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// varPattern matches {{variableName}} references, allowing internal whitespace.
var varPattern = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

// Resolver substitutes {{variable}} references from a merged set of values.
// Environment values may refer to process environment variables as ${NAME}; they are expanded
// here, when a run starts. Environment variables are loaded first; runtime variables (from scenario
// extractions) can be added with Set and take precedence.
type Resolver struct {
	vars map[string]string
	eng  *Engine
	errs []error
}

// NewResolver builds a resolver whose dynamic values ({{$guid}}, {{$randomInt 1 9}}, …) use a
// fresh random seed.
func NewResolver(env *domain.Environment) *Resolver {
	return NewResolverWith(env, nil)
}

// NewResolverWith is NewResolver with a chosen engine (and so a chosen seed). A nil engine means
// a fresh random one. Environment values are evaluated here, once, in order: a value may use
// functions and the environment variables defined before it.
func NewResolverWith(env *domain.Environment, eng *Engine) *Resolver {
	if eng == nil {
		eng = NewEngine(RandomSeed())
	}
	r := &Resolver{vars: make(map[string]string), eng: eng}
	if env != nil {
		for _, v := range env.Variables {
			r.vars[v.Name] = r.evaluate(ExpandEnv(v.Value))
		}
	}
	return r
}

// Get returns the current value of a variable.
func (r *Resolver) Get(name string) (string, bool) {
	v, ok := r.vars[name]
	return v, ok
}

// Engine returns the engine behind the resolver's dynamic values.
func (r *Resolver) Engine() *Engine { return r.eng }

// Define evaluates scenario variables in order and makes them available as {{name}}. Each value is
// generated once, here, and stays the same for the rest of the run. A value may use functions,
// environment variables and the scenario variables defined before it.
func (r *Resolver) Define(vars []domain.EnvironmentVariable) {
	for _, v := range vars {
		r.vars[v.Name] = r.evaluate(ExpandEnv(v.Value))
	}
}

// TakeErrors returns the dynamic-value errors collected since the last call (an unknown
// function, a bad argument, …) and clears them. A value that failed is left as written.
func (r *Resolver) TakeErrors() []error {
	errs := r.errs
	r.errs = nil
	return errs
}

func (r *Resolver) evaluate(s string) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	return r.Interpolate(s)
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
		if strings.HasPrefix(name, "$") {
			out, err := r.eng.Call(name)
			if err != nil {
				r.errs = append(r.errs, err)
				return match
			}
			return out
		}
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
