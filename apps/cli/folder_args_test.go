package main

import (
	"path/filepath"
	"strings"
	"testing"
)

const impactBaseSpec = `openapi: "3.0.3"
info: {title: t, version: "1"}
paths:
  /health:
    get:
      responses:
        "200": {description: OK}
  /users:
    post:
      responses:
        "201": {description: created}
`

const impactRevisionSpec = `openapi: "3.0.3"
info: {title: t, version: "2"}
paths:
  /health:
    get:
      responses:
        "200": {description: OK}
`

func scenarioCalling(title, method, url string) string {
	return "schemaVersion: \"1\"\ntitle: " + title + "\nsteps:\n  - label: call\n    request:\n      method: " + method + "\n      url: \"" + url + "\"\n    assertions:\n      - {type: status, operator: eq, expected: 200}\n"
}

func TestValidate_AcceptsFolders(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.yaml":               scenarioCalling("A", "GET", "http://x/a"),
		"sub/b.yaml":           scenarioCalling("B", "GET", "http://x/b"),
		"environments/ci.yaml": "schemaVersion: \"1\"\nname: ci\nvariables: []\n",
	})

	out, err := exec(t, "validate", root)
	if err != nil {
		t.Fatalf("a folder of valid scenarios validates: %v\n%s", err, out)
	}
	for _, want := range []string{"a.yaml", "b.yaml", "skipped 1 file that is not scenarios"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "ci.yaml:") {
		t.Errorf("the environment file must not be validated as a scenario:\n%s", out)
	}

	out, err = exec(t, "validate", "--no-recursive", root)
	if err != nil || strings.Contains(out, "b.yaml") {
		t.Errorf("--no-recursive skips sub-folders: %v\n%s", err, out)
	}

	writeFiles(t, root, map[string]string{"z-bad.yaml": "schemaVersion: \"1\"\ntitle: Bad\nsteps: []\n"})
	if _, err = exec(t, "validate", root); exitCode(err) != 1 {
		t.Errorf("a folder with an invalid scenario fails validation, got %v", err)
	}
}

func TestValidate_FilesStillBehaveAsBefore(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := exec(t, "validate", missing); err == nil {
		t.Error("a missing file is still an error")
	}
	if _, err := exec(t, "validate", t.TempDir()); err == nil || !strings.Contains(err.Error(), "no scenarios found") {
		t.Errorf("an empty folder is an error: %v", err)
	}
}

func TestImpact_AcceptsFolders(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"specs/base.yaml":     impactBaseSpec,
		"specs/rev.yaml":      impactRevisionSpec,
		"scen/health.yaml":    scenarioCalling("Health check", "GET", "http://x/health"),
		"scen/sub/users.yaml": scenarioCalling("Create user", "POST", "http://x/users"),
	})

	out, err := execFull(t, "impact", filepath.Join(root, "specs", "base.yaml"), filepath.Join(root, "specs", "rev.yaml"), filepath.Join(root, "scen"))
	if exitCode(err) != 1 {
		t.Fatalf("the removed POST /users breaks a scenario in a sub-folder (exit 1), got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Create user") || strings.Contains(out, "Health check") {
		t.Errorf("only the affected scenario is reported:\n%s", out)
	}

	out, err = execFull(t, "impact", "--no-recursive", filepath.Join(root, "specs", "base.yaml"), filepath.Join(root, "specs", "rev.yaml"), filepath.Join(root, "scen"))
	if err != nil {
		t.Errorf("--no-recursive leaves out the sub-folder, so nothing is affected: %v\n%s", err, out)
	}
}
