package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const generateSpec = `
openapi: "3.0.3"
info: {title: g, version: "1"}
paths:
  /a:
    get: {operationId: getA, responses: {"200": {description: ok}}}
  /b:
    get: {operationId: getB, responses: {"200": {description: ok}}}
`

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGenerateDoesNotReplaceAnEditedFileWithoutForce(t *testing.T) {
	spec := writeTemp(t, "spec.yaml", generateSpec)
	dir := filepath.Join(t.TempDir(), "out")
	gen := func(extra ...string) error {
		return runQuietly(t, append([]string{"generate", spec, "--output", dir, "--base-url", "http://localhost:1"}, extra...)...)
	}
	if err := gen(); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(dir, "getA.yaml"), filepath.Join(dir, "getB.yaml")
	original := read(t, a)
	if err := gen(); err != nil {
		t.Fatalf("running again over unchanged files is fine: %v", err)
	}
	// a checkout that changed only the line endings is unchanged too
	if err := os.WriteFile(b, []byte(strings.ReplaceAll(read(t, b), "\n", "\r\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gen(); err != nil {
		t.Fatalf("line endings: %v", err)
	}

	const edit = "# my edit\n"
	if err := os.WriteFile(a, []byte(original+edit), 0o644); err != nil {
		t.Fatal(err)
	}
	mine := filepath.Join(dir, "mine.txt")
	if err := os.WriteFile(mine, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	err := gen()
	if err == nil || !strings.Contains(err.Error(), "getA.yaml") || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("an edited file stops the run and is named: %v", err)
	}
	if read(t, a) != original+edit {
		t.Error("the edit was lost")
	}
	if _, err := os.Stat(b); err == nil {
		t.Error("a refusal writes nothing, not even the files that are missing")
	}
	if err := gen("--dry-run"); err != nil || read(t, a) != original+edit {
		t.Errorf("--dry-run changes nothing and does not fail: %v", err)
	}
	if err := gen("--force"); err != nil {
		t.Fatal(err)
	}
	if read(t, a) != original || read(t, mine) != "mine" {
		t.Error("--force replaces the generated files and no others")
	}
}

// Two operations can come out with one file name; the second must not replace the first.
func TestGenerateKeepsBothOfTwoRequestsWithOneName(t *testing.T) {
	spec := writeTemp(t, "spec.yaml", `
openapi: "3.0.3"
info: {title: g, version: "1"}
paths:
  /a:
    get: {summary: Same, responses: {"200": {description: ok}}}
  /b:
    get: {summary: Same, responses: {"200": {description: ok}}}
`)
	dir := filepath.Join(t.TempDir(), "out")
	if err := runQuietly(t, "generate", spec, "--output", dir, "--base-url", "http://localhost:1"); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil || len(files) != 2 {
		t.Fatalf("%v %v", files, err)
	}
	both := read(t, files[0]) + read(t, files[1])
	if !strings.Contains(both, "/a") || !strings.Contains(both, "/b") {
		t.Errorf("one request is missing:\n%s", both)
	}
}
