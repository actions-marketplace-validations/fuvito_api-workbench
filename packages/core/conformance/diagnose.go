package conformance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// Finding kinds, from the most to the least specific.
const (
	FindingMissingField     = "missing-field"        // the response omits a field the spec requires
	FindingWrongType        = "wrong-type"           // a field has a different type than the spec declares
	FindingUndeclaredStatus = "undeclared-status"    // the status code is not one the spec declares
	FindingWrongContentType = "wrong-content-type"   // the Content-Type is not one the spec declares
	FindingMissingHeader    = "missing-header"       // a header the spec requires was not sent
	FindingBadHeader        = "bad-header"           // a declared header has a value that does not fit
	FindingMissingValue     = "missing-value"        // the request was not sent: a value it needs is missing
	FindingRequest          = "request"              // the request built from the spec was itself invalid
	FindingNoResponse       = "no-response"          // the server never answered
	FindingDynamicValue     = "dynamic-value-failed" // the request was not sent: a {{$function}} failed
	FindingNoBranch         = "no-branch-matched"    // the body fits none (or several) of the oneOf / anyOf shapes
	FindingInvalidBody      = "invalid-body"         // the response body is not valid JSON
	FindingEmptyBody        = "empty-body"           // the response body is empty where the spec declares a schema
	FindingBodyTooLarge     = "body-too-large"       // the response is longer than the read limit
	FindingConstraint       = "constraint"           // a value breaks an enum, range, length, pattern or format
	FindingExtraProperty    = "unexpected-property"  // the body has a property the schema does not allow
	FindingOther            = "other"                // a violation this package does not recognise
)

// Finding explains one violation in plain language, so the person reading it can find the problem,
// understand it and fix it: what is wrong (Summary, Expected, Got), where it is (Part and Where in
// the response, Spec in the spec), why that counts as a failure (Why) and what to change (Fix).
type Finding struct {
	Kind string
	// CheckID is the ID of the check in the catalog of what Clear Signal checks (Help, What We
	// Check), for example CS-RESP-005. Empty for a violation this package does not recognise.
	CheckID string
	// Part is the part of the exchange the finding is about: "response body", "response status",
	// "response header" (also used for Content-Type), "request" or "network".
	Part string
	// Where narrows Part: the field for a body ("price", "[0].id"), the header name, the status code.
	Where string
	// Spec is the place in the spec the expectation comes from, for example
	// "GET /products/{id} → response 200 → headers → X-Total-Count".
	Spec     string
	Summary  string // one line, for example `"price" is text, the spec says number`
	Expected string
	Got      string
	Why      string
	Fix      string
	Detail   string // the validator's own message, unchanged
}

// WhereText says where the finding is, in words: `response header "X-Total-Count"`,
// `response body, field price`, `response status 500`.
func (f Finding) WhereText() string {
	switch {
	case f.Part == "" && f.Where == "":
		return ""
	case f.Where == "":
		return f.Part
	case f.Part == "response body":
		return f.Part + ", field " + f.Where
	case f.Part == "response header":
		return f.Part + " " + strconv.Quote(f.Where)
	}
	return f.Part + " " + f.Where
}

var (
	reMissing    = regexp.MustCompile(`Error at "([^"]*)": property "([^"]+)" is missing`)
	reWrongType  = regexp.MustCompile(`(?:Error at "([^"]*)": )?value must be (an? [A-Za-z ]+?)(?:\n|$)`)
	reConstraint = regexp.MustCompile(`(?i)allowed values|must be (at least|at most|more than|less than|a multiple)|minimum|maximum|doesn't match the (regular expression|format|pattern)|duplicate items|not nullable|exclusive`)
	reExtra      = regexp.MustCompile(`property "([^"]+)" is unsupported`)
	reGotValue   = regexp.MustCompile(`(?s)\nValue:\s*\n\s*(.+?)\s*$`)
	reAtPath     = regexp.MustCompile(`Error at "([^"]*)": ([^\n]+)`)
)

