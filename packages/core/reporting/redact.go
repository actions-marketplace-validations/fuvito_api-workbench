package reporting

import (
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/redact"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// Redacted replaces a value that must not appear in a report.
const Redacted = "***"

// minSecretLen is the shortest secret value replaced inside free text, so a short secret such
// as "1" cannot corrupt ordinary words and numbers.
const minSecretLen = 4

// IsSecretName reports whether a variable name looks like it holds a credential (token,
// password, api key, session, …).
func IsSecretName(name string) bool { return redact.IsSecretName(name) }

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

// text removes every known secret value from s, and any password written in an address or an
// assignment, known or not.
func (x redactor) text(s string) string {
	return redact.Credentials(x.known(s))
}

// known removes every known secret value from s.
func (x redactor) known(s string) string {
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
		case x.reveal:
			// the user asked to see extracted values
			out[name] = x.known(val)
		default:
			out[name] = x.text(val)
		}
	}
	return out
}
