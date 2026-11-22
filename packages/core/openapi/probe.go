package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/getkin/kin-openapi/openapi3"
)

// ProbeKind classifies the contract rule a negative probe exercises.
type ProbeKind string

const (
	ProbeDropRequiredField    ProbeKind = "drop_required_field"
	ProbeWrongFieldType       ProbeKind = "wrong_field_type"
	ProbeMissingRequiredParam ProbeKind = "missing_required_param"
	ProbeMissingAuth          ProbeKind = "missing_auth"
)

// NegativeProbe is a request intentionally crafted to violate the spec,
// paired with metadata describing which contract rule it exercises.
type NegativeProbe struct {
	Request     *domain.Request
	OperationID string
	Tags        []string
	Method      string
	Path        string
	Kind        ProbeKind
	Description string
}

// GenerateNegativeProbes creates probes for every operation that declares
// a request body, security requirement, or required query parameters.
// Each probe is a request that should be rejected by a spec-compliant server.
func GenerateNegativeProbes(s *Spec, baseURL string) ([]NegativeProbe, error) {
	base := baseURL
	if base == "" && len(s.doc.Servers) > 0 {
		base = strings.TrimRight(s.doc.Servers[0].URL, "/")
	}

	// Build happy-path requests as the base for mutations.
	happyPath, err := GenerateRequests(s, base)
	if err != nil {
		return nil, err
	}
	happyByOp := make(map[string]GeneratedRequest, len(happyPath))
	for _, gr := range happyPath {
		if gr.OperationID != "" {
			happyByOp[gr.OperationID] = gr
		}
	}

	// Sort paths for determinism.
	paths := make([]string, 0, len(s.doc.Paths.Map()))
	for p := range s.doc.Paths.Map() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var probes []NegativeProbe

	for _, path := range paths {
		item := s.doc.Paths.Value(path)
		if item == nil {
			continue
		}
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
			op := opFromItem(item, method)
			if op == nil {
				continue
			}
			op = withPathParameters(item, op)

			var baseReq *domain.Request
			if gr, ok := happyByOp[op.OperationID]; ok {
				baseReq = gr.Request
			} else {
				baseReq = buildRequest(s.doc, base, method, path, op)
			}

			// 1. Missing auth — when operation (or global spec) declares security.
			if hasSecurityRequirement(s.doc, op) {
				req := cloneRequest(baseReq)
				req.Auth = nil
				probes = append(probes, NegativeProbe{
					Request:     req,
					OperationID: op.OperationID,
					Tags:        op.Tags,
					Method:      method,
					Path:        path,
					Kind:        ProbeMissingAuth,
					Description: method + " " + path + " — no credentials (expect 401/403)",
				})
			}

			// 2. JSON body probes: drop and corrupt required fields. For a oneOf or anyOf body each
			// branch is probed with its own fields; the body generated for the happy path only covers
			// the first.
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				jsonMT := firstJSONMediaType(op.RequestBody.Value)
				if jsonMT != nil && jsonMT.Schema != nil && jsonMT.Schema.Value != nil {
					top := jsonMT.Schema.Value
					bodyProbes := func(schema *openapi3.Schema, suffix string) {
						for _, field := range schema.Required {
							req := cloneRequest(baseReq)
							req.Body = &domain.RequestBody{Type: domain.BodyJSON, Content: bodyWithoutField(schema, field)}
							probes = append(probes, NegativeProbe{
								Request: req, OperationID: op.OperationID, Tags: op.Tags, Method: method, Path: path,
								Kind:        ProbeDropRequiredField,
								Description: method + " " + path + " — missing required field '" + field + "'" + suffix + " (expect 400/422)",
							})
						}
						for _, field := range schema.Required {
							propRef, ok := schema.Properties[field]
							if !ok || propRef == nil || propRef.Value == nil {
								continue
							}
							wrongVal := wrongTypeValue(propRef.Value)
							if wrongVal == nil {
								continue
							}
							req := cloneRequest(baseReq)
							req.Body = &domain.RequestBody{Type: domain.BodyJSON, Content: bodyWithWrongField(schema, field, wrongVal)}
							probes = append(probes, NegativeProbe{
								Request: req, OperationID: op.OperationID, Tags: op.Tags, Method: method, Path: path,
								Kind:        ProbeWrongFieldType,
								Description: method + " " + path + " — wrong type for field '" + field + "'" + suffix + " (expect 400/422)",
							})
						}
					}
					bodyProbes(top, "")
					branches := append(append([]*openapi3.SchemaRef(nil), top.OneOf...), top.AnyOf...)
					for i, br := range branches {
						if br != nil && br.Value != nil {
							bodyProbes(br.Value, fmt.Sprintf(" of branch %d", i+1))
						}
					}
				}
			}

			// 3. Missing required query parameters and headers.
			for _, pRef := range op.Parameters {
				if pRef == nil || pRef.Value == nil || !pRef.Value.Required {
					continue
				}
				p := pRef.Value
				switch p.In {
				case "query":
					req := cloneRequest(baseReq)
					req.QueryParams = removeQueryParam(req.QueryParams, p.Name)
					probes = append(probes, NegativeProbe{
						Request: req, OperationID: op.OperationID, Tags: op.Tags, Method: method, Path: path,
						Kind:        ProbeMissingRequiredParam,
						Description: method + " " + path + " — missing required query param '" + p.Name + "' (expect 400)",
					})
				case "header":
					req := cloneRequest(baseReq)
					req.Headers = removeHeader(req.Headers, p.Name)
					probes = append(probes, NegativeProbe{
						Request: req, OperationID: op.OperationID, Tags: op.Tags, Method: method, Path: path,
						Kind:        ProbeMissingRequiredParam,
						Description: method + " " + path + " — missing required header '" + p.Name + "' (expect 4xx)",
					})
				}
			}
		}
	}

	return probes, nil
}

