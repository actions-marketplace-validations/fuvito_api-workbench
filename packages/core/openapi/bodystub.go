package openapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// stubSchema is the part of a schema the request body generator needs, so one implementation
// serves both OpenAPI 3.x and Swagger 2.0.
type stubSchema struct {
	Type     string // "object", "array", "string", "integer", "number", "boolean", or "" when unknown
	Props    map[string]*stubSchema
	Required []string
	Items    *stubSchema
	Example  any
	Default  any
	Enum     []any
	ReadOnly bool
}

const maxStubDepth = 8

// renderStub returns a JSON body template for a request:
//
//   - only the required properties are included, plus optional ones that carry an example or a
//     default (every other property would need a value the user has no reason to supply);
//   - read-only properties are left out (the server assigns them);
//   - a property with an example, a default or an enum uses that value as it is;
//   - any other property becomes a {{placeholder}} named after it: quoted for a string, bare for
//     a number, boolean, array or object, so that a value such as 9.99, true or [1,2] produces
//     valid JSON once variables are filled in.
//
// Nested objects and arrays stay one placeholder holding a JSON value.
func renderStub(s *stubSchema) string {
	if s == nil {
		return "{}"
	}
	if s.Type == "array" {
		if s.Items != nil && (s.Items.Type == "object" || len(s.Items.Props) > 0) {
			return "[\n  " + strings.ReplaceAll(renderStub(s.Items), "\n", "\n  ") + "\n]"
		}
		return "[]"
	}

	required := make(map[string]bool, len(s.Required))
	var fields []string
	for _, name := range s.Required {
		required[name] = true
		if p, ok := s.Props[name]; ok && !p.ReadOnly {
			fields = append(fields, name)
		}
	}
	var optional []string
	for name, p := range s.Props {
		if !required[name] && !p.ReadOnly && p.sample() != nil {
			optional = append(optional, name)
		}
	}
	sort.Strings(optional)
	fields = append(fields, optional...)
	if len(fields) == 0 {
		return "{}"
	}

	var sb strings.Builder
	sb.WriteString("{\n")
	for i, name := range fields {
		if i > 0 {
			sb.WriteString(",\n")
		}
		key, _ := json.Marshal(name)
		sb.WriteString("  " + string(key) + ": " + s.Props[name].valueText(name))
	}
	sb.WriteString("\n}")
	return sb.String()
}

// sample is the literal value taken from the schema: its example, else its default, else the first
// enum value; nil when it has none.
func (p *stubSchema) sample() any {
	switch {
	case p.Example != nil:
		return p.Example
	case p.Default != nil:
		return p.Default
	case len(p.Enum) > 0:
		return p.Enum[0]
	}
	return nil
}

