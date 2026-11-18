package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A ref that looks like a git option must never reach git: git would read
// "--output=<file>:<path>" as its --output option and write a file.
func TestDiff_AgainstRefThatLooksLikeAnOptionIsRefused(t *testing.T) {
	dir := gitRepo(t)
	spec := filepath.Join(dir, "api", "openapi.yaml")
	target := filepath.Join(t.TempDir(), "pwned")

	for _, ref := range []string{"--output=" + target, "-p", "--help", "-n1"} {
		_, err := exec(t, "diff", "--against="+ref, spec)
		if err == nil || !strings.Contains(err.Error(), "not a valid git ref") {
			t.Errorf("--against=%q: want a refusal, got %v", ref, err)
		}
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("git must not have been asked to write %s", target)
	}
	for _, name := range []string{"pwned", "pwned:api/openapi.yaml"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(target), name)); err == nil {
			t.Errorf("unexpected file %s", name)
		}
	}

	// ordinary refs, including ones with slashes, tags and revision syntax, still work
	for _, ref := range []string{"HEAD", "HEAD~0", "refs/heads/master"} {
		if _, err := exec(t, "diff", "--against="+ref, spec); err != nil && strings.Contains(err.Error(), "not a valid git ref") {
			t.Errorf("--against=%q should be accepted: %v", ref, err)
		}
	}
}

// conform reports are shared too, so a secret that ends up in an error message (a failed request
// quotes its URL) or in the base URL must not be printed.
func TestConform_ReportsNeverContainSecretEnvironmentValues(t *testing.T) {
	const secret = "s3cr3t-value-1234"
	spec := writeTempFile(t, "api.yaml", minimalSpec)
	env := writeTempFile(t, "env.yaml", "schemaVersion: \"1\"\nname: ci\nvariables:\n  - name: apiKey\n    value: "+secret+"\n    secret: true\n")
	// nothing listens here, so every request fails with an error that quotes the URL
	baseURL := "http://127.0.0.1:1/" + secret

	for _, format := range []string{"text", "json", "junit", "html", "markdown"} {
		out, err := exec(t, "conform", "--base-url", baseURL, "--env", env, "--format", format, spec)
		if exitCode(err) != 1 {
			t.Fatalf("%s: unreachable server should fail the run (exit 1), got %v\n%s", format, err, out)
		}
		if strings.Contains(out, secret) {
			t.Errorf("%s report leaked the secret:\n%s", format, out)
		}
		if !strings.Contains(out, "***") && !strings.Contains(out, `\*\*\*`) { // Markdown escapes the marker
			t.Errorf("%s report should show where the secret was removed:\n%s", format, out)
		}
	}
}