// Diagnose turns an operation's failure into findings. A passing or skipped operation has none.
func Diagnose(op OperationResult) []Finding {
	if op.Skipped {
		return nil
	}
	var out []Finding
	if op.Blocked != "" {
		out = append(out, blockedFinding(op))
	}
	if op.Error != "" {
		out = append(out, Finding{
			Kind:    FindingNoResponse,
			Part:    "network",
			Summary: "The server did not answer",
			Got:     op.Error,
			Why:     "No response arrived, so the operation could not be compared with the spec.",
			Fix:     "Check that the server is running, that the base URL is right, and that the request timeout is long enough.",
			Detail:  op.Error,
		})
	}
	for _, v := range op.RequestViolations {
		out = append(out, diagnoseRequest(op, v))
	}
	for _, v := range op.Violations {
		out = append(out, diagnoseViolation(op, v))
	}
	for i := range out {
		out[i].CheckID = checkIDFor(out[i])
	}
	return out
}

// checkIDFor maps a finding to its ID in the catalog of checks.
func checkIDFor(f Finding) string {
	switch f.Kind {
	case FindingUndeclaredStatus:
		return "CS-RESP-001"
	case FindingWrongContentType:
		return "CS-RESP-002"
	case FindingMissingHeader:
		return "CS-RESP-003"
	case FindingBadHeader:
		return "CS-RESP-004"
	case FindingMissingField:
		return "CS-RESP-005"
	case FindingWrongType:
		return "CS-RESP-006"
	case FindingConstraint:
		return "CS-RESP-007"
	case FindingInvalidBody, FindingEmptyBody:
		return "CS-RESP-008"
	case FindingNoBranch:
		return "CS-RESP-009"
	case FindingExtraProperty:
		return "CS-RESP-010"
	case FindingBodyTooLarge:
		return "CS-RESP-013"
	case FindingMissingValue:
		return "CS-RUN-001"
	case FindingDynamicValue:
		return "CS-RUN-002"
	case FindingNoResponse:
		return "CS-RUN-003"
	case FindingRequest:
		switch {
		case strings.HasPrefix(f.Part, "request body"):
			return "CS-REQ-002"
		case strings.Contains(f.Part, "parameter"):
			return "CS-REQ-001"
		}
	}
	return ""
}

// specLocation names where in the spec an expectation lives: the operation, then the part.
func specLocation(op OperationResult, rest string) string {
	path := op.SpecPath
	if path == "" {
		path = op.Path
	}
	loc := op.Method + " " + path
	if rest != "" {
		loc += " → " + rest
	}
	return loc
}

var (
	reDeclaresOnly = regexp.MustCompile(`but the spec declares only (.+)$`)
	reDeclares     = regexp.MustCompile(`but the spec declares (.+?) for this response`)
)

