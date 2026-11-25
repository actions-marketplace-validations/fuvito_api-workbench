package assertions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/redact"
)

// Evaluator checks domain.Assertions against domain.Responses.
type Evaluator struct {
	spec *openapi.Spec // nil if no spec loaded; used for operationId-based schema assertions
}

func New() *Evaluator { return &Evaluator{} }

// NewWithSpec creates an Evaluator that can resolve schema assertions by operationId.
func NewWithSpec(spec *openapi.Spec) *Evaluator { return &Evaluator{spec: spec} }

// Evaluate checks a single assertion against the response and returns the result.
func (e *Evaluator) Evaluate(a domain.Assertion, resp *domain.Response) domain.AssertionResult {
	res := domain.AssertionResult{Assertion: a}

	switch a.Type {
	case domain.AssertionStatus:
		res.Actual, res.Passed, res.Message = evalStatus(a, resp)
	case domain.AssertionHeader:
		res.Actual, res.Passed, res.Message = evalHeader(a, resp)
	case domain.AssertionJSONPath:
		res.Actual, res.Passed, res.Message = evalJSONPath(a, resp)
	case domain.AssertionSchema:
		res.Actual, res.Passed, res.Message = e.evalSchema(a, resp)
	case domain.AssertionDuration:
		res.Actual, res.Passed, res.Message = evalDuration(a, resp)
	default:
		res.Passed = false
		res.Message = fmt.Sprintf("unknown assertion type %q", a.Type)
	}
	return res
}

// EvaluateAll runs every assertion in the slice and returns all results.
func (e *Evaluator) EvaluateAll(assertions []domain.Assertion, resp *domain.Response) []domain.AssertionResult {
	results := make([]domain.AssertionResult, len(assertions))
	for i, a := range assertions {
		results[i] = e.Evaluate(a, resp)
	}
	return results
}

// ---- per-type evaluators ----

func evalStatus(a domain.Assertion, resp *domain.Response) (actual string, passed bool, msg string) {
	actual = strconv.Itoa(resp.StatusCode)
	expected := expectedStr(a.Expected)
	passed, msg = compare(a.Operator, actual, expected)
	return
}

func evalHeader(a domain.Assertion, resp *domain.Response) (actual string, passed bool, msg string) {
	actual, _ = resp.HeaderValue(a.Path)
	expected := expectedStr(a.Expected)
	passed, msg = compare(a.Operator, actual, expected)
	actual, msg = reportable(a.Operator, actual, expected, msg, safeHeader(a.Path, actual))
	return
}

// The comparison uses the real value. What is kept of it (the result's actual value and the
// message, which reach reports, history and the desktop) follows the rules of package redact.
func reportable(op domain.Operator, actual, expected, msg, safe string) (string, string) {
	if safe == actual {
		return actual, msg
	}
	if msg != "" {
		if _, again := compare(op, safe, expected); again != "" {
			msg = again
		} else {
			msg = fmt.Sprintf("expected %q %s %q; the value is not shown", expected, opLabel(op), expected)
		}
		msg = strings.ReplaceAll(msg, actual, redact.Mask)
	}
	return safe, msg
}

func safeHeader(name, value string) string {
	if value == "" {
		return value
	}
	if redact.IsSensitiveName(name) {
		return redact.Mask
	}
	return redact.Tokens(value)
}

// safeJSONValue is a value found by a JSONPath expression as a result may keep it.
func safeJSONValue(path string, val any, formatted string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool { return strings.ContainsRune("$.[]'\"*?()@ ", r) })
	if redact.IsSensitivePath(parts) {
		if formatted == "" {
			return formatted
		}
		return redact.Mask
	}
	switch val.(type) {
	case map[string]any:
		return "{…}" // a whole object or list is never quoted
	case []any:
		return "[…]"
	case string:
		return redact.Tokens(formatted)
	}
	return formatted
}

// evalDuration compares the response time of the request (milliseconds, from sending it to the end
// of the body) with the expected limit. With a retry it is the attempt that produced the response.
func evalDuration(a domain.Assertion, resp *domain.Response) (actual string, passed bool, msg string) {
	ms := resp.Duration.Milliseconds()
	actual = strconv.FormatInt(ms, 10)
	expected := expectedStr(a.Expected)
	passed, msg = compare(a.Operator, actual, expected)
	if !passed && msg != "" && !strings.HasPrefix(msg, "invalid") {
		op := a.Operator
		if op == "" {
			op = domain.OperatorEq
		}
		msg = fmt.Sprintf("the response took %d ms; expected %s %s ms", ms, opLabel(op), expected)
	}
	return
}

func evalJSONPath(a domain.Assertion, resp *domain.Response) (actual string, passed bool, msg string) {
	body := resp.BodyString()
	if body == "" {
		return "", false, "response body is empty"
	}
	data, err := DecodeJSON(body)
	if err != nil {
		return "", false, fmt.Sprintf("response body is not valid JSON: %v", err)
	}
	val, err := jsonpath.Get(a.Path, data)
	if err != nil {
		return "", false, fmt.Sprintf("jsonpath %q: %v", a.Path, err)
	}
	actual = FormatValue(val)
	expected := expectedStr(a.Expected)
	passed, msg = compare(a.Operator, actual, expected)
	actual, msg = reportable(a.Operator, actual, expected, msg, safeJSONValue(a.Path, val, actual))
	return
}

