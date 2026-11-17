package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/clibuild"
)

// buildAndRun builds a command with the CLI Command Builder, then executes exactly the
// arguments it produced through the real command tree, from the run-from folder.
func buildAndRun(t *testing.T, root, command string, values map[string]string) (string, error) {
	t.Helper()
	res := clibuild.Build(clibuild.Request{Command: command, Values: values}, root)
	if !res.Valid {
		t.Fatalf("builder rejected the command: %+v", res.Messages)
	}
	if res.Argv[0] != clibuild.Program {
		t.Fatalf("argv[0] = %q", res.Argv[0])
	}
	t.Chdir(root)
	return exec(t, res.Argv[1:]...)
}

// The commands the builder produces must work when run from the folder it assumed.
func TestBuilderCommandsRun(t *testing.T) {
	srv := pingServer(t)
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"openapi.yaml":         minimalSpec,
		"openapi-v2.yaml":      minimalSpec,
		"scenarios/s.yaml":     scenarioYAML(srv.URL),
		"environments/ci.yaml": "schemaVersion: \"1\"\nname: ci\nvariables:\n  - name: unused\n    value: x\n",
	})

	t.Run("run with a spec relative to the run-from folder", func(t *testing.T) {
		out, err := buildAndRun(t, root, "run", map[string]string{
			"scenario-file": filepath.Join(root, "scenarios", "s.yaml"), // absolute, as the picker returns it
			"env":           "environments/ci.yaml",
			"spec":          "openapi.yaml",
			"coverage":      "true",
		})
		if err != nil {
			t.Fatalf("run failed: %v\n%s", err, out)
		}
		if strings.Contains(out, "could not load spec") || !strings.Contains(out, "Spec coverage:") || !strings.Contains(out, "PASSED") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})

	t.Run("conform", func(t *testing.T) {
		out, err := buildAndRun(t, root, "conform", map[string]string{
			"spec": "openapi.yaml", "base-url": srv.URL, "coverage": "true", "request-timeout": "5000",
		})
		if err != nil {
			t.Fatalf("conform failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, "PASSED") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})

	t.Run("diff", func(t *testing.T) {
		out, err := buildAndRun(t, root, "diff", map[string]string{
			"base-spec": "openapi.yaml", "revision-spec": "openapi-v2.yaml", "breaking-only": "true",
		})
		if err != nil {
			t.Fatalf("diff failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, "No breaking changes") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})

	// Every report format, written where the builder said it would be.
	for _, c := range []struct {
		format, file, marker string
	}{
		{"junit", "reports/run.xml", "<testsuite"},
		{"html", "reports/run.html", "<!DOCTYPE html>"},
		{"json", "reports/run.json", `"passed"`},
		{"text", "reports/run.txt", "PASSED"},
	} {
		c := c
		t.Run("report "+c.format, func(t *testing.T) {
			out, err := buildAndRun(t, root, "run", map[string]string{
				"scenario-file": "scenarios/s.yaml", "format": c.format, "output": c.file,
			})
			if err != nil {
				t.Fatalf("run failed: %v: %s", err, out)
			}
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.file)))
			if err != nil {
				t.Fatalf("report not written where the builder said: %v", err)
			}
			if !strings.Contains(string(data), c.marker) {
				t.Errorf("%s report should contain %q, got: %s", c.format, c.marker, data)
			}
		})
	}

	t.Run("html diff report", func(t *testing.T) {
		out, err := buildAndRun(t, root, "diff", map[string]string{
			"base-spec": "openapi.yaml", "revision-spec": "openapi-v2.yaml",
			"format": "html", "output": "reports/diff.html",
		})
		if err != nil {
			t.Fatalf("diff failed: %v: %s", err, out)
		}
		data, err := os.ReadFile(filepath.Join(root, "reports", "diff.html"))
		if err != nil || !strings.Contains(string(data), "API Spec Diff") {
			t.Errorf("diff HTML report missing or wrong: %v", err)
		}
	})
}
