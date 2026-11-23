package openapi

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/getkin/kin-openapi/openapi3"
)

// Spec wraps a loaded OpenAPI document.
type Spec struct {
	doc     *openapi3.T
	rawPath string
}

// ParseBytes parses an OpenAPI 3.0 or 3.1 spec from raw bytes.
// format must be "yaml", "yml", or "json".
func ParseBytes(data []byte, format string) (*Spec, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	var doc *openapi3.T
	var err error

	switch strings.ToLower(format) {
	case "yaml", "yml":
		doc, err = loader.LoadFromData(data)
	case "json":
		doc, err = loader.LoadFromData(data)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
	if err != nil {
		return nil, fmt.Errorf("loading spec: %w", err)
	}

	restore := setAsideNonJSONExamples(doc)
	err = doc.Validate(context.Background())
	restore()
	if err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	return &Spec{doc: doc}, nil
}

// Info returns the spec title and version.
func (s *Spec) Info() (title, version string) {
	if s.doc.Info == nil {
		return "", ""
	}
	return s.doc.Info.Title, s.doc.Info.Version
}

// GeneratedRequest is a domain.Request augmented with metadata from the spec.
type GeneratedRequest struct {
	Request     *domain.Request
	OperationID string
	Summary     string
	Tags        []string
	Path        string // raw path from the spec, e.g. "/user/{username}"
}

// FirstServerURL returns the spec's first server URL without a trailing slash, or "" when the
// spec declares none.
func (s *Spec) FirstServerURL() string {
	if len(s.doc.Servers) == 0 {
		return ""
	}
	return strings.TrimRight(s.doc.Servers[0].URL, "/")
}

// GenerateRequests creates one domain.Request per operation in the spec.
// baseURL overrides the first server URL when non-empty.
func GenerateRequests(s *Spec, baseURL string) ([]GeneratedRequest, error) {
	base := baseURL
	if base == "" && len(s.doc.Servers) > 0 {
		base = strings.TrimRight(s.doc.Servers[0].URL, "/")
	}

	var results []GeneratedRequest

	// Iterate paths in sorted order for determinism.
	paths := make([]string, 0, len(s.doc.Paths.Map()))
	for p := range s.doc.Paths.Map() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := s.doc.Paths.Value(path)
		if item == nil {
			continue
		}
		ops := map[string]*openapi3.Operation{
			"GET":     item.Get,
			"POST":    item.Post,
			"PUT":     item.Put,
			"PATCH":   item.Patch,
			"DELETE":  item.Delete,
			"HEAD":    item.Head,
			"OPTIONS": item.Options,
			"TRACE":   item.Trace,
		}
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"} {
			op := ops[method]
			if op == nil {
				continue
			}

			req := buildRequest(s.doc, base, method, path, withPathParameters(item, op))
			gr := GeneratedRequest{
				Request:     req,
				OperationID: op.OperationID,
				Summary:     op.Summary,
				Tags:        op.Tags,
				Path:        path,
			}
			results = append(results, gr)
		}
	}

	return results, nil
}

// withPathParameters returns op with the parameters declared on its path item added, so a
// parameter written once for every method of a path (the usual place for {id}) is seen. An
// operation's own parameter replaces a path-level one with the same name and location. op itself
// is not modified.
func withPathParameters(item *openapi3.PathItem, op *openapi3.Operation) *openapi3.Operation {
	if len(item.Parameters) == 0 {
		return op
	}
	merged := make(openapi3.Parameters, 0, len(item.Parameters)+len(op.Parameters))
	for _, shared := range item.Parameters {
		overridden := false
		for _, own := range op.Parameters {
			if shared != nil && own != nil && shared.Value != nil && own.Value != nil &&
				shared.Value.Name == own.Value.Name && shared.Value.In == own.Value.In {
				overridden = true
				break
			}
		}
		if !overridden {
			merged = append(merged, shared)
		}
	}
	merged = append(merged, op.Parameters...)
	copied := *op
	copied.Parameters = merged
	return &copied
}