func lastGroup(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// diagnoseViolation explains one response violation. The kinds the validator classifies carry the
// header name, status or content type; anything else is matched on its message.
func diagnoseViolation(op OperationResult, v openapi.ValidationError) Finding {
	status := strconv.Itoa(op.StatusCode)
	msg := v.Error()
	switch v.Kind {
	case openapi.KindUndeclaredStatus:
		declared := lastGroup(reDeclaresOnly, msg)
		return Finding{
			Kind: FindingUndeclaredStatus, Part: "response status", Where: v.Subject,
			Spec:     specLocation(op, "responses"),
			Summary:  fmt.Sprintf("Status %s is not declared for %s", v.Subject, specLocation(op, "")),
			Expected: "one of: " + declared, Got: v.Subject,
			Why:    fmt.Sprintf("The spec lists the responses %s may return (%s) and %s is not one of them: the server returned something the spec does not describe, or the spec is incomplete.", specLocation(op, ""), declared, v.Subject),
			Fix:    fmt.Sprintf("Make the server answer a declared status, or add %s (or a default response) to %s in the spec.", v.Subject, specLocation(op, "responses")),
			Detail: msg,
		}
	case openapi.KindContentType:
		declared := lastGroup(reDeclares, msg)
		return Finding{
			Kind: FindingWrongContentType, Part: "response header", Where: "Content-Type",
			Spec:     specLocation(op, "response "+status+" → content"),
			Summary:  fmt.Sprintf("Content-Type is %q, the spec says %s", v.Subject, declared),
			Expected: declared, Got: v.Subject,
			Why:    fmt.Sprintf("The spec declares the %s response body as %s, but the server labelled it %q, so the body was not compared with the schema: nothing about its fields was checked.", status, declared, v.Subject),
			Fix:    fmt.Sprintf("Make the server send Content-Type %s, or declare %q for response %s in the spec.", declared, v.Subject, status),
			Detail: msg,
		}
	case openapi.KindMissingHeader:
		return Finding{
			Kind: FindingMissingHeader, Part: "response header", Where: v.Subject,
			Spec:     specLocation(op, "response "+status+" → headers → "+v.Subject),
			Summary:  fmt.Sprintf("The required header %q was not sent", v.Subject),
			Expected: "the header " + v.Subject, Got: "not sent",
			Why:    fmt.Sprintf("The spec marks the %q header of response %s as required, but the server's response did not include it. The body was not compared with the schema.", v.Subject, status),
			Fix:    fmt.Sprintf("Send %q from the server, or make it optional (or remove it) in the spec.", v.Subject),
			Detail: msg,
		}
	case openapi.KindBadHeader:
		return Finding{
			Kind: FindingBadHeader, Part: "response header", Where: v.Subject,
			Spec:     specLocation(op, "response "+status+" → headers → "+v.Subject),
			Summary:  fmt.Sprintf("The header %q has a value that does not fit the spec", v.Subject),
			Expected: "a value matching the header's schema", Got: firstLine(strings.TrimPrefix(msg, specLocation(op, "")+" "+status+": ")),
			Why:    fmt.Sprintf("The spec gives the %q header of response %s a type, and the value the server sent is not valid for it. The body was not compared with the schema.", v.Subject, status),
			Fix:    fmt.Sprintf("Send a value of the declared type in %q, or correct the header's schema in the spec.", v.Subject),
			Detail: msg,
		}
	}
	bodySpec := specLocation(op, "response "+status+" → body schema")
	switch v.Kind {
	case openapi.KindBodyEmpty:
		return Finding{
			Kind: FindingEmptyBody, Part: "response body", Spec: bodySpec,
			Summary: "The response body is empty", Expected: "a body matching the declared schema", Got: "nothing",
			Why:    fmt.Sprintf("The spec declares a body for response %s, but the server sent none, so no field could be compared.", status),
			Fix:    "Return the body from the server, or declare response " + status + " without content in the spec.",
			Detail: msg,
		}
	case openapi.KindBodyNotJSON:
		return Finding{
			Kind: FindingInvalidBody, Part: "response body", Spec: bodySpec,
			Summary: "The response body is not valid JSON", Expected: "valid JSON", Got: lastParen(msg),
			Why:    fmt.Sprintf("The spec declares a JSON body for response %s, but what arrived cannot be parsed, so no field was compared.", status),
			Fix:    "Make the server send valid JSON (check for a truncated, HTML or plain-text error page), or correct the declared media type in the spec.",
			Detail: msg,
		}
	case openapi.KindBodyBranch:
		summary, why := "The response body fits none of the allowed shapes", "The spec allows several shapes for this body (oneOf / anyOf) and the response fits none of them."
		if v.Subject == "several" {
			summary, why = "The response body fits more than one of the allowed shapes", "The spec says the body must fit exactly one of its oneOf shapes, and this response fits several, so it is ambiguous."
		}
		return Finding{
			Kind: FindingNoBranch, Part: "response body", Spec: bodySpec,
			Summary: summary, Expected: "exactly one of the declared shapes", Got: "see the detail",
			Why:    why,
			Fix:    "Make the response match one declared shape (for a discriminator, send its property), or add the shape the server really sends to the spec.",
			Detail: msg,
		}
	case openapi.KindBodyTooLarge:
		return Finding{
			Kind: FindingBodyTooLarge, Part: "response body", Spec: bodySpec,
			Summary: "The response body is longer than Clear Signal reads", Expected: "a body within the read limit", Got: "a body cut at the limit",
			Why:    "Clear Signal reads at most 50 MB of a response; the rest was cut off, so the body could not be compared with the schema (the JSON was incomplete).",
			Fix:    "Test an operation that returns less data (a smaller page or filter), or check this operation another way.",
			Detail: msg,
		}
	}
	f := diagnoseResponse(msg)
	f.Part = "response body"
	f.Spec = bodySpec
	return f
}

func lastParen(msg string) string {
	i, j := strings.LastIndex(msg, "("), strings.LastIndex(msg, ")")
	if i >= 0 && j > i {
		return msg[i+1 : j]
	}
	return ""
}

// diagnoseRequest explains a problem in the request clrsig built from the spec. The request is
// made of the spec's own path, parameters and body with {{placeholders}} filled from the
// environment, so the fix is nearly always a value in the environment (or an override).
func diagnoseRequest(op OperationResult, v openapi.ValidationError) Finding {
	msg := v.Error()
	body := specLocation(op, "request body")
	if v.Kind == openapi.KindRequestParameter {
		detail := msg
		if i := strings.Index(msg, "does not fit the spec: "); i >= 0 {
			detail = msg[i+len("does not fit the spec: "):]
		}
		return Finding{
			Kind: FindingRequest, Part: "request " + v.In + " parameter", Where: v.Subject,
			Spec:     specLocation(op, "parameters → "+v.Subject+" (in "+v.In+")"),
			Summary:  fmt.Sprintf("The %s parameter %q does not fit the spec", v.In, v.Subject),
			Expected: "a value of the declared type", Got: detail,
			Why:    fmt.Sprintf("The request clrsig built from the spec has a %s value for %q that does not match the parameter's declared type, so the server was not tested with a valid request.", v.In, v.Subject),
			Fix:    fmt.Sprintf("Give %q a value of the declared type in the environment, or with an override (for example %s=…).", v.Subject, v.Subject),
			Detail: msg,
		}
	}
	if v.Kind == openapi.KindRequestBody && v.Variable != "" {
		want := v.Expected
		if want == "" {
			want = "a valid JSON value"
		}
		return Finding{
			Kind: FindingRequest, Part: "request body", Where: v.Subject, Spec: body,
			Summary:  fmt.Sprintf("The request body field %q takes %q from {{%s}}, which is not valid JSON", v.Subject, v.Got, v.Variable),
			Expected: want, Got: v.Got,
			Why: fmt.Sprintf("The value is put into the body as it is, so the body is not valid JSON and the server was not tested with a valid request. The spec declares %q as %s.",
				v.Subject, indefinite(strings.TrimPrefix(want, "a "))),
			Fix:    fmt.Sprintf("Set %s in the environment, or with an override, to %s (a number is written without quotes, for example %s=9.99; text needs quotes inside the value).", v.Variable, indefinite(strings.TrimPrefix(want, "a ")), v.Variable),
			Detail: msg,
		}
	}
	if m := reMissing.FindStringSubmatch(msg); m != nil {
		return Finding{
			Kind: FindingRequest, Part: "request body", Where: displayPath(m[1]), Spec: body,
			Summary:  fmt.Sprintf("The request body is missing the required field %q", m[2]),
			Expected: fmt.Sprintf("the field %q", m[2]), Got: "no such field",
			Why:    fmt.Sprintf("The spec requires %q in the request body, but the request clrsig built did not include it, so the server was not tested with a valid request.", m[2]),
			Fix:    fmt.Sprintf("Check how %q is declared in the spec; if it is a composed schema, make sure the field is listed as required there.", m[2]),
			Detail: msg,
		}
	}
	if m := reWrongType.FindStringSubmatch(msg); m != nil {
		field := displayPath(m[1])
		want := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(m[2], "an "), "a "))
		got, gotType := "", ""
		if g := reGotValue.FindStringSubmatch(msg); g != nil {
			got, gotType = g[1], jsonTypeOf(g[1])
		}
		name := field
		if name == "" {
			name = "body"
		}
		summary := fmt.Sprintf("The request sends %s as %s, the spec says %s", quoteField(name), describeGot(gotType, ""), want)
		if gotType == "" {
			summary = fmt.Sprintf("The request body field %s should be %s", quoteField(name), indefinite(want))
		}
		return Finding{
			Kind: FindingRequest, Part: "request body", Where: field, Spec: body,
			Summary: summary, Expected: want, Got: got,
			Why: fmt.Sprintf("The request clrsig built from the spec sends %s for %s, but the spec declares %s, so the server was not tested with a valid request. This usually means the value taken from the environment is not %s.",
				describeGot(gotType, got), quoteField(name), indefinite(want), indefinite(want)),
			Fix:    fmt.Sprintf("Set %s to %s in the environment, or with an override (for example %s=…).", leafName(field), indefinite(want), leafName(field)),
			Detail: msg,
		}
	}
	return Finding{
		Kind: FindingRequest, Part: "request", Spec: specLocation(op, "request"),
		Summary: "The request built from the spec was not valid",
		Got:     firstLine(msg),
		Why:     "The values clrsig put in the request (path parameters, body fields) do not fit the spec, so the server was not tested fairly.",
		Fix:     "Give the missing values in the environment, for example the path parameter used as {{id}}.",
		Detail:  msg,
	}
}