func (ev *Evaluator) evalSchema(a domain.Assertion, resp *domain.Response) (actual string, passed bool, msg string) {
	if a.Schema == "" {
		return "", false, "schema assertion requires a schema value"
	}
	body := resp.BodyString()
	if body == "" {
		return "", false, "response body is empty"
	}

	// If the schema value looks like an operationId (no braces, colons, or newlines),
	// validate against the spec's declared response schema for that operation.
	if looksLikeOperationID(a.Schema) {
		if ev.spec == nil {
			return "", false, fmt.Sprintf("schema assertion: %q looks like an operationId but no specFile is loaded — add specFile to the scenario or use an inline JSON/YAML schema", a.Schema)
		}
		errs := openapi.ValidateResponse(ev.spec, a.Schema, resp)
		if len(errs) == 0 {
			return "", true, ""
		}
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return "", false, strings.Join(msgs, "; ")
	}

	// Inline schema (JSON object or YAML fragment) — wrap it in a minimal spec
	// so kin-openapi can validate it.
	specYAML := []byte(`
openapi: "3.0.3"
info:
  title: inline
  version: "1"
paths:
  /validate:
    get:
      operationId: validate
      responses:
        "200":
          content:
            application/json:
              schema: ` + inlineSchemaYAML(a.Schema) + `
          description: ok
`)
	inlineSpec, err := openapi.ParseBytes(specYAML, "yaml")
	if err != nil {
		return "", false, fmt.Sprintf("invalid schema: %v", err)
	}

	fakeResp := &domain.Response{
		StatusCode: resp.StatusCode,
		Headers:    []domain.Header{{Name: "Content-Type", Value: "application/json"}},
		Body:       resp.Body,
	}

	errs := openapi.ValidateResponse(inlineSpec, "validate", fakeResp)
	if len(errs) == 0 {
		return "", true, ""
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return "", false, strings.Join(msgs, "; ")
}

// looksLikeOperationID returns true when s could be an OpenAPI operationId rather
// than an inline YAML/JSON schema fragment. Inline schemas contain braces, colons,
// or newlines; operationIds are simple identifiers that contain none of those.
func looksLikeOperationID(s string) bool {
	return !strings.ContainsAny(s, "{\n\r:")
}

func inlineSchemaYAML(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return s
	}
	if s[0] == '{' {
		// JSON schema: collapse to a single line so it embeds as an inline
		// YAML mapping (JSON is valid YAML; a block scalar would be a string).
		return strings.NewReplacer("\n", "", "\r", "").Replace(s)
	}
	// YAML schema: indent every line to sit under the `schema:` key (14-space
	// indent level in the wrapper spec) so the block parses as a mapping.
	const indent = "                " // 16 spaces
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		b.WriteString("\n" + indent + strings.TrimLeft(l, " \t"))
	}
	return b.String()
}

// ---- comparison engine ----

func compare(op domain.Operator, actual, expected string) (passed bool, msg string) {
	switch op {
	case domain.OperatorEq, "":
		passed = actual == expected
	case domain.OperatorNe:
		passed = actual != expected
	case domain.OperatorContains:
		passed = strings.Contains(actual, expected)
	case domain.OperatorMatches:
		re, err := regexp.Compile(expected)
		if err != nil {
			return false, fmt.Sprintf("invalid regex %q: %v", expected, err)
		}
		passed = re.MatchString(actual)
	case domain.OperatorGt:
		a, b, err := parseFloats(actual, expected)
		if err != nil {
			return false, err.Error()
		}
		passed = a > b
	case domain.OperatorLt:
		a, b, err := parseFloats(actual, expected)
		if err != nil {
			return false, err.Error()
		}
		passed = a < b
	default:
		return false, fmt.Sprintf("unknown operator %q", op)
	}

	if !passed {
		msg = fmt.Sprintf("expected %q %s %q; got %q", expected, opLabel(op), expected, actual)
		// cleaner form
		switch op {
		case domain.OperatorEq, "":
			msg = fmt.Sprintf("expected %q; got %q", expected, actual)
		case domain.OperatorNe:
			msg = fmt.Sprintf("expected value != %q; got %q", expected, actual)
		case domain.OperatorContains:
			msg = fmt.Sprintf("expected value to contain %q; got %q", expected, actual)
		case domain.OperatorMatches:
			msg = fmt.Sprintf("expected value to match %q; got %q", expected, actual)
		case domain.OperatorGt:
			msg = fmt.Sprintf("expected value > %q; got %q", expected, actual)
		case domain.OperatorLt:
			msg = fmt.Sprintf("expected value < %q; got %q", expected, actual)
		}
	}
	return
}

func opLabel(op domain.Operator) string {
	if op == "" {
		return "eq"
	}
	return string(op)
}

func parseFloats(a, b string) (float64, float64, error) {
	fa, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot compare %q as a number: %v", a, err)
	}
	fb, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot compare %q as a number: %v", b, err)
	}
	return fa, fb, nil
}

func expectedStr(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
