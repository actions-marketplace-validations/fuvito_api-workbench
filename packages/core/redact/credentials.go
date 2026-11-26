package redact

import (
	"regexp"
	"strings"
)

// Passwords written inside longer text: in an address, or after a name that says what they are.
// Only the password is replaced, so that the rest still says which server or setting it was.

var (
	// scheme://user:password@  (the user may be empty; the password may itself hold an "@")
	reURLPassword = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://[^\s/?#@:"'<>]*:)([^\s/?#"'<>]+)@`)
	// name=value, where the value is quoted, in braces (ODBC) or runs to a separator. "==" is a
	// comparison, not an assignment.
	reAssignment = regexp.MustCompile(`([A-Za-z0-9_.-]+)=("[^"\s;][^"\n]*"|'[^'\s;][^'\n]*'|\{[^{}\n]+\}|[^\s;&"'<>={}][^\s;&"'<>]*)`)
)

// passwordName says whether name=value gives away a credential. It is deliberately narrower than
// IsSensitiveName: this runs over any text, where "key=", "session=" or "auth=" are often harmless.
func passwordName(name string) bool {
	n := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(name))
	for _, end := range []string{"password", "passwd", "passphrase"} {
		if strings.HasSuffix(n, end) {
			return true
		}
	}
	switch n {
	case "pwd", "dbpwd", "secret", "clientsecret", "token", "accesstoken", "refreshtoken", "apikey", "xapikey", "accountkey", "sharedaccesskey":
		return true
	}
	return false
}

// notASecret: already masked, a placeholder that names the value without holding it, or a switch.
func notASecret(value string) bool {
	switch strings.ToLower(value) {
	case "", Mask, "true", "false", "null", "none":
		return true
	}
	for _, start := range []string{Mask, "{{", "${", "$(", "<", "{password}"} {
		if strings.HasPrefix(value, start) {
			return true
		}
	}
	return len(value) > 1 && value[0] == '$' && strings.ToUpper(value) == value
}

// Credentials masks the password of an address with credentials, and the value of password=, pwd=,
// token= and similar assignments (connection strings, query strings), wherever they stand in s.
// Applying it twice changes nothing more.
func Credentials(s string) string {
	if strings.Contains(s, "://") {
		s = reURLPassword.ReplaceAllStringFunc(s, func(m string) string {
			p := reURLPassword.FindStringSubmatch(m)
			if notASecret(p[2]) {
				return m
			}
			return p[1] + Mask + "@"
		})
	}
	if !strings.Contains(s, "=") {
		return s
	}
	return reAssignment.ReplaceAllStringFunc(s, func(m string) string {
		p := reAssignment.FindStringSubmatch(m)
		name, value := p[1], p[2]
		// PWD and OLDPWD are the shell's working directories
		if !passwordName(name) || (name == "PWD" && strings.HasPrefix(value, "/")) {
			return m
		}
		switch value[0] {
		case '"', '\'':
			if notASecret(value[1 : len(value)-1]) {
				return m
			}
			return name + "=" + value[:1] + Mask + value[:1]
		}
		if notASecret(value) {
			return m
		}
		// punctuation that ends a sentence or closes a bracket stays
		kept := strings.TrimRight(value, `.,:)]\`)
		if kept == "" || notASecret(kept) {
			return m
		}
		return name + "=" + Mask + value[len(kept):]
	})
}
