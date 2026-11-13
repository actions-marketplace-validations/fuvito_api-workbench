package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"gopkg.in/yaml.v3"
)

// Minimal Swagger 2.0 structs — only the fields needed to generate request stubs.

type swagger2Doc struct {
	Info         swagger2Info                    `yaml:"info"                json:"info"`
	Host         string                          `yaml:"host"                json:"host"`
	BasePath     string                          `yaml:"basePath"            json:"basePath"`
	Schemes      []string                        `yaml:"schemes"             json:"schemes"`
	Paths        map[string]swagger2PathItem     `yaml:"paths"               json:"paths"`
	SecurityDefs map[string]swagger2SecurityDef  `yaml:"securityDefinitions" json:"securityDefinitions"`
}

type swagger2Info struct {
	Title   string `yaml:"title"   json:"title"`
	Version string `yaml:"version" json:"version"`
}

type swagger2PathItem struct {
	Get        *swagger2Operation  `yaml:"get"        json:"get"`
	Post       *swagger2Operation  `yaml:"post"       json:"post"`
	Put        *swagger2Operation  `yaml:"put"        json:"put"`
	Patch      *swagger2Operation  `yaml:"patch"      json:"patch"`
	Delete     *swagger2Operation  `yaml:"delete"     json:"delete"`
	Head       *swagger2Operation  `yaml:"head"       json:"head"`
	Options    *swagger2Operation  `yaml:"options"    json:"options"`
	Parameters []swagger2Parameter `yaml:"parameters" json:"parameters"`
}

type swagger2Operation struct {
	OperationID string                `yaml:"operationId" json:"operationId"`
	Summary     string                `yaml:"summary"     json:"summary"`
	Tags        []string              `yaml:"tags"        json:"tags"`
	Parameters  []swagger2Parameter   `yaml:"parameters"  json:"parameters"`
	Security    []map[string][]string `yaml:"security"    json:"security"`
}

type swagger2Parameter struct {
	Name     string          `yaml:"name"     json:"name"`
	In       string          `yaml:"in"       json:"in"`
	Required bool            `yaml:"required" json:"required"`
	Schema   *swagger2Schema `yaml:"schema"   json:"schema"`
}

type swagger2Schema struct {
	Type       string                    `yaml:"type"       json:"type"`
	Properties map[string]swagger2Schema `yaml:"properties" json:"properties"`
	Required   []string                  `yaml:"required"   json:"required"`
}

type swagger2SecurityDef struct {
	Type string `yaml:"type" json:"type"`
	In   string `yaml:"in"   json:"in"`
	Name string `yaml:"name" json:"name"`
}

// IsSwagger2 returns true when data appears to be a Swagger 2.0 (OAS2) spec.
// Scans only the first 1 KB to avoid parsing large files.
func IsSwagger2(data []byte) bool {
	n := 1024
	if len(data) < n {
		n = len(data)
	}
	s := string(data[:n])
	// JSON: "swagger": "2.0"
	if strings.Contains(s, `"swagger"`) && strings.Contains(s, `"2.`) {
		return true
	}
	// YAML: swagger: "2.0" or swagger: '2.0'
	return strings.Contains(s, "swagger:") &&
		(strings.Contains(s, `"2.0"`) || strings.Contains(s, "'2.0'") || strings.Contains(s, " 2.0"))
}

// ParseSwagger2Bytes parses a Swagger 2.0 spec and generates request stubs directly,
// bypassing kin-openapi which only supports OpenAPI 3.x.
// If baseURL is non-empty it overrides the spec's host+basePath+schemes.
func ParseSwagger2Bytes(data []byte, format, baseURL string) (title, version string, reqs []GeneratedRequest, err error) {
	var doc swagger2Doc
	switch strings.ToLower(format) {
	case "json":
		err = json.Unmarshal(data, &doc)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &doc)
	default:
		if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
			err = yaml.Unmarshal(data, &doc)
		}
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("parsing Swagger 2.0: %w", err)
	}

	title = doc.Info.Title
	version = doc.Info.Version

	base := baseURL
	if base == "" {
		base = buildSwagger2BaseURL(doc)
	}

	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := doc.Paths[path]
		ops := map[string]*swagger2Operation{
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
			params := mergeSwagger2Params(item.Parameters, op.Parameters)
			req := buildSwagger2Request(base, method, path, op, params, doc.SecurityDefs)
			reqs = append(reqs, GeneratedRequest{
				Request:     req,
				OperationID: op.OperationID,
				Summary:     op.Summary,
				Tags:        op.Tags,
			})
		}
	}

	return title, version, reqs, nil
}

