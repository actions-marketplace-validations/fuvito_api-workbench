// Package redact decides what of a server's answer may be written into a message, a report or a
// file. The values an environment holds are masked elsewhere, by value; what only the response
// holds (a token it has just issued, someone's data) cannot be known in advance, so the rules here
// are about shape and name:
//
//   - a whole body, object or list is never quoted;
//   - a single value may be, shortened, because it is often what explains a problem;
//   - but not when its field or header is named like a credential, or it looks like a token.
//
// A password inside a longer text is masked on its own, wherever it stands (Credentials).
//
// These are rules of thumb. A secret in a field with an ordinary name, short enough not to look like
// a token, is shown if it is itself the value that broke the spec.
package redact

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
)

// Mask replaces a value that must not be shown.
const Mask = "***"

// maxValue is how much of a single text value a message quotes.
const maxValue = 80

var secretWords = map[string]bool{
	"token": true, "secret": true, "password": true, "passwd": true, "credential": true,
	"credentials": true, "bearer": true, "jwt": true, "cookie": true, "session": true,
	"authorization": true, "auth": true, "key": true, "apikey": true, "private": true, "pwd": true, "passphrase": true,
}

// moreInResponses are names that make a response value sensitive without making an environment
// variable a secret.
var moreInResponses = map[string]bool{
	"sid": true, "otp": true, "pin": true, "sig": true, "signature": true, "cvv": true, "cvc": true, "ssn": true, "iban": true,
}

// IsSecretName reports whether a variable name looks like it holds a credential (token,
// password, api key, session, …). Names are split on punctuation and camelCase, so apiKey is
// secret but authorId and monkey are not.
func IsSecretName(name string) bool {
	for _, w := range nameWords(name) {
		if secretWords[w] || strings.HasPrefix(w, "passw") {
			return true
		}
	}
	return false
}

// IsSensitiveName is IsSecretName for the name of a field or header in a response.
func IsSensitiveName(name string) bool {
	if IsSecretName(name) {
		return true
	}
	for _, w := range nameWords(name) {
		if moreInResponses[w] {
			return true
		}
	}
	return false
}

// IsSensitivePath says whether any part of the way to a value is named like a credential.
func IsSensitivePath(path []string) bool {
	for _, part := range path {
		if IsSensitiveName(part) {
			return true
		}
	}
	return false
}

func nameWords(name string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r) && len(cur) > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

var (
	reJWT    = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}(?:\.[A-Za-z0-9_-]*)?`)
	reScheme = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}`)
	reRun    = regexp.MustCompile(`[A-Za-z0-9_+=-]{20,}`)
	reUUID   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// name=value and name: value in free text, when the name says it is a credential
	rePair  = regexp.MustCompile(`([A-Za-z0-9_.-]+)(["']?\s*[:=]\s*["']?)([^\s"',;&<>}]+)`)
	reEmail = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

// looksLikeToken: long, with letters and digits mixed, and not a UUID (which is usually an id).
func looksLikeToken(run string) bool {
	if reUUID.MatchString(run) {
		return false
	}
	letters, digits := false, false
	for _, r := range run {
		letters = letters || unicode.IsLetter(r)
		digits = digits || unicode.IsDigit(r)
	}
	return letters && digits
}

// Tokens masks what looks like a credential inside a text value from a response, passwords in
// addresses and assignments included (see Credentials).
func Tokens(s string) string { return tokens(Credentials(s)) }

// LooksSecret says whether s holds something token-like, so that the whole of it is to be kept out
// of messages. A password in an address does not count: Credentials masks just the password, and
// the rest of the address is worth keeping.
func LooksSecret(s string) bool {
	s = Credentials(s)
	return tokens(s) != s
}

func tokens(s string) string {
	s = reJWT.ReplaceAllString(s, Mask)
	s = reScheme.ReplaceAllString(s, "$1 "+Mask)
	return reRun.ReplaceAllStringFunc(s, func(run string) string {
		if looksLikeToken(run) {
			return Mask
		}
		return run
	})
}

// freeText is Tokens for prose and pages of text, where a credential may also stand after its name,
// and addresses of people are masked.
func freeText(s string) string {
	s = Tokens(s)
	s = rePair.ReplaceAllStringFunc(s, func(m string) string {
		p := rePair.FindStringSubmatch(m)
		if IsSensitiveName(p[1]) && !strings.EqualFold(p[3], "bearer") && !strings.EqualFold(p[3], "basic") {
			return p[1] + p[2] + Mask
		}
		return m
	})
	return reEmail.ReplaceAllString(s, Mask)
}

func tree(v any, sensitive bool, text func(string) string) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = tree(val, sensitive || IsSensitiveName(k), text)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = tree(val, sensitive, text)
		}
		return out
	case nil:
		return nil
	}
	if sensitive {
		return Mask
	}
	if s, ok := v.(string); ok {
		return text(s)
	}
	return v
}

func decode(s string) (any, bool) {
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil || d.More() {
		return nil, false
	}
	return v, true
}

func encode(v any) string {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		return `"` + Mask + `"`
	}
	return strings.TrimSpace(b.String())
}

// Excerpt is the start of a response body that may be shown to say why a server refused a request:
// credentials, token-like texts and e-mail addresses are masked, then it is cut to max characters.
func Excerpt(body string, max int) string {
	var out string
	if v, ok := decode(body); ok {
		switch v.(type) {
		case map[string]any, []any:
			out = encode(tree(v, false, freeText))
		case string:
			// a body that is one bare text may be nothing but the credential
			out = `"` + Mask + `"`
		default:
			out = encode(v)
		}
	} else {
		out = freeText(body)
	}
	out = strings.Join(strings.Fields(out), " ")
	if r := []rune(out); len(r) > max {
		return string(r[:max]) + "…"
	}
	return out
}

// Value renders one offending value, found at path, as a message may quote it: JSON text, with
// {…} and […] standing for an object and a list. ok is false when nothing of it may be shown.
func Value(path []string, v any) (shown string, ok bool) {
	switch x := v.(type) {
	case map[string]any:
		return "{…}", true
	case []any:
		return "[…]", true
	case nil:
		return "null", true
	case string:
		if IsSensitivePath(path) || len(path) == 0 {
			return `"` + Mask + `"`, true
		}
		s := Tokens(x)
		if r := []rune(s); len(r) > maxValue {
			s = string(r[:maxValue]) + "…"
		}
		return encode(s), true
	}
	if IsSensitivePath(path) {
		return "", false
	}
	return encode(v), true
}

// Hidden returns the texts of a decoded response that must not appear anywhere in a message about
// it: what stands under a sensitive name, and token-like values wherever they are.
func Hidden(v any) []string {
	var out []string
	var walk func(v any, sensitive bool)
	walk = func(v any, sensitive bool) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				walk(val, sensitive || IsSensitiveName(k))
			}
		case []any:
			for _, val := range x {
				walk(val, sensitive)
			}
		case string:
			if sensitive || LooksSecret(x) {
				out = append(out, x)
			}
		case json.Number:
			if sensitive {
				out = append(out, x.String())
			}
		}
	}
	walk(v, false)
	return out
}
