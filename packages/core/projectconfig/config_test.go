package projectconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const full = `schemaVersion: "1"
spec: specs/openapi.yaml
environment: environments/ci.yaml
baseUrl: http://localhost:8080
x-owner: platform-team
conform:
  readOnly: true
  negative: false
  failOn: warning
  requestTimeoutMs: 5000
  maxResponseTimeMs: 800
  maxResponseTimeFor: ["/reports=2000"]
  tags: [users]
  paths: [/users]
  baseline: conform-baseline.yaml
  x-note: kept
run:
  scenarios: [scenarios, more/one.yaml]
  failFast: true
reports:
  - junit=reports/conform.xml
`

func TestParse_EveryKey(t *testing.T) {
	c, err := Parse([]byte(full), "clrsig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Spec != "specs/openapi.yaml" || c.Environment != "environments/ci.yaml" || c.BaseURL != "http://localhost:8080" {
		t.Errorf("top level: %+v", c)
	}
	if c.Conform == nil || c.Conform.ReadOnly == nil || !*c.Conform.ReadOnly || c.Conform.Negative == nil || *c.Conform.Negative {
		t.Fatalf("booleans, including an explicit false, are kept: %+v", c.Conform)
	}
	if c.Conform.FailOn != "warning" || *c.Conform.RequestTimeoutMs != 5000 || *c.Conform.MaxResponseTimeMs != 800 ||
		len(c.Conform.MaxResponseTimeFor) != 1 || c.Conform.Tags[0] != "users" || c.Conform.Paths[0] != "/users" || c.Conform.Baseline != "conform-baseline.yaml" {
		t.Errorf("conform: %+v", c.Conform)
	}
	if len(c.Run.Scenarios) != 2 || !*c.Run.FailFast || len(c.Reports) != 1 {
		t.Errorf("run and reports: %+v %+v", c.Run, c.Reports)
	}
}

func TestParse_YAMLAndJSONAreEquivalent(t *testing.T) {
	js := `{"schemaVersion":"1","spec":"a.yaml","conform":{"readOnly":true,"failOn":"warning"},"reports":["html=r.html"]}`
	ym := "schemaVersion: \"1\"\nspec: a.yaml\nconform:\n  readOnly: true\n  failOn: warning\nreports: [html=r.html]\n"
	a, err := Parse([]byte(js), "clrsig.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse([]byte(ym), "clrsig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if a.Spec != b.Spec || *a.Conform.ReadOnly != *b.Conform.ReadOnly || a.Conform.FailOn != b.Conform.FailOn || a.Reports[0] != b.Reports[0] {
		t.Errorf("%+v vs %+v", a, b)
	}
}

func TestParse_OnlyTheVersionIsValid(t *testing.T) {
	if _, err := Parse([]byte("schemaVersion: \"1\"\n"), "clrsig.yaml"); err != nil {
		t.Fatalf("the smallest file must be valid: %v", err)
	}
}

func TestParse_Errors(t *testing.T) {
	for name, tc := range map[string]struct{ text, file, want string }{
		"empty":            {"", "clrsig.yaml", "empty"},
		"not yaml":         {"a: [", "clrsig.yaml", "not valid YAML"},
		"not json":         {"{", "clrsig.json", "not valid JSON"},
		"no version":       {"spec: a.yaml\n", "clrsig.yaml", "schemaVersion is missing"},
		"unquoted version": {"schemaVersion: 1\n", "clrsig.yaml", `quoted string "1"`},
		"newer version":    {"schemaVersion: \"2\"\n", "clrsig.yaml", "not supported"},
		"unknown key":      {"schemaVersion: \"1\"\nspecc: a\n", "clrsig.yaml", `unknown key "specc"`},
		"suggestion":       {"schemaVersion: \"1\"\nbaseURL: x\n", "clrsig.yaml", `did you mean "baseUrl"`},
		"unknown nested":   {"schemaVersion: \"1\"\nconform:\n  readonly: true\n", "clrsig.yaml", `did you mean "readOnly"`},
		"wrong type bool":  {"schemaVersion: \"1\"\nconform:\n  readOnly: yes please\n", "clrsig.yaml", "conform.readOnly has the wrong type: expected true or false"},
		"wrong type list":  {"schemaVersion: \"1\"\nrun:\n  scenarios: scenarios\n", "clrsig.yaml", "run.scenarios has the wrong type: expected a list"},
		"negative number":  {"schemaVersion: \"1\"\nconform:\n  requestTimeoutMs: -1\n", "clrsig.yaml", "must not be negative"},
		"bad fail-on":      {"schemaVersion: \"1\"\nconform:\n  failOn: note\n", "clrsig.yaml", "conform.failOn"},
		"bad limit":        {"schemaVersion: \"1\"\nconform:\n  maxResponseTimeFor: [nope]\n", "clrsig.yaml", "conform.maxResponseTimeFor"},
		"env reference":    {"schemaVersion: \"1\"\nbaseUrl: ${API_URL}\n", "clrsig.yaml", "baseUrl contains ${...}"},
		"secret in report": {"schemaVersion: \"1\"\nreports: [\"junit=${DIR}/r.xml\"]\n", "clrsig.yaml", "reports contains ${...}"},
	} {
		_, err := Parse([]byte(tc.text), tc.file)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want an error containing %q", name, err, tc.want)
		}
	}
}

