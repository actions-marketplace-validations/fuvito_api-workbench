package conformance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Finding kinds, from the most to the least specific.
const (
	FindingMissingField = "missing-field" // the response omits a field the spec requires
	FindingWrongType    = "wrong-type"    // a field has a different type than the spec declares
	FindingMissingValue = "missing-value" // the request was not sent: a value it needs is missing
	FindingRequest      = "request"       // the request built from the spec was itself invalid
	FindingNoResponse   = "no-response"   // the server never answered
	FindingOther        = "other"         // a violation this package does not recognise
)

// Finding explains one violation in plain language: where it is, what the spec wanted, what the
// server sent, why that counts as a failure and what to change.
type Finding struct {
	Kind     string
	Where    string // the field, for example "price" or "[0].id"; empty when not about a field
	Summary  string // one line, for example `"price" is text, the spec says number`
	Expected string
	Got      string
	Why      string
	Fix      string
	Detail   string // the validator's own message, unchanged
}

var (
	reMissing   = regexp.MustCompile(`Error at "([^"]*)": property "([^"]+)" is missing`)
	reWrongType = regexp.MustCompile(`Error at "([^"]*)": value must be (an? [A-Za-z ]+?)(?:\n|$)`)
	reGotValue  = regexp.MustCompile(`(?s)\nValue:\s*\n\s*(.+?)\s*$`)
	reAtPath    = regexp.MustCompile(`Error at "([^"]*)": ([^\n]+)`)
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
			Summary: "The server did not answer",
			Got:     op.Error,
			Why:     "No response arrived, so the operation could not be compared with the spec.",
			Fix:     "Check that the server is running, that the base URL is right, and that the request timeout is long enough.",
			Detail:  op.Error,
		})
	}
	for _, v := range op.RequestViolations {
		out = append(out, diagnoseRequest(v.Error()))
	}
	for _, v := range op.Violations {
		out = append(out, diagnoseResponse(v.Error()))
	}
	return out
}

func diagnoseRequest(msg string) Finding {
	return Finding{
		Kind:    FindingRequest,
		Summary: "The request built from the spec was not valid",
		Got:     firstLine(msg),
		Why:     "The values clrsig put in the request (path parameters, body fields) do not fit the spec, so the server was not tested fairly.",
		Fix:     "Give the missing values in the environment, for example the path parameter used as {{id}}.",
		Detail:  msg,
	}
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
			Kind:    FindingMissingValue,
			Summary: "The request was not sent",
			Got:     op.Blocked,
			Why:     "A value the request needs could not be produced, so it was not sent and nothing was compared with the spec.",
			Fix:     "Correct the dynamic value named above in the environment or the request.",
			Detail:  op.Blocked,
		}
	}
	return Finding{
		Kind:    FindingMissingValue,
		Where:   strings.Join(op.Missing, ", "),
		Summary: "No value for " + strings.Join(refs, ", "),
		Got:     "not sent",
		Why: "The request needs " + strings.Join(refs, ", ") + " (a path parameter, a credential or a body field taken from the spec), " +
			"but neither the environment nor the variable overrides provide it. It was not sent, so nothing was compared with the spec.",
		Fix:    "Add " + strings.Join(op.Missing, ", ") + " to the environment, or give it as an override (" + strings.Join(overrides, ", ") + ").",
		Detail: op.Blocked,
	}
}
