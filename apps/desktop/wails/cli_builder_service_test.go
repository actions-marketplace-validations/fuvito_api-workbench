package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cliTestScenario = `schemaVersion: "1"
title: Ping flow
steps:
  - label: Ping
    request:
      method: GET
      url: "{{baseUrl}}/ping"
    assertions:
      - type: status
        operator: eq
        expected: 200
`

const cliTestEnv = `schemaVersion: "1"
name: ci
variables:
  - name: baseUrl
    value: "${API_BASE_URL}"
`

const cliTestSpec = `openapi: "3.0.3"
info:
  title: Svc
  version: "1.0.0"
paths:
  /ping:
    get:
      responses:
        "200":
          description: OK
`

func cliWorkspace(t *testing.T) (*CliBuilderService, string) {
	t.Helper()
	isolateHome(t)
	root := t.TempDir()
	files := map[string]string{
		".git/HEAD":                "ref: refs/heads/main\n",
		"scenarios/ping.yaml":      cliTestScenario,
		"scenarios/sub/other.json": `{"schemaVersion":"1","title":"x","steps":[]}`,
		"environments/ci.yaml":     cliTestEnv,
		"api/openapi.yaml":         cliTestSpec,
		"notes.yaml":               "just: notes\n",
		"node_modules/x/spec.yaml": cliTestSpec,
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
	ws := newWS(t, root)
	return NewCliBuilderService(ws), root
}

func TestCliBuilder_Catalog(t *testing.T) {
	svc, _ := cliWorkspace(t)
	cat := svc.GetCliCatalog()
	names := []string{}
	for _, c := range cat {
		names = append(names, c.Name)
		if c.Goal == "" || len(c.ExitCodes) == 0 || len(c.Inputs) == 0 {
			t.Errorf("%s: catalog entry incomplete: %+v", c.Name, c)
		}
		have := map[string]CliInputIO{}
		for _, in := range c.Inputs {
			have[in.Name] = in
		}
		format, hasFormat := have["format"]
		if !hasFormat || strings.Join(format.Enum, ",") != "text,json,junit,html" || format.Default != "text" {
			t.Errorf("%s: format input = %+v", c.Name, format)
		}
		if out, ok := have["output"]; !ok || out.Kind != "outputFile" {
			t.Errorf("%s: output input = %+v", c.Name, out)
		}
	}
	if strings.Join(names, ",") != "run,conform,diff" {
		t.Errorf("commands = %v", names)
	}
}

func TestCliBuilder_ContextListsWorkspaceFiles(t *testing.T) {
	svc, root := cliWorkspace(t)
	ctx := svc.GetCliContext()

	if ctx.Workspace != root || ctx.RunFrom != root {
		t.Errorf("workspace/runFrom = %q / %q, want %q (it has .git)", ctx.Workspace, ctx.RunFrom, root)
	}
	labels := func(fs []CliFileIO) string {
		var out []string
		for _, f := range fs {
			out = append(out, f.Label)
		}
		return strings.Join(out, ",")
	}
	if got := labels(ctx.Scenarios); got != "scenarios/ping.yaml,scenarios/sub/other.json" {
		t.Errorf("scenarios = %s", got)
	}
	if got := labels(ctx.Environments); got != "environments/ci.yaml" {
		t.Errorf("environments = %s", got)
	}
	if got := labels(ctx.Specs); got != "api/openapi.yaml" {
		t.Errorf("specs = %s (notes.yaml is not a spec, node_modules is skipped)", got)
	}
}

func TestCliBuilder_NoWorkspace(t *testing.T) {
	isolateHome(t)
	svc := NewCliBuilderService(newWS(t, ""))
	ctx := svc.GetCliContext()
	if ctx.Workspace != "" || len(ctx.Scenarios) != 0 || ctx.Scenarios == nil {
		t.Errorf("empty workspace should give empty, non-nil lists: %+v", ctx)
	}
}

func TestCliBuilder_Build(t *testing.T) {
	svc, root := cliWorkspace(t)
	out := svc.BuildCliCommand(CliBuildInput{
		Command: "run",
		Values: map[string]string{
			"scenario-file": filepath.Join(root, "scenarios", "ping.yaml"),
			"env":           filepath.Join(root, "environments", "ci.yaml"),
			"spec":          filepath.Join(root, "api", "openapi.yaml"),
			"coverage":      "true",
		},
	})
	if !out.Valid {
		t.Fatalf("messages: %+v", out.Messages)
	}
	if got := strings.Join(out.Argv, " "); got != "clrsig run scenarios/ping.yaml --env environments/ci.yaml --spec api/openapi.yaml --coverage" {
		t.Errorf("argv = %s", got)
	}
	if len(out.Renderings) != 3 || out.Renderings[0].Shell != "bash" {
		t.Errorf("renderings = %+v", out.Renderings)
	}
	if len(out.Needs) != 1 || out.Needs[0].Name != "API_BASE_URL" {
		t.Errorf("needs = %+v", out.Needs)
	}
	if out.Explanation.Goal != "Run a scenario" || len(out.Explanation.Parts) < 4 {
		t.Errorf("explanation = %+v", out.Explanation)
	}
}

func TestCliBuilder_BuildSuggestsReportFile(t *testing.T) {
	svc, root := cliWorkspace(t)
	out := svc.BuildCliCommand(CliBuildInput{
		Command: "run",
		Values: map[string]string{
			"scenario-file": filepath.Join(root, "scenarios", "ping.yaml"),
			"format":        "html",
		},
	})
	if !out.Valid {
		t.Fatalf("messages: %+v", out.Messages)
	}
	if len(out.Suggestions) != 1 || out.Suggestions[0].Field != "output" || out.Suggestions[0].Value != "reports/run-ping.html" {
		t.Errorf("suggestions = %+v", out.Suggestions)
	}
	if got := strings.Join(out.Argv, " "); got != "clrsig run scenarios/ping.yaml --format html" {
		t.Errorf("argv = %s", got)
	}
}

func TestCliBuilder_BuildReportsProblems(t *testing.T) {
	svc, _ := cliWorkspace(t)
	out := svc.BuildCliCommand(CliBuildInput{Command: "run", Values: map[string]string{"scenario-file": "missing.yaml"}})
	if out.Valid || len(out.Messages) == 0 || out.Messages[0].Severity != "error" {
		t.Errorf("want an error message, got %+v", out)
	}
	if out.Argv == nil || out.Renderings == nil || out.Needs == nil || out.Explanation.Parts == nil {
		t.Error("lists must be non-nil so the UI can iterate them")
	}
}
