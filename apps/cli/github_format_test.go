package main

import (
	"strings"
	"testing"
)

func TestConformFormatGitHub_AnnotatesTheOperationLineInTheSpec(t *testing.T) {
	srv, _ := hitServer(t, "/b") // /b answers a body that breaks its spec
	project(t, map[string]string{"api/openapi.yaml": changedSpec("", "")})

	out, err := exec(t, "conform", "api/openapi.yaml", "--base-url", srv.URL, "--format", "github")
	if exitCodeOf(err) != 1 {
		t.Fatalf("the exit code is the same as for any other format: %v\n%s", err, out)
	}
	// /b is the second path; its get key is on line 14 of the generated spec
	var errLine string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "::error ") {
			errLine = l
		}
	}
	if !strings.HasPrefix(errLine, "::error file=api/openapi.yaml,line=") || !strings.Contains(errLine, "title=CS-RESP-005 GET /b::") {
		t.Fatalf("one error on the spec file with the check in the title:\n%s", out)
	}
	if !strings.Contains(out, "::notice title=Clear Signal conform::2 operations: 1 passed, 1 failed, 0 skipped.") {
		t.Errorf("a summary notice:\n%s", out)
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasPrefix(l, "::") {
			t.Errorf("--format github prints only workflow commands, got %q", l)
		}
	}
}

func TestConformFormatGitHub_IsAScreenFormatOfConformOnly(t *testing.T) {
	srv, _ := hitServer(t)
	project(t, map[string]string{"openapi.yaml": changedSpec("", ""), "s.yaml": scenarioYAML(srv.URL)})

	if _, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--report", "github=out.txt"); err == nil || !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("annotations are for the screen, not a --report file: %v", err)
	}
	if _, err := exec(t, "run", "s.yaml", "--format", "github"); err == nil || !strings.Contains(err.Error(), "unknown --format") {
		t.Errorf("run does not offer it: %v", err)
	}
	if _, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--format", "nope"); err == nil || !strings.Contains(err.Error(), "github") {
		t.Errorf("the list of formats names github for conform: %v", err)
	}
}

func TestConformFormatGitHub_AUnreadableLocationStaysOnTheRun(t *testing.T) {
	srv, _ := hitServer(t, "/b")
	root := project(t, map[string]string{"openapi.yaml": changedSpec("", "")})
	t.Chdir(t.TempDir()) // the spec is now outside the current folder
	out, err := exec(t, "conform", root+"/openapi.yaml", "--base-url", srv.URL, "--format", "github")
	if exitCodeOf(err) != 1 || strings.Contains(out, "file=") || !strings.Contains(out, "::error title=CS-RESP-005 GET /b::") {
		t.Errorf("%v\n%s", err, out)
	}
}
