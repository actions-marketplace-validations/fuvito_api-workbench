package variables

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("API_TOKEN", "s3cret")
	t.Setenv("BASE", "http://x")
	t.Setenv("EMPTY", "")
	t.Setenv("UNSET_NEVER", "")
	cases := []struct{ in, want string }{
		{"${API_TOKEN}", "s3cret"},
		{"Bearer ${API_TOKEN}!", "Bearer s3cret!"},
		{"${BASE}/v1/${BASE}", "http://x/v1/http://x"},
		{"${EMPTY}", ""},
		{"${DOES_NOT_EXIST_ANYWHERE}", "${DOES_NOT_EXIST_ANYWHERE}"},
		{"$${API_TOKEN}", "${API_TOKEN}"},
		{"plain {{baseUrl}} stays", "plain {{baseUrl}} stays"},
		{"$HOME and ${1BAD} and ${}", "$HOME and ${1BAD} and ${}"},
		{"", ""},
	}
	for _, c := range cases {
		if got := ExpandEnv(c.in); got != c.want {
			t.Errorf("ExpandEnv(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolverExpandsEnvironmentReferences(t *testing.T) {
	t.Setenv("API_BASE", "http://localhost:8080")
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{
		{Name: "baseUrl", Value: "${API_BASE}"},
		{Name: "literal", Value: "$${API_BASE}"},
	}}
	r := NewResolver(env)
	if got := r.Interpolate("{{baseUrl}}/health"); got != "http://localhost:8080/health" {
		t.Errorf("baseUrl = %q", got)
	}
	if got := r.Interpolate("{{literal}}"); got != "${API_BASE}" {
		t.Errorf("escaped reference = %q", got)
	}
	if env.Variables[0].Value != "${API_BASE}" {
		t.Error("the environment itself must keep the reference, so saving never writes the secret")
	}
}

func TestMissingEnv(t *testing.T) {
	t.Setenv("PRESENT", "1")
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{
		{Name: "token", Value: "${B_MISSING}", Secret: true},
		{Name: "url", Value: "${A_MISSING}/${PRESENT}"},
		{Name: "again", Value: "${A_MISSING}"},
		{Name: "escaped", Value: "$${C_MISSING}"},
	}}
	got := MissingEnv(env)
	want := []EnvRef{{"A_MISSING", "again"}, {"A_MISSING", "url"}, {"B_MISSING", "token"}}
	if len(got) != len(want) {
		t.Fatalf("missing = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("missing[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if MissingEnv(nil) != nil {
		t.Error("nil environment has nothing missing")
	}
}

func TestSecretValues(t *testing.T) {
	t.Setenv("TOKEN_A", "alpha-secret")
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{
		{Name: "a", Value: "${TOKEN_A}", Secret: true},
		{Name: "b", Value: "literal-secret", Secret: true},
		{Name: "c", Value: "${NOT_SET_ANYWHERE_X}", Secret: true},
		{Name: "d", Value: "   ", Secret: true},
		{Name: "e", Value: "public", Secret: false},
	}}
	got := SecretValues(env)
	if len(got) != 3 || got[0] != "alpha-secret" || got[1] != "literal-secret" || got[2] != "${NOT_SET_ANYWHERE_X}" {
		t.Errorf("secrets = %q", got)
	}
}
