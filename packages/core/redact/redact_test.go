package redact_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/redact"
)

// A report still has to say what was wrong: hiding everything would be safe and useless.
func TestOrdinaryValuesStayReadable(t *testing.T) {
	for _, c := range []struct {
		path []string
		v    any
		want string
	}{
		{[]string{"price"}, "9.99", `"9.99"`},
		{[]string{"0", "status"}, "archived", `"archived"`},
		{[]string{"createdAt"}, "2026-11-25T21:54:00Z", `"2026-11-25T21:54:00Z"`},
		{[]string{"id"}, "3f2b8c1e-9d4a-4e6b-8a1f-0c5d7e9b2a44", `"3f2b8c1e-9d4a-4e6b-8a1f-0c5d7e9b2a44"`},
		{[]string{"authorId"}, "a-17", `"a-17"`},
		{[]string{"email"}, "nope", `"nope"`},
		{[]string{"count"}, 3, "3"},
		{[]string{"ok"}, nil, "null"},
	} {
		if got, ok := redact.Value(c.path, c.v); !ok || got != c.want {
			t.Errorf("%v %v: %q %v, want %q", c.path, c.v, got, ok, c.want)
		}
	}
	for _, body := range []string{
		`{"error":"name is required","field":"name"}`,
		`user 42 was not found at /v1/billing-accounts/42`,
	} {
		if got := redact.Excerpt(body, 200); got != body {
			t.Errorf("an ordinary error body is kept: %q became %q", body, got)
		}
	}
}

func TestWhatIsHidden(t *testing.T) {
	if got, _ := redact.Value([]string{"session", "id"}, "abc"); got != `"***"` {
		t.Errorf("under a credential name: %s", got)
	}
	if _, ok := redact.Value([]string{"pin"}, 1234); ok {
		t.Error("a secret number is not shown at all")
	}
	if got, _ := redact.Value(nil, "abc"); got != `"***"` {
		t.Errorf("a body that is one bare text: %s", got)
	}
	if got, _ := redact.Value([]string{"note"}, strings.Repeat("word ", 40)); len(got) > 100 {
		t.Errorf("a long value is shortened: %d characters", len(got))
	}
	for _, v := range []string{"eyJhbGciOi.eyJzdWIiOiIx.sig-part", "Bearer abcdef123456", "tk_9fK2mQx7LpZr4TnB8vWcAa31"} {
		if got := redact.Tokens(v); !strings.Contains(got, redact.Mask) || strings.Contains(got, "abcdef123456") || strings.Contains(got, "9fK2mQx7") {
			t.Errorf("%q -> %q", v, got)
		}
	}
	got := redact.Excerpt(`{"message":"no","access_token":"abc12345","user":{"mail":"a.b@example.com"}}`, 200)
	if strings.Contains(got, "abc12345") || strings.Contains(got, "a.b@example.com") || !strings.Contains(got, `"message":"no"`) {
		t.Errorf("excerpt: %s", got)
	}
}
