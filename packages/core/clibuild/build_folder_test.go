package clibuild_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/clibuild"
)

// folderRepo adds a scenarios tree to the sample repository: three valid scenarios (one
// nested), a non-scenario file and, optionally, a broken scenario.
func folderRepo(t *testing.T, broken bool) string {
	t.Helper()
	root := repo(t)
	files := map[string]string{
		"suite/a.yaml":        scenarioYAML,
		"suite/b.yaml":        scenarioYAML,
		"suite/deep/c.yaml":   scenarioYAML,
		"suite/environment.y": "name: not a scenario\n",
		"suite/env.yaml":      envYAML,
	}
	if broken {
		files["suite/zz-broken.yaml"] = "schemaVersion: \"1\"\ntitle: Broken\nsteps: []\n"
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestBuild_RunFolder(t *testing.T) {
	root := folderRepo(t, false)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": filepath.Join(root, "suite"), "env": "environments/ci.yaml",
	}}, root)
	if !r.Valid {
		t.Fatalf("errors: %v", errorsOf(r))
	}
	if got := strings.Join(r.Argv, " "); got != "clrsig run suite --env environments/ci.yaml" {
		t.Errorf("argv = %q", got)
	}
	if !has(r, clibuild.Info, "3 scenarios found in suite and its sub-folders, all valid") {
		t.Errorf("want a count with sub-folders, got %+v", r.Messages)
	}
	if !has(r, clibuild.Info, "1 other file skipped") {
		t.Errorf("the env file is skipped, not run: %+v", r.Messages)
	}
}

func TestBuild_RunFolderNoRecursive(t *testing.T) {
	root := folderRepo(t, false)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "suite", "no-recursive": "true", "fail-fast": "true",
	}}, root)
	if !r.Valid {
		t.Fatalf("errors: %v", errorsOf(r))
	}
	if got := strings.Join(r.Argv, " "); got != "clrsig run suite --fail-fast --no-recursive" {
		t.Errorf("argv = %q", got)
	}
	if !has(r, clibuild.Info, "2 scenarios found in suite, all valid") || has(r, clibuild.Info, "sub-folders") {
		t.Errorf("only the top level counts: %+v", r.Messages)
	}
}

func TestBuild_RunFolderReportsBrokenScenarios(t *testing.T) {
	root := folderRepo(t, true)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "suite"}}, root)
	if r.Valid {
		t.Fatal("a broken scenario in the folder makes the command invalid, as the CLI would fail it")
	}
	if got := strings.Join(errorsOf(r), "|"); !strings.Contains(got, "zz-broken.yaml") || !strings.Contains(got, "no steps") {
		t.Errorf("errors should name the broken file: %v", got)
	}
	if !has(r, clibuild.Info, "4 scenarios found") {
		t.Errorf("still shows what was found: %+v", r.Messages)
	}
}

func TestBuild_RunFolderLimitsTheBrokenList(t *testing.T) {
	root := folderRepo(t, false)
	for i := 0; i < 8; i++ {
		name := filepath.Join(root, "suite", "bad", "bad"+string(rune('a'+i))+".yaml")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("schemaVersion: \"1\"\ntitle: Bad\nsteps: []\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "suite"}}, root)
	if n := len(errorsOf(r)); n != 6 {
		t.Errorf("5 broken files plus an 'and N more' line, got %d: %v", n, errorsOf(r))
	}
	if !has(r, clibuild.Error, "and 3 more broken scenarios") {
		t.Errorf("want the remainder counted: %+v", r.Messages)
	}
}

func TestBuild_RunFolderWithNoScenarios(t *testing.T) {
	root := repo(t)
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "empty"}}, root)
	if r.Valid || !strings.Contains(strings.Join(errorsOf(r), "|"), "no scenarios found") {
		t.Errorf("want 'no scenarios found', got valid=%v %v", r.Valid, errorsOf(r))
	}
}

func TestBuild_RunFolderCoverageNeedsASpec(t *testing.T) {
	root := folderRepo(t, false)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "suite", "coverage": "true"}}, root)
	if r.Valid || !strings.Contains(strings.Join(errorsOf(r), "|"), "--coverage needs openapi spec") {
		t.Errorf("no spec anywhere: valid=%v %v", r.Valid, errorsOf(r))
	}

	// a specFile: in only some of the scenarios satisfies the requirement but is called out
	withSpec := strings.Replace(scenarioYAML, "title: Lifecycle", "title: Lifecycle\nspecFile: ../../openapi.yaml", 1)
	if err := os.WriteFile(filepath.Join(root, "suite", "a.yaml"), []byte(withSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	r = clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "suite", "coverage": "true"}}, root)
	if !r.Valid {
		t.Fatalf("one specFile: is enough to be valid: %v", errorsOf(r))
	}
	if !has(r, clibuild.Warning, "only 1 of 3 scenarios name a specFile:") {
		t.Errorf("should say only some scenarios have a spec: %+v", r.Messages)
	}
}

func TestBuild_RunFolderCombinesVariableChecks(t *testing.T) {
	root := folderRepo(t, false)
	partial := "schemaVersion: \"1\"\nname: partial\nvariables:\n  - name: token\n    value: x\n"
	if err := os.WriteFile(filepath.Join(root, "environments", "partial.yaml"), []byte(partial), 0o644); err != nil {
		t.Fatal(err)
	}
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "suite", "env": "environments/partial.yaml",
	}}, root)
	if !has(r, clibuild.Warning, "{{baseUrl}}") {
		t.Errorf("the folder's scenarios use baseUrl, which the environment lacks: %+v", r.Messages)
	}
}

func TestBuild_FolderFlagsWarnForASingleFile(t *testing.T) {
	root := folderRepo(t, false)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "suite/a.yaml", "fail-fast": "true", "no-recursive": "true",
	}}, root)
	if !r.Valid {
		t.Fatal(errorsOf(r))
	}
	if !has(r, clibuild.Warning, "no effect when you run a single scenario file") {
		t.Errorf("want 'no effect' warnings, got %+v", r.Messages)
	}
}

func TestBuild_RunFolderReportNameUsesTheFolder(t *testing.T) {
	root := folderRepo(t, false)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "suite", "format": "html"}}, root)
	if len(r.Suggestions) != 1 || r.Suggestions[0].Value != "reports/run-suite.html" {
		t.Errorf("suggestions = %+v", r.Suggestions)
	}
}
