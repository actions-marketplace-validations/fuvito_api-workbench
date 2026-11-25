package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/fuvito/api-workbench/packages/core/projectconfig"
	"github.com/fuvito/api-workbench/packages/core/scaffold"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

func TestProjectFilesParseWithTheRealParsers(t *testing.T) {
	for name, o := range map[string]scaffold.ProjectOptions{
		"nothing":         {},
		"spec only":       {Spec: "specs/openapi.yaml"},
		"url spec":        {Spec: "https://example.com/api/openapi.yaml?v=2&x=1", BaseURL: "https://api.example.com/v1"},
		"spec with space": {Spec: "my specs/api #1.yaml", BaseURL: "http://localhost:8080"},
		"odd base url":    {BaseURL: "http://localhost:8080/path: x"},
	} {
		entries := scaffold.ProjectEntries(o)
		for _, e := range entries {
			if e.Dir {
				continue
			}
			switch {
			case e.Path == "clrsig.yaml":
				cfg, err := projectconfig.Parse([]byte(e.Content), e.Path)
				if err != nil {
					t.Errorf("%s: %s: %v\n%s", name, e.Path, err, e.Content)
					continue
				}
				if cfg.Spec != o.Spec || cfg.BaseURL != o.BaseURL {
					t.Errorf("%s: the values must read back as written: spec %q baseUrl %q", name, cfg.Spec, cfg.BaseURL)
				}
				if cfg.Conform == nil || cfg.Conform.ReadOnly == nil || !*cfg.Conform.ReadOnly {
					t.Errorf("%s: a new project starts read-only", name)
				}
			case strings.HasPrefix(e.Path, "environments/"):
				env, err := serialization.ParseEnvironment([]byte(e.Content), "yaml")
				if err != nil {
					t.Errorf("%s: %s: %v\n%s", name, e.Path, err, e.Content)
					continue
				}
				if len(env.Variables) == 0 {
					t.Errorf("%s: %s has no variables", name, e.Path)
				}
			}
		}
	}
}

func TestLocalEnvironmentCarriesTheBaseURL(t *testing.T) {
	for _, e := range scaffold.ProjectEntries(scaffold.ProjectOptions{BaseURL: "https://api.example.com"}) {
		if e.Path != "environments/local.yaml" {
			continue
		}
		env, err := serialization.ParseEnvironment([]byte(e.Content), "yaml")
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := env.Lookup("baseUrl"); !ok || v != "https://api.example.com" {
			t.Errorf("baseUrl = %q %v", v, ok)
		}
	}
}

func TestTheCIEnvironmentHoldsNoSecret(t *testing.T) {
	for _, e := range scaffold.ProjectEntries(scaffold.ProjectOptions{}) {
		if e.Path != "environments/ci.yaml" {
			continue
		}
		if !strings.Contains(e.Content, "${API_TOKEN}") || !strings.Contains(e.Content, "secret: true") {
			t.Errorf("the CI token is a ${NAME} reference marked secret:\n%s", e.Content)
		}
	}
}

