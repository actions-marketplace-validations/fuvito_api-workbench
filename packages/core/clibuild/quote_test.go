package clibuild

import "testing"

func TestQuote(t *testing.T) {
	cases := []struct {
		shell Shell
		in    string
		want  string
	}{
		// plain values stay bare
		{Bash, "scenarios/user-lifecycle.yaml", "scenarios/user-lifecycle.yaml"},
		{PowerShell, "scenarios/user-lifecycle.yaml", "scenarios/user-lifecycle.yaml"},
		{Cmd, `scenarios\user-lifecycle.yaml`, `scenarios\user-lifecycle.yaml`},
		{Bash, "http://localhost:8080", "http://localhost:8080"},

		// spaces
		{Bash, "my folder/a.yaml", `'my folder/a.yaml'`},
		{PowerShell, "my folder/a.yaml", `'my folder/a.yaml'`},
		{Cmd, `my folder\a.yaml`, `"my folder\a.yaml"`},

		// single quotes
		{Bash, "it's.yaml", `'it'\''s.yaml'`},
		{PowerShell, "it's.yaml", `'it''s.yaml'`},
		{Cmd, "it's.yaml", `"it's.yaml"`},

		// double quotes
		{Bash, `a"b`, `'a"b'`},
		{PowerShell, `a"b`, `'a"b'`},
		{Cmd, `a"b`, `"a\"b"`},

		// shell metacharacters must be neutralised
		{Bash, "$HOME/x", `'$HOME/x'`},
		{Bash, "a`b", "'a`b'"},
		{Bash, "a;b", `'a;b'`},
		{PowerShell, "$env:TOKEN", `'$env:TOKEN'`},
		{PowerShell, "a`b", "'a`b'"},
		{Cmd, "a&b", `"a&b"`},

		// commas are an array operator in PowerShell
		{Bash, "users,orders", "users,orders"},
		{PowerShell, "users,orders", `'users,orders'`},
		{Cmd, "users,orders", `"users,orders"`},

		// unicode and empty
		{Bash, "çalışma/özet.yaml", `'çalışma/özet.yaml'`},
		{Bash, "", "''"},
		{PowerShell, "", "''"},
	}
	for _, c := range cases {
		got, _ := quote(c.shell, c.in)
		if got != c.want {
			t.Errorf("%s quote(%q) = %s, want %s", c.shell, c.in, got, c.want)
		}
	}
}

func TestQuoteCmdWarnsAboutPercent(t *testing.T) {
	got, warn := quote(Cmd, "100%done")
	if got != `"100%done"` || warn == "" {
		t.Errorf("cmd with %%: got %s, warn %q", got, warn)
	}
	if _, warn := quote(Cmd, "plain.yaml"); warn != "" {
		t.Errorf("no warning expected for plain values, got %q", warn)
	}
}
