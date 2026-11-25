package openapi

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/redact"
)

// The validation library ends a schema error with the whole schema and the whole value that failed,
// which for a missing field is the whole body. Those messages go into reports, CI logs, baseline
// files and the desktop app, so every violation passes through here before it leaves this package.

var (
	// the library indents both blocks by two spaces; a string value cannot hold a raw line break
	reDetails = regexp.MustCompile(`\nSchema:\n(?:  [^\n]*\n)+\n?Value:\n((?:  [^\n]*(?:\n|$))+)`)
	// "property "x" is missing" (3.0), "missing property 'x'" (3.1), as the last thing said
	reMissingField = regexp.MustCompile(`(?:property "[^"\n]*" is missing|missing propert(?:y|ies) [^\n]*)$`)
	// where the failing value is: Error at "/a/b" (3.0), at '/a/b' (3.1)
	reValuePath = regexp.MustCompile(`(?:[Ee]rror at "([^"]*)"|\bat '([^']*)')`)
)

// DetailStart is where the library's schema and value dump begins in a message. Baselines written
// by earlier versions stored messages with it.
const DetailStart = "\nSchema:\n"

func pointerParts(pointer string) []string {
	var out []string
	for _, p := range strings.Split(pointer, "/") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// withoutDetails replaces each schema-and-value dump by the failing value as it may be quoted.
func withoutDetails(msg string) string {
	var b strings.Builder
	last := 0
	for _, m := range reDetails.FindAllStringSubmatchIndex(msg, -1) {
		b.WriteString(msg[last:m[0]])
		last = m[1]
		var path []string
		if at := reValuePath.FindAllStringSubmatch(msg[:m[0]], -1); len(at) > 0 {
			final := at[len(at)-1]
			path = pointerParts(final[1] + final[2])
		}
		d := json.NewDecoder(strings.NewReader(msg[m[2]:m[3]]))
		d.UseNumber()
		var value any
		if d.Decode(&value) != nil {
			continue
		}
		// for a missing field the value is the object that lacks it, which says nothing more
		if _, isObject := value.(map[string]any); isObject && reMissingField.MatchString(msg[:m[0]]) {
			continue
		}
		if shown, ok := redact.Value(path, value); ok {
			b.WriteString("\nValue:\n  " + shown + "\n")
		}
	}
	b.WriteString(msg[last:])
	return strings.TrimRight(b.String(), "\n")
}

// hiddenIn lists the texts of a response (or of a request body) that no message about it may hold.
func hiddenIn(body []byte, headers []domain.Header) []string {
	var out []string
	d := json.NewDecoder(strings.NewReader(string(body)))
	d.UseNumber()
	var data any
	if d.Decode(&data) == nil {
		out = redact.Hidden(data)
		if s, ok := data.(string); ok {
			out = append(out, s)
		}
	}
	for _, h := range headers {
		if redact.IsSensitiveName(h.Name) {
			out = append(out, h.Value)
			// a cookie's own value, without its name and attributes
			for _, part := range strings.Split(h.Value, ";") {
				if _, v, ok := strings.Cut(part, "="); ok && strings.EqualFold(h.Name, "Set-Cookie") {
					out = append(out, v)
				}
			}
		} else if redact.Tokens(h.Value) != h.Value {
			out = append(out, h.Value)
		}
	}
	// longest first, so a text that holds a shorter one is replaced whole
	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// minHidden keeps a very short value, such as "1", from blanking out ordinary words and numbers.
const minHidden = 4

// safeViolations makes the messages of violations fit to be written anywhere.
func safeViolations(violations []ValidationError, body []byte, headers []domain.Header) []ValidationError {
	if len(violations) == 0 {
		return violations
	}
	hidden := hiddenIn(body, headers)
	for i := range violations {
		msg := withoutDetails(violations[i].Message)
		for _, h := range hidden {
			if len(h) >= minHidden {
				msg = strings.ReplaceAll(msg, h, redact.Mask)
			}
		}
		violations[i].Message = msg
	}
	return violations
}