func TestApplyCreatesThenKeepsEverything(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new", "project") // does not exist yet
	entries := scaffold.ProjectEntries(scaffold.ProjectOptions{Spec: "specs/openapi.yaml"})
	first, err := scaffold.Apply(root, entries, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Created) != len(entries) || len(first.Kept) != 0 {
		t.Fatalf("first run: %+v", first)
	}
	for _, p := range []string{"clrsig.yaml", "environments/local.yaml", "environments/ci.yaml", "scenarios/.gitkeep", "requests/.gitkeep", "specs/.gitkeep"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	second, err := scaffold.Apply(root, entries, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Created) != 0 || len(second.Kept) != len(entries) {
		t.Fatalf("a second run changes nothing: %+v", second)
	}
	leftovers, _ := filepath.Glob(filepath.Join(root, "*", ".clrsig-init-*"))
	top, _ := filepath.Glob(filepath.Join(root, ".clrsig-init-*"))
	if len(leftovers)+len(top) != 0 {
		t.Errorf("temporary files left behind: %v %v", leftovers, top)
	}
}

func TestApplyNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	mine := "schemaVersion: \"1\"\nspec: mine.yaml\n"
	if err := os.WriteFile(filepath.Join(root, "clrsig.yaml"), []byte(mine), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(root, "environments", "local.yaml")
	if err := os.WriteFile(local, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := scaffold.Apply(root, scaffold.ProjectEntries(scaffold.ProjectOptions{Spec: "other.yaml"}), false)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "clrsig.yaml")); string(got) != mine {
		t.Errorf("an existing clrsig.yaml was changed:\n%s", got)
	}
	if got, _ := os.ReadFile(local); string(got) != "mine" {
		t.Errorf("an existing environment file was changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "scenarios", ".gitkeep")); err == nil {
		t.Error("an existing folder is left as it is, with no .gitkeep added")
	}
	if len(out.Kept) != 3 || len(out.Created) != 3 {
		t.Errorf("kept clrsig.yaml, local.yaml and scenarios; created ci.yaml, requests, specs: %+v", out)
	}
}

func TestAnExistingProjectFileUnderAnotherNameCounts(t *testing.T) {
	for _, name := range []string{"clrsig.yml", "clrsig.json"} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, name), []byte(`{"schemaVersion":"1"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := scaffold.Apply(root, scaffold.ProjectEntries(scaffold.ProjectOptions{}), false); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, "clrsig.yaml")); err == nil {
			t.Errorf("%s exists, so a second project file must not be created", name)
		}
		if _, err := projectconfig.Discover(root, ""); err != nil {
			t.Errorf("the folder must still have exactly one project file: %v", err)
		}
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	out, err := scaffold.Apply(root, scaffold.ProjectEntries(scaffold.ProjectOptions{}), true)
	if err != nil || len(out.Created) == 0 {
		t.Fatalf("%v %+v", err, out)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("a dry run wrote %d item(s)", len(entries))
	}
}

func TestApplyRefusesPathsOutsideTheFolder(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"../escape.yaml", "a/../../b.yaml", "/abs.yaml", ""} {
		if _, err := scaffold.Apply(root, []scaffold.Entry{{Path: p, Content: "x"}}, false); err == nil {
			t.Errorf("%q must be refused", p)
		}
	}
}

func TestApplyRefusesASymbolicLinkOutOfTheFolder(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "environments")); err != nil {
		t.Skipf("cannot create a symbolic link here: %v", err)
	}
	_, err := scaffold.Apply(root, scaffold.ProjectEntries(scaffold.ProjectOptions{}), false)
	if err == nil || !strings.Contains(err.Error(), "outside the project folder") {
		t.Fatalf("%v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("something was written outside the folder: %v", entries)
	}
	if _, err := os.Stat(filepath.Join(root, "clrsig.yaml")); err == nil {
		t.Error("nothing is written when one path is refused")
	}
}

func TestWriteNewRefusesAnExistingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "w.yml")
	if err := scaffold.WriteNew(p, "one"); err != nil {
		t.Fatal(err)
	}
	if err := scaffold.WriteNew(p, "two"); err == nil {
		t.Error("an existing file is never replaced")
	}
	if got, _ := os.ReadFile(p); string(got) != "one" {
		t.Errorf("%q", got)
	}
}

func TestWorkflowIsValidAndOnlyUsesInputsTheActionHas(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action struct {
		Inputs map[string]any `yaml:"inputs"`
	}
	if err := yaml.Unmarshal(raw, &action); err != nil {
		t.Fatal(err)
	}
	for name, o := range map[string]scaffold.WorkflowOptions{
		"release":   {Version: "v0.9.0", Spec: "api/openapi.yaml", Environment: "environments/ci.yaml", ReadOnly: true},
		"dev build": {Version: "dev"},
		"empty":     {},
	} {
		text := scaffold.GitHubWorkflow(o)
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					Uses string            `yaml:"uses"`
					With map[string]any    `yaml:"with"`
					Env  map[string]string `yaml:"env"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal([]byte(text), &wf); err != nil {
			t.Fatalf("%s: not valid YAML: %v\n%s", name, err, text)
		}
		found := false
		for _, step := range wf.Jobs["conform"].Steps {
			if !strings.HasPrefix(step.Uses, "fuvito/api-workbench@") {
				continue
			}
			found = true
			for k := range step.With {
				if _, ok := action.Inputs[k]; !ok {
					t.Errorf("%s: the workflow passes %q, which action.yml does not declare", name, k)
				}
			}
			if step.With["command"] != "conform" {
				t.Errorf("%s: command = %v", name, step.With["command"])
			}
		}
		if !found {
			t.Errorf("%s: no step uses the action:\n%s", name, text)
		}
	}
	rel := scaffold.GitHubWorkflow(scaffold.WorkflowOptions{Version: "v0.9.0", ReadOnly: true})
	if !strings.Contains(rel, "fuvito/api-workbench@v0.9.0\n") || !strings.Contains(rel, "read-only: 'true'") {
		t.Errorf("a release pins its own tag and honours read-only:\n%s", rel)
	}
	if dev := scaffold.GitHubWorkflow(scaffold.WorkflowOptions{Version: "dev"}); !strings.Contains(dev, "@main") || !strings.Contains(dev, "pin a release tag") {
		t.Errorf("a build that is not a release says so:\n%s", dev)
	}
	var parsed struct {
		Permissions map[string]string `yaml:"permissions"`
	}
	if err := yaml.Unmarshal([]byte(rel), &parsed); err != nil || len(parsed.Permissions) != 1 || parsed.Permissions["contents"] != "read" {
		t.Errorf("the workflow asks only to read the repository: %v %v\n%s", parsed.Permissions, err, rel)
	}
	if strings.Contains(scaffold.GitHubWorkflow(scaffold.WorkflowOptions{}), "read-only") {
		t.Error("read-only is only written when the project asks for it")
	}
}
