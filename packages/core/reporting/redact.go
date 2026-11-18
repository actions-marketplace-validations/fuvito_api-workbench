package reporting

import (
	"sort"
	"strings"
	"unicode"

	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// Redacted replaces a value that must not appear in a report.
const Redacted = "***"

// minSecretLen is the shortest secret value replaced inside free text, so a short secret such
// as "1" cannot corrupt ordinary words and numbers.
const minSecretLen = 4

var secretWords = map[string]bool{
	"token": true, "secret": true, "password": true, "passwd": true, "credential": true,
	"credentials": true, "bearer": true, "jwt": true, "cookie": true, "session": true,
	"authorization": true, "auth": true, "key": true, "apikey": true, "private": true, "pwd": true, "passphrase": true,
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

// redactor removes secrets from the text a report prints. Secret environment values are always
// removed; extracted values are removed when their variable name looks secret, unless the user
// asked to see them.
type redactor struct {
	secrets []string
	reveal  bool
}

func redactorFor(r *scenario.Result) redactor {
	return newRedactor(r.Secrets, r.RevealExtracted)
}

func newRedactor(secrets []string, reveal bool) redactor {
	x := redactor{reveal: reveal}
	for _, s := range secrets {
		if len(s) >= minSecretLen {
			x.secrets = append(x.secrets, s)
		}
	}
	// longest first, so a secret that contains another is replaced whole
	sort.Slice(x.secrets, func(i, j int) bool { return len(x.secrets[i]) > len(x.secrets[j]) })
	return x
}

// text removes every known secret value from s.
func (x redactor) text(s string) string {
	for _, secret := range x.secrets {
		if strings.Contains(s, secret) {
			s = strings.ReplaceAll(s, secret, Redacted)
		}
	}
	return s
}

// variable returns a scenario variable's value as it may appear in a report: a variable marked
// secret is never shown.
func (x redactor) variable(v scenario.ResolvedVariable) string {
	if v.Secret {
		return Redacted
	}
	return x.text(v.Value)
}

// extracted returns the extracted variables as they may appear in a report.
func (x redactor) extracted(m map[string]string) map[string]string {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for name, val := range m {
		switch {
		case !x.reveal && IsSecretName(name):
			out[name] = Redacted
		default:
			out[name] = x.text(val)
		}
	}
	return out
}