func opFromItem(item *openapi3.PathItem, method string) *openapi3.Operation {
	switch method {
	case "GET":
		return item.Get
	case "POST":
		return item.Post
	case "PUT":
		return item.Put
	case "PATCH":
		return item.Patch
	case "DELETE":
		return item.Delete
	case "HEAD":
		return item.Head
	case "OPTIONS":
		return item.Options
	}
	return nil
}

// hasSecurityRequirement returns true if the operation (or the spec globally)
// declares at least one security scheme. An explicit empty list on the operation
// overrides the global requirement.
func hasSecurityRequirement(doc *openapi3.T, op *openapi3.Operation) bool {
	reqs := securityRequirements(doc, op)
	if len(reqs) == 0 {
		return false
	}
	for _, r := range reqs {
		if len(r) == 0 {
			return false // an empty alternative makes authentication optional
		}
	}
	return true
}

func firstJSONMediaType(rb *openapi3.RequestBody) *openapi3.MediaType {
	for ct, mt := range rb.Content {
		if strings.Contains(ct, "json") {
			return mt
		}
	}
	return nil
}

// cloneRequest creates a copy so mutations don't affect the original.
// Slices are deep-copied; the Auth pointer is shared (callers override it explicitly).
func cloneRequest(r *domain.Request) *domain.Request {
	clone := *r
	if r.Headers != nil {
		clone.Headers = make([]domain.Header, len(r.Headers))
		copy(clone.Headers, r.Headers)
	}
	if r.QueryParams != nil {
		clone.QueryParams = make([]domain.QueryParam, len(r.QueryParams))
		copy(clone.QueryParams, r.QueryParams)
	}
	return &clone
}

// bodyWithoutField returns a JSON object with all schema properties populated
// except the named field.
func bodyWithoutField(schema *openapi3.Schema, omit string) string {
	obj := make(map[string]interface{})
	for name, ref := range schema.Properties {
		if name == omit {
			continue
		}
		typ := ""
		if ref != nil && ref.Value != nil {
			typ = firstSchemaType(ref.Value)
		}
		obj[name] = placeholderForType(typ)
	}
	b, _ := json.Marshal(obj)
	return string(b)
}

// bodyWithWrongField returns a JSON object with all schema properties populated,
// but sets fieldName to a value of the wrong type.
func bodyWithWrongField(schema *openapi3.Schema, fieldName string, wrongValue interface{}) string {
	obj := make(map[string]interface{})
	for name, ref := range schema.Properties {
		if name == fieldName {
			obj[name] = wrongValue
			continue
		}
		typ := ""
		if ref != nil && ref.Value != nil {
			typ = firstSchemaType(ref.Value)
		}
		obj[name] = placeholderForType(typ)
	}
	b, _ := json.Marshal(obj)
	return string(b)
}

func firstSchemaType(schema *openapi3.Schema) string {
	if schema.Type == nil {
		return ""
	}
	for _, t := range *schema.Type {
		return t
	}
	return ""
}

func placeholderForType(typ string) interface{} {
	switch typ {
	case "integer", "number":
		return 1
	case "boolean":
		return true
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return "placeholder"
	}
}

// wrongTypeValue returns a value of the wrong type for the given schema.
// Returns nil when no meaningful wrong-type probe can be constructed.
func wrongTypeValue(schema *openapi3.Schema) interface{} {
	switch firstSchemaType(schema) {
	case "string":
		return 0 // integer where string is expected
	case "integer", "number":
		return "not-a-number"
	case "boolean":
		return "not-a-bool"
	case "array":
		return "not-an-array"
	case "object":
		return "not-an-object"
	default:
		return nil
	}
}

func removeQueryParam(params []domain.QueryParam, name string) []domain.QueryParam {
	out := make([]domain.QueryParam, 0, len(params))
	for _, p := range params {
		if !strings.EqualFold(p.Name, name) {
			out = append(out, p)
		}
	}
	return out
}

func removeHeader(headers []domain.Header, name string) []domain.Header {
	out := make([]domain.Header, 0, len(headers))
	for _, h := range headers {
		if !strings.EqualFold(h.Name, name) {
			out = append(out, h)
		}
	}
	return out
}
