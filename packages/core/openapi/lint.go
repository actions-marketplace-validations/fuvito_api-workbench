package openapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Severity of a finding in the spec itself.
const (
	SpecError   = "error"   // conformance results for the operation cannot be trusted
	SpecWarning = "warning" // worth fixing; nothing breaks
)

// SpecIssue is a problem in the spec document, not in the server: the response checks blame the
// server for what the spec got wrong, so these are worth fixing first.
type SpecIssue struct {
	Check    string // CS-SPEC-001 ...
	Severity string
	// Operation is "METHOD /path" when the problem belongs to one operation, empty otherwise.
	Operation string
	// Where says where in the spec: the operation, a schema, a parameter.
	Where string
	What  string
	Why   string
	Fix   string
}

// Lint reports what is wrong with the spec as a document. It reads only the spec. Problems the
// OpenAPI parser already refuses (a duplicate operationId, a path parameter that is not declared)
// never get this far: such a spec does not load.
func (s *Spec) Lint() []SpecIssue {
	var out []SpecIssue
	add := func(i SpecIssue) { out = append(out, i) }

	type schemaAt struct {
		schema *openapi3.Schema
		where  string
		op     string
	}
	var schemas []schemaAt

	paths := s.doc.Paths.Map()
	pathKeys := make([]string, 0, len(paths))
	for p := range paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)
	for _, path := range pathKeys {
		item := paths[path]
		ops := item.Operations()
		methods := make([]string, 0, len(ops))
		for m := range ops {
			methods = append(methods, m)
		}
		sort.Strings(methods)
		for _, method := range methods {
			op := ops[method]
			key := OperationKey(method, path)

			if op.OperationID == "" {
				add(SpecIssue{Check: "CS-SPEC-001", Severity: SpecWarning, Operation: key, Where: key,
					What: key + " has no operationId.",
					Why:  "Reports, coverage and the schema assertion name operations by operationId; without one they fall back to the method and path.",
					Fix:  "Add a unique operationId to the operation."})
			}

			if op.RequestBody != nil && op.RequestBody.Value != nil {
				for _, mt := range sortedMedia(op.RequestBody.Value.Content) {
					if mt.Schema != nil && mt.Schema.Value != nil {
						schemas = append(schemas, schemaAt{mt.Schema.Value, key + " request body", key})
					}
				}
			}
			if op.Responses != nil {
				codes := op.Responses.Map()
				statuses := make([]string, 0, len(codes))
				for c := range codes {
					statuses = append(statuses, c)
				}
				sort.Strings(statuses)
				for _, code := range statuses {
					resp := codes[code]
					if resp == nil || resp.Value == nil {
						continue
					}
					for _, mt := range sortedMedia(resp.Value.Content) {
						if mt.Schema != nil && mt.Schema.Value != nil {
							schemas = append(schemas, schemaAt{mt.Schema.Value, key + " response " + code, key})
						}
					}
				}
			}
		}
	}

	if s.doc.Components != nil {
		names := make([]string, 0, len(s.doc.Components.Schemas))
		for n := range s.doc.Components.Schemas {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if ref := s.doc.Components.Schemas[n]; ref != nil && ref.Value != nil {
				schemas = append(schemas, schemaAt{ref.Value, "components.schemas." + n, ""})
			}
		}
	}
	seen := map[*openapi3.Schema]bool{}
	var walk func(sch *openapi3.Schema, where, op string)
	walk = func(sch *openapi3.Schema, where, op string) {
		if sch == nil || seen[sch] {
			return
		}
		seen[sch] = true
		if len(sch.Properties) > 0 && len(sch.Required) > 0 && len(sch.AllOf)+len(sch.OneOf)+len(sch.AnyOf) == 0 &&
			sch.AdditionalProperties.Has == nil && sch.AdditionalProperties.Schema == nil {
			for _, req := range sch.Required {
				if _, ok := sch.Properties[req]; !ok {
					add(SpecIssue{Check: "CS-SPEC-002", Severity: SpecError, Operation: op, Where: where,
						What: fmt.Sprintf("%s lists %q as required, but declares no property of that name.", where, req),
						Why:  "A response can never satisfy it (or never be checked for it), so a correct server fails or a field goes unchecked.",
						Fix:  fmt.Sprintf("Declare the property %s, or remove it from required.", req)})
				}
			}
		}
		names := make([]string, 0, len(sch.Properties))
		for n := range sch.Properties {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if p := sch.Properties[n]; p != nil {
				walk(p.Value, where+"."+n, op)
			}
		}
		if sch.Items != nil {
			walk(sch.Items.Value, where+"[]", op)
		}
	}
	for _, sa := range schemas {
		walk(sa.schema, sa.where, sa.op)
	}

	if s.ignoredExample != "" {
		add(SpecIssue{Check: "CS-SPEC-003", Severity: SpecWarning, Where: "an example in the spec",
			What: "An example in the spec does not match its own format.",
			Why:  "Examples are documentation and test data; one that breaks its schema misleads readers. The spec still loads, and none of its examples were checked.",
			Fix:  "Correct the example. The first problem found: " + firstLine(s.ignoredExample)})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Check != out[j].Check {
			return out[i].Check < out[j].Check
		}
		return out[i].Where < out[j].Where
	})
	return out
}

func sortedMedia(c openapi3.Content) []*openapi3.MediaType {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*openapi3.MediaType, 0, len(keys))
	for _, k := range keys {
		out = append(out, c[k])
	}
	return out
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
