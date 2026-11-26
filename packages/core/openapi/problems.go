package openapi

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// With OpenAPI 3.1 the validation library reports a body as one tree of errors, collected while
// walking Go maps: one message could hold several problems, in an order that changed from run to
// run. Reports then differed between identical runs, and a baseline could not tell the problems
// apart. Here every independent problem becomes a message of its own, in a fixed order.

// groupStart opens the library's message for an error that has causes.
const groupStart = "validation failed due to: "

var reExtraNames = regexp.MustCompile(`(additional properties )('[^\n]*')( not allowed)`)

// problems splits a schema error into the messages of its independent problems, sorted.
func problems(err error) []string {
	var out []string
	var walk func(error)
	walk = func(err error) {
		if list, ok := err.(openapi3.MultiError); ok {
			for _, e := range list {
				walk(e)
			}
			return
		}
		if se, ok := err.(*openapi3.SchemaError); ok {
			// these only collect their causes, which are separate problems. The causes of anyOf,
			// oneOf and the like together explain one problem and stay together.
			if causes := causesOf(se); causes != nil && (se.SchemaField == "" || se.SchemaField == "allOf" || se.SchemaField == "$ref") {
				for _, e := range causes {
					walk(e)
				}
				return
			}
		}
		out = append(out, ordered(err))
	}
	walk(err)
	sort.Strings(out)
	return out
}

// causesOf returns the causes of an error of the 3.1 validator, or nil. Errors of the 3.0
// validator are not built this way and are left alone.
func causesOf(se *openapi3.SchemaError) openapi3.MultiError {
	var list openapi3.MultiError
	if se.Origin == nil || !strings.HasPrefix(se.Origin.Error(), groupStart) || !errors.As(se.Origin, &list) {
		return nil
	}
	return list
}

// ordered is the message of one problem, with whatever it lists in a fixed order.
func ordered(err error) string {
	if se, ok := err.(*openapi3.SchemaError); ok {
		if causes := causesOf(se); causes != nil {
			parts := make([]string, len(causes))
			for i, c := range causes {
				parts[i] = ordered(c)
			}
			sort.Strings(parts)
			return groupStart + strings.Join(parts, " | ")
		}
	}
	msg := err.Error()
	// the names of properties that are not allowed come in map order too
	first, rest, cut := strings.Cut(msg, "\n")
	first = reExtraNames.ReplaceAllStringFunc(first, func(m string) string {
		p := reExtraNames.FindStringSubmatch(m)
		names := strings.Split(strings.Trim(p[2], "'"), "', '")
		sort.Strings(names)
		return p[1] + "'" + strings.Join(names, "', '") + "'" + p[3]
	})
	if cut {
		return first + "\n" + rest
	}
	return first
}
