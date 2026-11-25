package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/projectconfig"
)

const fullProjectFile = `schemaVersion: "1"
spec: specs/ro.yaml
environment: environments/ci.yaml
baseUrl: http://localhost:8080
conform:
  readOnly: true
  failOn: warning
  requestTimeoutMs: 5000
  tags: [things]
  baseline: conform-baseline.yaml
run:
  scenarios: [scenarios]
  failFast: true
reports:
  - junit=reports/out.xml
`

// Every key a project file can hold is mapped to a flag or argument, and every flag the table names
// exists on the commands that read it. A key added to the format without a mapping fails here.
func TestConfigSettingsCoverEveryKeyAndRealFlag(t *testing.T) {
	var mapped []string
	for _, s := range configSettings {
		mapped = append(mapped, s.key)
		for _, name := range s.commands {
			cmd := newConfigTarget(name)
			if s.flag != "" && cmd.Flags().Lookup(s.flag) == nil {
				t.Errorf("%s: clrsig %s has no --%s flag", s.key, name, s.flag)
			}
		}
	}
	want := projectconfig.Keys()
	sort.Strings(mapped)
	sort.Strings(want)
	if strings.Join(mapped, " ") != strings.Join(want, " ") {
		t.Errorf("settings table and file format disagree:\n table: %v\n format: %v", mapped, want)
	}
}

func TestConfigShow_SaysWhereEachValueComesFrom(t *testing.T) {
	project(t, map[string]string{"clrsig.yaml": fullProjectFile})
	out, err := exec(t, "config", "show", "conform")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{
		"Project file: clrsig.yaml",
		"clrsig conform (with no flags or arguments):",
		"spec ", "specs/ro.yaml", "environments/ci.yaml", "http://localhost:8080",
		"conform.readOnly", "warning", "5000", "things", "conform-baseline.yaml", "junit=reports/out.xml",
		"A flag or argument on the command line replaces any value above.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	line := func(key string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), key+" ") {
				return l
			}
		}
		return ""
	}
	if l := line("conform.failOn"); !strings.HasSuffix(strings.TrimSpace(l), "file") {
		t.Errorf("a value from the file says so: %q", l)
	}
	for _, key := range []string{"conform.negative", "conform.maxResponseTimeMs", "conform.paths"} {
		if l := line(key); !strings.HasSuffix(strings.TrimSpace(l), "default") {
			t.Errorf("%s was not set in the file, so it is the default: %q", key, l)
		}
	}
	if l := line("conform.requestTimeoutMs"); !strings.Contains(l, "5000") {
		t.Errorf("%q", l)
	}
	if strings.Contains(out, "run.failFast") {
		t.Errorf("only the settings of the asked command are listed:\n%s", out)
	}
}

func TestConfigShow_AllCommandsByDefault(t *testing.T) {
	project(t, map[string]string{"clrsig.yaml": fullProjectFile})
	out, err := exec(t, "config", "show")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"clrsig conform (", "clrsig run (", "clrsig validate (", "run.failFast", "run.scenarios"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// What config show says is what the command does.
func TestConfigShow_AgreesWithTheCommand(t *testing.T) {
	srv, writes := writeServer(t)
	project(t, map[string]string{
		"specs/ro.yaml": readOnlySpec,
		"clrsig.yaml":   "schemaVersion: \"1\"\nspec: specs/ro.yaml\nbaseUrl: " + srv.URL + "\nconform:\n  readOnly: true\n",
	})
	show, err := exec(t, "config", "show", "conform")
	if err != nil || !strings.Contains(show, srv.URL) || !strings.Contains(show, "specs/ro.yaml") {
		t.Fatalf("%v\n%s", err, show)
	}
	out, err := exec(t, "conform")
	if err != nil || writes.Load() != 0 || !strings.Contains(out, "SKIP  POST    /things") || !strings.Contains(out, "Base URL: "+srv.URL) {
		t.Fatalf("the command did what config show said: %v (writes %d)\n%s", err, writes.Load(), out)
	}
	if !strings.Contains(show, "true") {
		t.Errorf("readOnly is shown as true:\n%s", show)
	}
}

func TestConfigShow_NoFileIgnoredAndBadCommand(t *testing.T) {
	root := project(t, nil)
	if out, err := exec(t, "config", "show"); err != nil || !strings.Contains(out, "No project file found") || !strings.Contains(out, "clrsig init") {
		t.Errorf("%v\n%s", err, out)
	}
	if err := writeProject(root); err != nil {
		t.Fatal(err)
	}
	if out, err := exec(t, "config", "show", "--no-config"); err != nil || !strings.Contains(out, "ignored (--no-config)") {
		t.Errorf("%v\n%s", err, out)
	}
	if out, err := exec(t, "config", "show", "--config", filepath.Join(root, "clrsig.yaml"), "run"); err != nil || !strings.Contains(out, "clrsig run (") {
		t.Errorf("%v\n%s", err, out)
	}
	if _, err := exec(t, "config", "show", "diff"); err == nil || !strings.Contains(err.Error(), "supports: conform, run, validate") {
		t.Errorf("%v", err)
	}
}

func writeProject(root string) error {
	return writeFileText(filepath.Join(root, "clrsig.yaml"), "schemaVersion: \"1\"\nrun:\n  scenarios: [scenarios]\n")
}

func writeFileText(path, text string) error { return os.WriteFile(path, []byte(text), 0o600) }
