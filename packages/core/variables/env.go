package variables

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// envRef matches ${NAME}. A preceding $ escapes it: $${NAME} is the literal text ${NAME}.
var envRef = regexp.MustCompile(`(\$?)\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ExpandEnv replaces ${NAME} references with the process environment variable of that name.
// A reference to an unset variable is left as written, so a missing variable is visible
// rather than silently empty. $${NAME} produces the literal text ${NAME}.
//
// Expansion happens when a run starts, never when an environment file is read or saved, so a
// real secret is never written back to disk.
func ExpandEnv(s string) string {
	return envRef.ReplaceAllStringFunc(s, func(match string) string {
		sub := envRef.FindStringSubmatch(match)
		if sub[1] == "$" {
			return match[1:]
		}
		if val, ok := os.LookupEnv(sub[2]); ok {
			return val
		}
		return match
	})
}

// EnvRef is an environment variable an environment file refers to but the process does not set.
type EnvRef struct {
	Name string // the process environment variable, for example API_TOKEN
	Var  string // the environment-file variable that refers to it, for example token
}

// MissingEnv lists the ${NAME} references in env whose process environment variable is not set.
func MissingEnv(env *domain.Environment) []EnvRef {
	if env == nil {
		return nil
	}
	var out []EnvRef
	seen := map[string]bool{}
	for _, v := range env.Variables {
		for _, m := range envRef.FindAllStringSubmatch(v.Value, -1) {
			if m[1] == "$" {
				continue
			}
			if _, ok := os.LookupEnv(m[2]); ok || seen[m[2]+"\x00"+v.Name] {
				continue
			}
			seen[m[2]+"\x00"+v.Name] = true
			out = append(out, EnvRef{Name: m[2], Var: v.Name})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Var < out[j].Var
	})
	return out
}

// SecretValues returns the resolved values of the variables marked secret, so reports can
// keep them out of their output.
func SecretValues(env *domain.Environment) []string {
	if env == nil {
		return nil
	}
	var out []string
	for _, v := range env.Variables {
		if !v.Secret {
			continue
		}
		if val := strings.TrimSpace(ExpandEnv(v.Value)); val != "" {
			out = append(out, val)
		}
	}
	return out
}