// parameterValues are the values a generated request carries for a required parameter: the spec's
// own example, default or first enum value, otherwise a {{name}} placeholder so a run that has no
// value for it is reported as "not sent" instead of sending an empty value. An array is written
// the way the parameter's style and explode say (a,b,c in one value, or one value per item).
func parameterValues(p *openapi3.Parameter) []string {
	var v any
	switch {
	case p.Example != nil:
		v = p.Example
	case len(p.Examples) > 0:
		names := make([]string, 0, len(p.Examples))
		for n := range p.Examples {
			names = append(names, n)
		}
		sort.Strings(names)
		if ref := p.Examples[names[0]]; ref != nil && ref.Value != nil {
			v = ref.Value.Value
		}
	}
	if v == nil && p.Schema != nil && p.Schema.Value != nil {
		sc := p.Schema.Value
		switch {
		case schemaExample(sc) != nil:
			v = schemaExample(sc)
		case sc.Default != nil:
			v = sc.Default
		case len(sc.Enum) > 0:
			v = sc.Enum[0]
		}
	}
	if v == nil {
		return []string{"{{" + p.Name + "}}"}
	}
	switch x := v.(type) {
	case []any:
		items := make([]string, len(x))
		for i, it := range x {
			items[i] = fmt.Sprint(it)
		}
		explode := p.In == "query" || p.In == "cookie"
		if p.Explode != nil {
			explode = *p.Explode
		}
		if p.In == "query" && explode && (p.Style == "" || p.Style == "form") {
			return items
		}
		sep := ","
		switch p.Style {
		case "spaceDelimited":
			sep = " "
		case "pipeDelimited":
			sep = "|"
		}
		return []string{strings.Join(items, sep)}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k, fmt.Sprint(x[k]))
		}
		return []string{strings.Join(parts, ",")}
	}
	return []string{fmt.Sprint(v)}
}

// securityRequirements is what the operation needs: its own list, or the spec's when it has none
// (an explicit empty list means no security).
func securityRequirements(doc *openapi3.T, op *openapi3.Operation) openapi3.SecurityRequirements {
	if op.Security != nil {
		return *op.Security
	}
	return doc.Security
}

// authFor builds the credentials a generated request carries for the first security scheme it can
// express, with {{placeholders}} for the values: {{token}} (bearer and OAuth2), {{basicUser}} and
// {{basicPass}} (basic), {{apiKey}} (API keys). An alternative that needs no credentials (an empty
// requirement) means authentication is optional, so none is added. A cookie API key comes back as
// a cookie pair.
func authFor(doc *openapi3.T, op *openapi3.Operation) (*domain.Auth, string) {
	reqs := securityRequirements(doc, op)
	for _, r := range reqs {
		if len(r) == 0 {
			return nil, ""
		}
	}
	for _, r := range reqs {
		names := make([]string, 0, len(r))
		for n := range r {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, name := range names {
			var scheme *openapi3.SecurityScheme
			if doc.Components != nil {
				if ref := doc.Components.SecuritySchemes[name]; ref != nil {
					scheme = ref.Value
				}
			}
			if scheme == nil {
				lower := strings.ToLower(name)
				if strings.Contains(lower, "bearer") || strings.Contains(lower, "auth") {
					return &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "{{token}}"}}, ""
				}
				continue
			}
			switch strings.ToLower(scheme.Type) {
			case "http":
				switch strings.ToLower(scheme.Scheme) {
				case "basic":
					return &domain.Auth{Type: domain.AuthBasic, Basic: &domain.BasicAuth{Username: "{{basicUser}}", Password: "{{basicPass}}"}}, ""
				case "bearer":
					return &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "{{token}}"}}, ""
				}
			case "oauth2", "openidconnect":
				return &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "{{token}}"}}, ""
			case "apikey":
				switch scheme.In {
				case "header":
					return &domain.Auth{Type: domain.AuthAPIKey, APIKey: &domain.APIKeyAuth{Key: scheme.Name, Value: "{{apiKey}}", Location: domain.APIKeyInHeader}}, ""
				case "query":
					return &domain.Auth{Type: domain.AuthAPIKey, APIKey: &domain.APIKeyAuth{Key: scheme.Name, Value: "{{apiKey}}", Location: domain.APIKeyInQuery}}, ""
				case "cookie":
					return nil, scheme.Name + "={{apiKey}}"
				}
			}
		}
	}
	return nil, ""
}