func (p *stubSchema) valueText(name string) string {
	if v := p.sample(); v != nil {
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	if p.Type == "" || p.Type == "string" {
		return `"{{` + name + `}}"`
	}
	return "{{" + name + "}}"
}

// ---- OpenAPI 3.x ----

func stubFromOpenAPI(s *openapi3.Schema, depth int) *stubSchema {
	if s == nil || depth > maxStubDepth {
		return &stubSchema{}
	}
	out := &stubSchema{
		Type:     firstSchemaType(s),
		Required: append([]string(nil), s.Required...),
		Example:  schemaExample(s),
		Default:  s.Default,
		Enum:     s.Enum,
		ReadOnly: s.ReadOnly,
	}
	if len(s.Properties) > 0 {
		out.Props = make(map[string]*stubSchema, len(s.Properties))
		for name, ref := range s.Properties {
			if ref == nil || ref.Value == nil {
				out.Props[name] = &stubSchema{}
				continue
			}
			// Properties are described shallowly: nested structure stays one placeholder.
			prop := ref.Value
			ps := &stubSchema{Type: firstSchemaType(prop), Example: schemaExample(prop), Default: prop.Default, Enum: prop.Enum, ReadOnly: prop.ReadOnly}
			if ps.Type == "" && len(prop.Properties) > 0 {
				ps.Type = "object"
			}
			out.Props[name] = ps
		}
		if out.Type == "" {
			out.Type = "object"
		}
	}
	if s.Items != nil && s.Items.Value != nil {
		out.Items = stubFromOpenAPI(s.Items.Value, depth+1)
		if out.Type == "" {
			out.Type = "array"
		}
	}
	// Composition: allOf members are merged into this schema; for oneOf and anyOf the first branch is.
	members := append([]*openapi3.SchemaRef(nil), s.AllOf...)
	if len(s.OneOf) > 0 {
		members = append(members, s.OneOf[0])
	} else if len(s.AnyOf) > 0 {
		members = append(members, s.AnyOf[0])
	}
	for _, m := range members {
		if m != nil && m.Value != nil {
			out.merge(stubFromOpenAPI(m.Value, depth+1))
		}
	}
	return out
}

// merge adds the properties and required names of other to s.
func (s *stubSchema) merge(other *stubSchema) {
	if other == nil {
		return
	}
	if len(other.Props) > 0 && s.Props == nil {
		s.Props = map[string]*stubSchema{}
	}
	for name, p := range other.Props {
		if _, exists := s.Props[name]; !exists {
			s.Props[name] = p
		}
	}
	have := make(map[string]bool, len(s.Required))
	for _, r := range s.Required {
		have[r] = true
	}
	for _, r := range other.Required {
		if !have[r] {
			s.Required = append(s.Required, r)
		}
	}
	if s.Type == "" {
		s.Type = other.Type
	}
	if s.Items == nil {
		s.Items = other.Items
	}
}

// buildBodyStub generates a JSON body template from a request body media type.
func buildBodyStub(mt *openapi3.MediaType) string {
	if mt == nil || mt.Schema == nil || mt.Schema.Value == nil {
		return "{}"
	}
	if ex := wholeBodyExample(mt); ex != nil {
		if b, err := json.MarshalIndent(ex, "", "  "); err == nil {
			return string(b)
		}
	}
	return renderStub(stubFromOpenAPI(mt.Schema.Value, 0))
}

// buildOASFormStub generates the fields of a form or multipart body as name=value pairs: required
// properties and ones with an example, each with its example or a {{name}} placeholder.
func buildOASFormStub(mt *openapi3.MediaType) string {
	if mt == nil || mt.Schema == nil || mt.Schema.Value == nil {
		return ""
	}
	st := stubFromOpenAPI(mt.Schema.Value, 0)
	required := make(map[string]bool, len(st.Required))
	var names []string
	for _, n := range st.Required {
		required[n] = true
		if p, ok := st.Props[n]; ok && !p.ReadOnly {
			names = append(names, n)
		}
	}
	var extra []string
	for n, p := range st.Props {
		if !required[n] && !p.ReadOnly && p.sample() != nil {
			extra = append(extra, n)
		}
	}
	sort.Strings(extra)
	names = append(names, extra...)
	pairs := make([]string, 0, len(names))
	for _, n := range names {
		v := "{{" + n + "}}"
		if sv := st.Props[n].sample(); sv != nil {
			v = url.QueryEscape(fmt.Sprint(sv))
		}
		pairs = append(pairs, url.QueryEscape(n)+"="+v)
	}
	return strings.Join(pairs, "&")
}

// schemaExample is a schema's example: `example` (3.0, and deprecated in 3.1) or, when that is
// absent, the first entry of `examples` (3.1).
func schemaExample(s *openapi3.Schema) any {
	if s.Example != nil {
		return s.Example
	}
	if len(s.Examples) > 0 {
		return s.Examples[0]
	}
	return nil
}

// wholeBodyExample is a complete example the spec gives for the body, used as the body as it is:
// the media type's `example`, else its first named `examples` entry, else the schema's own example
// (this is what makes a oneOf body with an example on the schema work). Properties the server
// assigns (readOnly) are dropped from an object example.
func wholeBodyExample(mt *openapi3.MediaType) any {
	var ex any
	switch {
	case mt.Example != nil:
		ex = mt.Example
	case len(mt.Examples) > 0:
		names := make([]string, 0, len(mt.Examples))
		for n := range mt.Examples {
			names = append(names, n)
		}
		sort.Strings(names)
		if ref := mt.Examples[names[0]]; ref != nil && ref.Value != nil {
			ex = ref.Value.Value
		}
	}
	if ex == nil && mt.Schema != nil && mt.Schema.Value != nil {
		ex = schemaExample(mt.Schema.Value)
	}
	if obj, ok := ex.(map[string]any); ok && mt.Schema != nil && mt.Schema.Value != nil {
		cleaned := make(map[string]any, len(obj))
		for k, v := range obj {
			if ref := mt.Schema.Value.Properties[k]; ref != nil && ref.Value != nil && ref.Value.ReadOnly {
				continue
			}
			cleaned[k] = v
		}
		return cleaned
	}
	return ex
}

// ---- Swagger 2.0 ----

func stubFromSwagger2(s *swagger2Schema, defs map[string]swagger2Schema, depth int) *stubSchema {
	r := resolveSwagger2Ref(s, defs)
	if r == nil || depth > maxStubDepth {
		return &stubSchema{}
	}
	out := &stubSchema{
		Type:     r.Type,
		Required: append([]string(nil), r.Required...),
		Example:  r.Example,
		Default:  r.Default,
		Enum:     r.Enum,
		ReadOnly: r.ReadOnly,
	}
	if len(r.Properties) > 0 {
		out.Props = make(map[string]*stubSchema, len(r.Properties))
		for name, p := range r.Properties {
			rp := resolveSwagger2Ref(&p, defs)
			ps := &stubSchema{Type: rp.Type, Example: rp.Example, Default: rp.Default, Enum: rp.Enum, ReadOnly: rp.ReadOnly}
			if ps.Type == "" && len(rp.Properties) > 0 {
				ps.Type = "object"
			}
			out.Props[name] = ps
		}
		if out.Type == "" {
			out.Type = "object"
		}
	}
	if r.Items != nil {
		out.Items = stubFromSwagger2(r.Items, defs, depth+1)
		if out.Type == "" {
			out.Type = "array"
		}
	}
	for i := range r.AllOf {
		out.merge(stubFromSwagger2(&r.AllOf[i], defs, depth+1))
	}
	return out
}

func buildSwagger2BodyStub(s *swagger2Schema, defs map[string]swagger2Schema) string {
	return renderStub(stubFromSwagger2(s, defs, 0))
}
