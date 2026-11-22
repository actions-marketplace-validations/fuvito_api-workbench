package main

import (
	"bytes"
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// The Action is a published interface: one stray character in action.yml breaks every workflow that
// uses it, and nothing else loads the file. (0.7.0 shipped with a lone carriage return in a shell
// script, which YAML reads as a line break, so the whole file was invalid.)
func TestActionFileIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "action.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for i, b := range raw {
		if b == '\r' && (i+1 >= len(raw) || raw[i+1] != '\n') {
			t.Fatalf("action.yml line %d: a lone carriage return; YAML reads it as a line break", bytes.Count(raw[:i], []byte("\n"))+1)
		}
	}

	var doc struct {
		Name   string                    `yaml:"name"`
		Inputs map[string]map[string]any `yaml:"inputs"`
		Runs   struct {
			Using string `yaml:"using"`
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("action.yml is not valid YAML: %v", err)
	}
	if doc.Name == "" || doc.Runs.Using != "composite" || len(doc.Inputs) == 0 || len(doc.Runs.Steps) == 0 {
		t.Fatalf("action.yml is missing its name, inputs or steps: %+v", doc)
	}

	// Every input the scripts read must be declared, so a renamed input cannot be silently ignored.
	for _, name := range []string{"command", "spec", "read-only", "negative", "baseline", "fail-on", "against", "var", "report"} {
		if _, ok := doc.Inputs[name]; !ok {
			t.Errorf("input %q is not declared", name)
		}
	}

	bash, err := osexec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available to check the scripts")
	}
	for _, step := range doc.Runs.Steps {
		if step.Run == "" {
			continue
		}
		cmd := osexec.Command(bash, "-n")
		cmd.Stdin = bytes.NewReader([]byte(step.Run))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("step %q is not valid bash: %v\n%s", step.Name, err, out)
		}
	}
}
