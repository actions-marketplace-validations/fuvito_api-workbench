package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// The path the proposal promises: an empty folder, one init, then clrsig conform with no flags.
func TestInit_ThenConformNeedsNoFlags(t *testing.T) {
	srv, writes := writeServer(t)
	root := project(t, map[string]string{"openapi.yaml": readOnlySpec})

	out, err := exec(t, "init", "--spec", "openapi.yaml", "--base-url", srv.URL)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{"Created in this folder:", "clrsig.yaml", "environments/local.yaml", "environments/ci.yaml", "scenarios", "Next:", "clrsig conform"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output lacks %q:\n%s", want, out)
		}
	}
	cfg := readFile(t, filepath.Join(root, "clrsig.yaml"))
	if !strings.Contains(cfg, "spec: openapi.yaml") || !strings.Contains(cfg, "baseUrl: "+srv.URL) || !strings.Contains(cfg, "readOnly: true") {
		t.Errorf("clrsig.yaml:\n%s", cfg)
	}

	out, err = exec(t, "conform")
	if err != nil || !strings.Contains(out, "PASSED") || !strings.Contains(out, "Using clrsig.yaml") {
		t.Fatalf("conform with no flags after init: %v\n%s", err, out)
	}
	if writes.Load() != 0 {
		t.Errorf("a new project is read-only, but %d write(s) were sent", writes.Load())
	}
}

func TestInit_RunningItAgainChangesNothing(t *testing.T) {
	root := project(t, nil)
	if out, err := exec(t, "init"); err != nil || !strings.Contains(out, "Created in this folder:") {
		t.Fatalf("%v\n%s", err, out)
	}
	before := readFile(t, filepath.Join(root, "clrsig.yaml"))
	out, err := exec(t, "init", "--spec", "https://example.com/other.yaml")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "Nothing to create") || !strings.Contains(out, "Kept") || strings.Contains(out, "Created in") {
		t.Errorf("a second init creates nothing:\n%s", out)
	}
	if readFile(t, filepath.Join(root, "clrsig.yaml")) != before {
		t.Error("an existing clrsig.yaml must not change, even when other values are given")
	}
}

func TestInit_NeverReplacesYourFiles(t *testing.T) {
	root := project(t, map[string]string{
		"clrsig.yml":              "schemaVersion: \"1\"\nspec: mine.yaml\n",
		"environments/local.yaml": "mine",
	})
	out, err := exec(t, "init")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "clrsig.yaml")); err == nil {
		t.Error("a clrsig.yml exists, so no second project file may be created")
	}
	if readFile(t, filepath.Join(root, "environments", "local.yaml")) != "mine" {
		t.Error("an existing environment file was changed")
	}
	if !strings.Contains(out, "clrsig.yaml") || !strings.Contains(out, "Kept") {
		t.Errorf("the kept files are listed:\n%s", out)
	}
}

func TestInit_DryRunWritesNothing(t *testing.T) {
	root := project(t, nil)
	out, err := exec(t, "init", "--dry-run")
	if err != nil || !strings.Contains(out, "Dry run: nothing was written") || !strings.Contains(out, "Would create:") {
		t.Fatalf("%v\n%s", err, out)
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.Name() != ".git" {
			t.Errorf("a dry run wrote %s", e.Name())
		}
	}
}

func TestInit_AFolderArgumentAndASpecPathRelativeToIt(t *testing.T) {
	root := project(t, map[string]string{"api/openapi.yaml": readOnlySpec})
	out, err := exec(t, "init", "service", "--spec", filepath.Join("api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "cd service") {
		t.Errorf("the next steps say to change folder:\n%s", out)
	}
	cfg := readFile(t, filepath.Join(root, "service", "clrsig.yaml"))
	if !strings.Contains(cfg, "spec: ../api/openapi.yaml") {
		t.Errorf("the spec path is written relative to the project folder:\n%s", cfg)
	}
}

func TestInit_ChecksBeforeWriting(t *testing.T) {
	root := project(t, map[string]string{"bad.yaml": "not: an openapi document\n"})
	for name, args := range map[string][]string{
		"spec that does not parse": {"init", "--spec", "bad.yaml"},
		"missing spec file":        {"init", "--spec", "missing.yaml"},
		"base url without scheme":  {"init", "--base-url", "localhost:8080"},
	} {
		if _, err := exec(t, args...); err == nil {
			t.Errorf("%s must fail", name)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "clrsig.yaml")); err == nil {
		t.Error("nothing is written when a value is wrong")
	}
	if _, err := exec(t, "init", "bad.yaml/inside"); err == nil {
		t.Error("a folder path through a file must fail")
	}
}

func TestInit_AURLSpecIsNotFetched(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	defer srv.Close()
	root := project(t, nil)
	if out, err := exec(t, "init", "--spec", srv.URL+"/openapi.yaml"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if hits.Load() != 0 {
		t.Error("init must not touch the network")
	}
	if cfg := readFile(t, filepath.Join(root, "clrsig.yaml")); !strings.Contains(cfg, srv.URL+"/openapi.yaml") {
		t.Errorf("the URL is written as given:\n%s", cfg)
	}
}

func TestInitGitHubActions_PrintsByDefaultAndUsesTheProjectFile(t *testing.T) {
	root := project(t, map[string]string{
		"specs/api.yaml":       readOnlySpec,
		"environments/ci.yaml": "schemaVersion: \"1\"\nname: ci\nvariables: []\n",
		"clrsig.yaml":          "schemaVersion: \"1\"\nspec: specs/api.yaml\nenvironment: environments/local.yaml\nconform:\n  readOnly: false\n",
	})
	out, err := exec(t, "init", "github-actions")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{"uses: fuvito/api-workbench@", "command: conform", "spec: specs/api.yaml", "env: environments/ci.yaml", "API_TOKEN"} {
		if !strings.Contains(out, want) {
			t.Errorf("workflow lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "read-only") {
		t.Errorf("readOnly: false in the project file means the workflow does not force read-only:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); err == nil {
		t.Error("printing must not write anything")
	}
}

func TestInitGitHubActions_WriteCreatesTheFileOnceOnly(t *testing.T) {
	root := project(t, nil)
	out, err := exec(t, "init", "github-actions", "--write")
	if err != nil || !strings.Contains(out, "Wrote .github/workflows/clrsig.yml") || !strings.Contains(out, "API_BASE_URL") {
		t.Fatalf("%v\n%s", err, out)
	}
	path := filepath.Join(root, ".github", "workflows", "clrsig.yml")
	first := readFile(t, path)
	if !strings.Contains(first, "read-only: 'true'") {
		t.Errorf("with no project file the workflow is read-only:\n%s", first)
	}
	if err := os.WriteFile(path, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := exec(t, "init", "github-actions", "--write"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("an existing workflow is not replaced: %v", err)
	}
	if readFile(t, path) != "mine" {
		t.Error("the existing workflow changed")
	}
	if _, err := exec(t, "init", "github-actions", "--output", "x.yml"); err == nil || !strings.Contains(err.Error(), "--output needs --write") {
		t.Errorf("%v", err)
	}
	if _, err := exec(t, "init", "github-actions", "--write", "--output", filepath.Join("ci", "api.yml")); err != nil {
		t.Errorf("--output chooses the file: %v", err)
	}
}