func buildSwagger2BaseURL(doc swagger2Doc) string {
	scheme := "https"
	for _, s := range doc.Schemes {
		if s == "https" {
			break
		}
		if s == "http" {
			scheme = "http"
		}
	}
	if doc.Host == "" {
		return strings.TrimRight(doc.BasePath, "/")
	}
	return strings.TrimRight(scheme+"://"+doc.Host+doc.BasePath, "/")
}

func buildSwagger2Request(
	base, method, path string,
	op *swagger2Operation,
	params []swagger2Parameter,
	secDefs map[string]swagger2SecurityDef,
) *domain.Request {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = method + " " + path
	}

	req := &domain.Request{
		Title:  name,
		Method: domain.HTTPMethod(method),
		URL:    base + path,
	}

	for _, p := range params {
		switch p.In {
		case "path":
			req.URL = strings.ReplaceAll(req.URL, "{"+p.Name+"}", "{{"+p.Name+"}}")
		case "query":
			if p.Required {
				req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: p.Name})
			}
		case "header":
			if p.Required {
				req.Headers = append(req.Headers, domain.Header{Name: p.Name})
			}
		case "body":
			if p.Schema != nil {
				req.Body = &domain.RequestBody{
					Type:    domain.BodyJSON,
					Content: buildSwagger2BodyStub(p.Schema),
				}
			}
		}
	}

	req.Auth = resolveSwagger2Auth(op.Security, secDefs)
	return req
}

func resolveSwagger2Auth(security []map[string][]string, defs map[string]swagger2SecurityDef) *domain.Auth {
	for _, req := range security {
		for name := range req {
			def, ok := defs[name]
			if !ok {
				continue
			}
			switch def.Type {
			case "apiKey":
				loc := domain.APIKeyInHeader
				if def.In == "query" {
					loc = domain.APIKeyInQuery
				}
				return &domain.Auth{
					Type:   domain.AuthAPIKey,
					APIKey: &domain.APIKeyAuth{Key: def.Name, Value: "{{apiKey}}", Location: loc},
				}
			case "oauth2", "basic":
				return &domain.Auth{
					Type:   domain.AuthBearer,
					Bearer: &domain.BearerAuth{Token: "{{token}}"},
				}
			}
		}
	}
	return nil
}

// mergeSwagger2Params merges path-level parameters with operation-level ones.
// Operation-level parameters with the same name+in take precedence.
func mergeSwagger2Params(pathLevel, opLevel []swagger2Parameter) []swagger2Parameter {
	seen := make(map[string]bool, len(opLevel))
	for _, p := range opLevel {
		seen[p.In+":"+p.Name] = true
	}
	result := make([]swagger2Parameter, 0, len(opLevel)+len(pathLevel))
	result = append(result, opLevel...)
	for _, p := range pathLevel {
		if !seen[p.In+":"+p.Name] {
			result = append(result, p)
		}
	}
	return result
}

func buildSwagger2BodyStub(s *swagger2Schema) string {
	if s == nil || len(s.Properties) == 0 {
		return "{}"
	}

	required := make(map[string]bool, len(s.Required))
	for _, r := range s.Required {
		required[r] = true
	}

	var reqFields, optFields []string
	for _, r := range s.Required {
		if _, ok := s.Properties[r]; ok {
			reqFields = append(reqFields, r)
		}
	}
	for name := range s.Properties {
		if !required[name] {
			optFields = append(optFields, name)
		}
	}
	sort.Strings(optFields)

	fields := append(reqFields, optFields...)
	var sb strings.Builder
	sb.WriteString("{")
	for i, name := range fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(`"` + name + `": "{{` + name + `}}"`)
	}
	sb.WriteString("}")
	return sb.String()
}