// leafName is the last part of a field path: [0].price becomes price.
func leafName(field string) string {
	if i := strings.LastIndex(field, "."); i >= 0 {
		return field[i+1:]
	}
	return strings.TrimLeft(field, "[]0123456789")
}

func diagnoseResponse(msg string) Finding {
	if m := reMissing.FindStringSubmatch(msg); m != nil {
		field := displayPath(m[1])
		if field == "" {
			field = m[2]
		}
		return Finding{
			Kind:     FindingMissingField,
			Where:    field,
			Summary:  fmt.Sprintf("%q is missing from the response", m[2]),
			Expected: fmt.Sprintf("the field %q", m[2]),
			Got:      "no such field",
			Why:      fmt.Sprintf("The spec marks %q as required, but the server's response did not include it.", m[2]),
			Fix:      fmt.Sprintf("Return %q from the server, or make it optional in the spec if it is not always present.", m[2]),
			Detail:   msg,
		}
	}
	if m := reWrongType.FindStringSubmatch(msg); m != nil {
		field := displayPath(m[1])
		want := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(m[2], "an "), "a "))
		got := ""
		gotType := ""
		if g := reGotValue.FindStringSubmatch(msg); g != nil {
			got = g[1]
			gotType = jsonTypeOf(got)
		}
		label := field
		if label == "" {
			label = "the response body"
		}
		summary := fmt.Sprintf("%s should be %s", quoteField(label), indefinite(want))
		if gotType != "" {
			summary = fmt.Sprintf("%s is %s, the spec says %s", quoteField(label), gotType, want)
		}
		return Finding{
			Kind:     FindingWrongType,
			Where:    field,
			Summary:  summary,
			Expected: want,
			Got:      got,
			Why:      fmt.Sprintf("The server sent %s where the spec declares %s.", describeGot(gotType, got), indefinite(want)),
			Fix:      fmt.Sprintf("Send %s from the server (for example without quotes for a number), or change the type in the spec if the server is right.", indefinite(want)),
			Detail:   msg,
		}
	}
	if m := reExtra.FindStringSubmatch(msg); m != nil {
		return Finding{
			Kind: FindingExtraProperty, Where: m[1],
			Summary: fmt.Sprintf("%q is not allowed in the response", m[1]),
			Why:     fmt.Sprintf("The schema forbids properties it does not declare (additionalProperties: false) and the response has %q.", m[1]),
			Fix:     fmt.Sprintf("Stop sending %q, or declare it in the schema.", m[1]),
			Detail:  msg,
		}
	}
	if m := reAtPath.FindStringSubmatch(msg); m != nil && reConstraint.MatchString(m[2]) {
		field := displayPath(m[1])
		return Finding{
			Kind: FindingConstraint, Where: field,
			Summary: strings.TrimSpace(m[2]),
			Why:     "A value breaks a rule the schema sets (an allowed set, a range, a length, a pattern or a format).",
			Fix:     "Send a value the rule allows, or relax the rule in the spec if the server is right.",
			Detail:  msg,
		}
	}
	if m := reAtPath.FindStringSubmatch(msg); m != nil {
		field := displayPath(m[1])
		return Finding{
			Kind:    FindingOther,
			Where:   field,
			Summary: strings.TrimSpace(m[2]),
			Why:     "The response does not match what the spec declares.",
			Fix:     "Compare the response with the schema in the spec and change whichever is wrong.",
			Detail:  msg,
		}
	}
	return Finding{
		Kind:    FindingOther,
		Summary: firstLine(msg),
		Why:     "The response does not match what the spec declares.",
		Fix:     "Compare the response with the spec and change whichever is wrong.",
		Detail:  msg,
	}
}