func write(t *testing.T, dir, name, text string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiscover_WalksUpAndStopsAtTheFirstFile(t *testing.T) {
	root := t.TempDir()
	outer := write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\n")
	inner := write(t, root, "svc/clrsig.yaml", "schemaVersion: \"1\"\n")
	sub := filepath.Join(root, "svc", "deep", "er")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(sub, "")
	if err != nil || got != inner {
		t.Fatalf("nearest file wins: %q %v (want %q)", got, err, inner)
	}
	got, err = Discover(filepath.Join(root, "other"), "")
	if err != nil || got != "" {
		// "other" does not exist, so the walk starts at a missing folder and still reaches root
		if got != outer {
			t.Fatalf("%q %v", got, err)
		}
	}
}

func TestDiscover_StopsAtGitAndHome(t *testing.T) {
	root := t.TempDir()
	write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\n")
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "a")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := Discover(sub, ""); err != nil || got != "" {
		t.Errorf("a file above the repository is not used: %q %v", got, err)
	}
	// the repository root itself is looked at before stopping
	want := write(t, repo, "clrsig.yaml", "schemaVersion: \"1\"\n")
	if got, _ := Discover(sub, ""); got != want {
		t.Errorf("%q, want %q", got, want)
	}

	home := filepath.Join(root, "home")
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, _ := Discover(work, home); got != "" {
		t.Errorf("a file above the home folder is not used (the one in root): %q", got)
	}
}

func TestDiscover_TwoFilesInOneFolderIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\n")
	write(t, root, "clrsig.json", `{"schemaVersion":"1"}`)
	if _, err := Discover(root, ""); err == nil || !strings.Contains(err.Error(), "more than one project file") {
		t.Errorf("%v", err)
	}
}

func TestLoad_ReportsTheFileInErrors(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "clrsig.yaml", "schemaVersion: \"1\"\nspecc: a\n")
	if _, err := Load(p); err == nil || !strings.HasPrefix(err.Error(), "clrsig.yaml: ") {
		t.Errorf("%v", err)
	}
	if _, err := Load(filepath.Join(dir, "missing.yaml")); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("%v", err)
	}
}

func TestForCommandLine(t *testing.T) {
	root := t.TempDir()
	p := write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\n")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.ForCommandLine("specs/a.yaml", root); got != "specs/a.yaml" {
		t.Errorf("from the folder of the file: %q", got)
	}
	sub := filepath.Join(root, "svc")
	if got := c.ForCommandLine("specs/a.yaml", sub); got != filepath.Join(root, "specs", "a.yaml") {
		t.Errorf("from a subfolder the path is absolute: %q", got)
	}
	if got := c.ForCommandLine("https://x/api.yaml", root); got != "https://x/api.yaml" {
		t.Errorf("a URL is left alone: %q", got)
	}
	abs := filepath.Join(root, "x.yaml")
	if got := c.ForCommandLine(abs, sub); got != abs {
		t.Errorf("an absolute path is left alone: %q", got)
	}
	if got := c.ForCommandLine("", root); got != "" {
		t.Errorf("empty stays empty: %q", got)
	}
}
