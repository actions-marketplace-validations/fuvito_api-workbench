package conformance

import "testing"

func TestPathOf(t *testing.T) {
	cases := []struct{ full, base, want string }{
		{"http://h:8080/users/1", "http://h:8080", "/users/1"},
		{"http://h:8080/users/1?x=1", "http://h:8080/", "/users/1"},
		{"http://h:8080", "http://h:8080", "/"},
		{"http://other/a/b", "http://h:8080", "/a/b"},
		{"http://other/a?q=1", "", "/a"},
		{"http://other", "", ""},
		{"%zz", "", "%zz"},
	}
	for _, c := range cases {
		if got := pathOf(c.full, c.base); got != c.want {
			t.Errorf("pathOf(%q, %q) = %q, want %q", c.full, c.base, got, c.want)
		}
	}
}