// displayPath turns a JSON pointer such as /0/price or /items/2/id into 0.price style text:
// [0].price and items[2].id.
func displayPath(pointer string) string {
	var b strings.Builder
	for _, seg := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		if seg == "" {
			continue
		}
		if _, err := strconv.Atoi(seg); err == nil {
			b.WriteString("[" + seg + "]")
			continue
		}
		if b.Len() > 0 {
			b.WriteString(".")
		}
		b.WriteString(seg)
	}
	return b.String()
}

func quoteField(s string) string {
	if strings.Contains(s, " ") {
		return s
	}
	return strconv.Quote(s)
}

// jsonTypeOf names the JSON type of a value as the validator printed it.
func jsonTypeOf(v string) string {
	v = strings.TrimSpace(v)
	switch {
	case strings.HasPrefix(v, `"`):
		return "text"
	case v == "true" || v == "false":
		return "true/false"
	case v == "null":
		return "null"
	case strings.HasPrefix(v, "["):
		return "a list"
	case strings.HasPrefix(v, "{"):
		return "an object"
	case v == "":
		return ""
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return "a number"
	}
	return ""
}

func describeGot(gotType, got string) string {
	switch {
	case gotType == "":
		return "a value of another type"
	case got == "" || len(got) > 40:
		return gotType
	}
	return fmt.Sprintf("%s (%s)", gotType, got)
}

