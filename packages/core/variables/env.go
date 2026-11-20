package variables

import (
	"fmt"
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

// Message is the sentence the CLI and the desktop show for an unset reference.
func (r EnvRef) Message() string {
	return fmt.Sprintf("environment variable %s is not set (the environment variable %q refers to it as ${%s})", r.Name, r.Var, r.Name)
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

// ResolveEnvironment returns a copy of env with every value evaluated once: ${NAME} references
// expanded and {{$functions}} generated, in order, using eng. Resolving the environment once per
// command, then giving that copy to every scenario, is what makes a generated environment value
// (for example a runId) the same in all of them. The original env is not modified.
func ResolveEnvironment(env *domain.Environment, eng *Engine) (*domain.Environment, []error) {
	if env == nil {
		return nil, nil
	}
	r := NewResolverWith(env, eng)
	out := *env
	out.Variables = make([]domain.EnvironmentVariable, len(env.Variables))
	for i, v := range env.Variables {
		val, _ := r.Get(v.Name)
		out.Variables[i] = domain.EnvironmentVariable{Name: v.Name, Value: val, Secret: v.Secret}
	}
	return &out, r.TakeErrors()
}

// WithOverrides returns a copy of env with each name=value in vars overriding or adding a
// variable; env itself is not modified and may be nil. An override keeps the variable's secret
// flag, so a secret stays redacted. A malformed entry is an error naming it.
func WithOverrides(env *domain.Environment, vars []string) (*domain.Environment, error) {
	out := &domain.Environment{SchemaVersion: domain.CurrentSchemaVersion, Name: "command line"}
	if env != nil {
		copied := *env
		copied.Variables = append([]domain.EnvironmentVariable(nil), env.Variables...)
		out = &copied
	}
	for _, raw := range vars {
		name, value, ok := strings.Cut(raw, "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			return nil, fmt.Errorf("%q must look like name=value", raw)
		}
		replaced := false
		for i := range out.Variables {
			if out.Variables[i].Name == name {
				out.Variables[i].Value = value
				replaced = true
				break
			}
		}
		if !replaced {
			out.Variables = append(out.Variables, domain.EnvironmentVariable{Name: name, Value: value})
		}
	}
	return out, nil
}
