package variables

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func TestWithOverrides(t *testing.T) {
	orig := &domain.Environment{Name: "e", Variables: []domain.EnvironmentVariable{
		{Name: "token", Value: "old", Secret: true},
		{Name: "baseUrl", Value: "http://a"},
	}}
	got, err := WithOverrides(orig, []string{"token=new", "extra=1", "baseUrl=http://b=c"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"token": "new", "extra": "1", "baseUrl": "http://b=c"}
	for _, v := range got.Variables {
		if want[v.Name] != v.Value {
			t.Errorf("%s = %q, want %q", v.Name, v.Value, want[v.Name])
		}
	}
	if !got.Variables[0].Secret {
		t.Error("an override must keep the secret flag so the value stays redacted")
	}
	if orig.Variables[0].Value != "old" || len(orig.Variables) != 2 {
		t.Errorf("the original environment must not change: %+v", orig.Variables)
	}

	fromNil, err := WithOverrides(nil, []string{"a=b"})
	if err != nil || len(fromNil.Variables) != 1 || fromNil.Variables[0].Value != "b" {
		t.Fatalf("nil environment: %+v, %v", fromNil, err)
	}
	for _, bad := range []string{"novalue", "=x", " =y"} {
		if _, err := WithOverrides(nil, []string{bad}); err == nil || !strings.Contains(err.Error(), "must look like name=value") {
			t.Errorf("%q: %v", bad, err)
		}
	}
}