func indefinite(noun string) string {
	if strings.HasPrefix(noun, "a ") || strings.HasPrefix(noun, "an ") {
		return noun
	}
	if strings.ContainsAny(noun[:1], "aeiou") {
		return "an " + noun
	}
	return "a " + noun
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func blockedFinding(op OperationResult) Finding {
	refs := make([]string, len(op.Missing))
	overrides := make([]string, len(op.Missing))
	for i, n := range op.Missing {
		refs[i] = "{{" + n + "}}"
		overrides[i] = n + "=…"
	}
	if len(op.Missing) == 0 {
		return Finding{
			Kind:    FindingDynamicValue,
			Part:    "request",
			Summary: "The request was not sent",
			Got:     op.Blocked,
			Why:     "A value the request needs could not be produced, so it was not sent and nothing was compared with the spec.",
			Fix:     "Correct the dynamic value named above in the environment or the request.",
			Detail:  op.Blocked,
		}
	}
	return Finding{
		Kind:    FindingMissingValue,
		Part:    "request",
		Where:   strings.Join(op.Missing, ", "),
		Summary: "No value for " + strings.Join(refs, ", "),
		Got:     "not sent",
		Why: "The request needs " + strings.Join(refs, ", ") + " (a path parameter, a credential or a body field taken from the spec), " +
			"but neither the environment nor the variable overrides provide it. It was not sent, so nothing was compared with the spec.",
		Fix:    "Add " + strings.Join(op.Missing, ", ") + " to the environment, or give it as an override (" + strings.Join(overrides, ", ") + ").",
		Detail: op.Blocked,
	}
}