func buildRequest(doc *openapi3.T, base, method, path string, op *openapi3.Operation) *domain.Request {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = method + " " + path
	}

	url := base + path

	req := &domain.Request{
		Title:  name,
		Method: domain.HTTPMethod(method),
		URL:    url,
	}

	var cookies []string

	// Path parameters → keep as {{paramName}} placeholders
	for _, pRef := range op.Parameters {
		if pRef == nil || pRef.Value == nil {
			continue
		}
		p := pRef.Value
		switch p.In {
		case "path":
			placeholder := "{" + p.Name + "}"
			req.URL = strings.ReplaceAll(req.URL, placeholder, "{{"+p.Name+"}}")
		case "query":
			if p.Required {
				for _, v := range parameterValues(p) {
					req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: p.Name, Value: v})
				}
			}
		case "header":
			if p.Required {
				req.Headers = append(req.Headers, domain.Header{
					Name:  p.Name,
					Value: strings.Join(parameterValues(p), ","),
				})
			}
		case "cookie":
			if p.Required {
				cookies = append(cookies, p.Name+"="+strings.Join(parameterValues(p), ","))
			}
		}
	}

	auth, authCookie := authFor(doc, op)
	req.Auth = auth
	if authCookie != "" {
		cookies = append(cookies, authCookie)
	}
	if len(cookies) > 0 {
		req.Headers = append(req.Headers, domain.Header{Name: "Cookie", Value: strings.Join(cookies, "; ")})
	}

	// Request body → set body type to json when requestBody has application/json
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		content := op.RequestBody.Value.Content
		cts := make([]string, 0, len(content))
		for ct := range content {
			cts = append(cts, ct)
		}
		sort.Strings(cts)
		pick := func(match string) *openapi3.MediaType {
			for _, ct := range cts {
				if strings.Contains(ct, match) {
					return content[ct]
				}
			}
			return nil
		}
		if mt := pick("json"); mt != nil {
			req.Body = &domain.RequestBody{Type: domain.BodyJSON, Content: buildBodyStub(mt)}
		} else if mt := pick("x-www-form-urlencoded"); mt != nil {
			req.Body = &domain.RequestBody{Type: domain.BodyFormURLEncoded, Content: buildOASFormStub(mt)}
		} else if mt := pick("multipart/form-data"); mt != nil {
			req.Body = &domain.RequestBody{Type: domain.BodyMultipart, Content: buildOASFormStub(mt)}
		}
	}

	return req
}

// setAsideNonJSONExamples removes, until the returned function puts them back, the examples of
// media types that are not JSON. The library checks every example against the media type's schema
// as a JSON value, so a valid XML example string ("<note>...</note>") made the whole document
// invalid. The examples stay in the document for everything else.
func setAsideNonJSONExamples(doc *openapi3.T) func() {
	type saved struct {
		mt       *openapi3.MediaType
		example  any
		examples openapi3.Examples
	}
	var stash []saved
	visit := func(content openapi3.Content) {
		for ct, mt := range content {
			if mt == nil || isJSONDocumentType(ct) || (mt.Example == nil && len(mt.Examples) == 0) {
				continue
			}
			stash = append(stash, saved{mt, mt.Example, mt.Examples})
			mt.Example, mt.Examples = nil, nil
		}
	}
	visitResponses := func(r *openapi3.Responses) {
		if r == nil {
			return
		}
		for _, ref := range r.Map() {
			if ref != nil && ref.Value != nil {
				visit(ref.Value.Content)
			}
		}
	}
	visitBody := func(ref *openapi3.RequestBodyRef) {
		if ref != nil && ref.Value != nil {
			visit(ref.Value.Content)
		}
	}
	if doc.Paths != nil {
		for _, item := range doc.Paths.Map() {
			if item == nil {
				continue
			}
			for _, op := range item.Operations() {
				visitBody(op.RequestBody)
				visitResponses(op.Responses)
			}
		}
	}
	if c := doc.Components; c != nil {
		for _, rb := range c.RequestBodies {
			visitBody(rb)
		}
		for _, r := range c.Responses {
			if r != nil && r.Value != nil {
				visit(r.Value.Content)
			}
		}
	}
	return func() {
		for _, s := range stash {
			s.mt.Example, s.mt.Examples = s.example, s.examples
		}
	}
}
