package openapi

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/fuvito/api-workbench/packages/core/domain"
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

	if err := doc.Validate(context.Background()); err != nil {
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
		}
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
			op := ops[method]
			if op == nil {
				continue
			}

			req := buildRequest(base, method, path, op)
			gr := GeneratedRequest{
				Request:     req,
				OperationID: op.OperationID,
				Summary:     op.Summary,
				Tags:        op.Tags,
			}
			results = append(results, gr)
		}
	}

	return results, nil
}

func buildRequest(base, method, path string, op *openapi3.Operation) *domain.Request {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = method + " " + path
	}

	url := base + path

	req := &domain.Request{
		Name:   name,
		Method: domain.HTTPMethod(method),
		URL:    url,
	}

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
				req.QueryParams = append(req.QueryParams, domain.QueryParam{
					Name:  p.Name,
					Value: "",
				})
			}
		case "header":
			if p.Required {
				req.Headers = append(req.Headers, domain.Header{
					Name:  p.Name,
					Value: "",
				})
			}
		}
	}

	// Security → set auth type hint if bearer is required
	if op.Security != nil {
		for _, req2 := range *op.Security {
			for name := range req2 {
				if strings.Contains(strings.ToLower(name), "bearer") ||
					strings.Contains(strings.ToLower(name), "auth") {
					req.Auth = &domain.Auth{
						Type:   domain.AuthBearer,
						Bearer: &domain.BearerAuth{Token: "{{token}}"},
					}
					break
				}
			}
		}
	}

	// Request body → set body type to json when requestBody has application/json
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for ct := range op.RequestBody.Value.Content {
			if strings.Contains(ct, "json") {
				req.Body = &domain.RequestBody{
					Type:    domain.BodyJSON,
					Content: "{}",
				}
				break
			}
		}
	}

	return req
}
