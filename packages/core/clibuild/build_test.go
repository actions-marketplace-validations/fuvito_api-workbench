package clibuild_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/clibuild"
)

const scenarioYAML = `schemaVersion: "1"
title: Lifecycle
steps:
  - label: Login
    request:
      method: POST
      url: "{{baseUrl}}/login"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"
    extractions:
      - name: userId
        source: jsonpath
        path: $.id
    assertions:
      - type: status
        operator: eq
        expected: 200
  - label: Fetch
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
    assertions:
      - type: status
        operator: eq
        expected: 200
`

const envYAML = `schemaVersion: "1"
name: ci
variables:
  - name: baseUrl
    value: "${API_BASE_URL}"
  - name: token
    value: "${API_TOKEN}"
    secret: true
`

const specYAML = `openapi: "3.0.3"
info:
  title: Demo
  version: "1.0.0"
paths:
  /ping:
    get:
      operationId: ping
      responses:
        "200":
          description: OK
`

// repo creates a fake repository root (with .git) holding the sample files.
func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".git/HEAD":                "ref: refs/heads/main\n",
		"scenarios/s.yaml":         scenarioYAML,
		"environments/ci.yaml":     envYAML,
		"openapi.yaml":             specYAML,
		"api v2/openapi copy.yaml": specYAML,
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

func errorsOf(r clibuild.Result) []string {
	var out []string
	for _, m := range r.Messages {
		if m.Severity == clibuild.Error {
			out = append(out, m.Field+": "+m.Text)
		}
	}
	return out
}

func has(r clibuild.Result, sev clibuild.Severity, sub string) bool {
	for _, m := range r.Messages {
		if m.Severity == sev && strings.Contains(m.Text, sub) {
			return true
		}
	}
	return false
}

func TestBuild_RunWithRelativePaths(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": filepath.Join(root, "scenarios", "s.yaml"), // absolute, as a picker returns it
		"env":           "environments/ci.yaml",
		"spec":          "openapi.yaml",
		"coverage":      "true",
	}}, root)

	if !r.Valid {
		t.Fatalf("want valid, errors: %v", errorsOf(r))
	}
	want := "clrsig run scenarios/s.yaml --env environments/ci.yaml --spec openapi.yaml --coverage"
	if got := strings.Join(r.Argv, " "); got != want {
		t.Errorf("argv\n got %s\nwant %s", got, want)
	}
	if !has(r, clibuild.Info, "valid, 2 steps") || !has(r, clibuild.Info, "Demo 1.0.0, 1 operation") {
		t.Errorf("expected inspection info, got %+v", r.Messages)
	}
	// userId is extracted by the scenario itself, so only baseUrl/token matter and both exist.
	if has(r, clibuild.Warning, "does not define") {
		t.Errorf("no variable warning expected, got %+v", r.Messages)
	}
}

func TestBuild_NeedsFromEnvironment(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "scenarios/s.yaml", "env": "environments/ci.yaml",
	}}, root)
	got := map[string]bool{}
	for _, n := range r.Needs {
		got[n.Name] = n.Secret
	}
	if len(got) != 2 || got["API_BASE_URL"] || !got["API_TOKEN"] {
		t.Errorf("needs = %+v, want API_BASE_URL (plain) and API_TOKEN (secret)", r.Needs)
	}
}

func TestBuild_PathOutsideRunFromIsAbsoluteWithWarning(t *testing.T) {
	root := repo(t)
	sub := filepath.Join(root, "scenarios")
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": filepath.Join(root, "scenarios", "s.yaml"),
		"spec":          filepath.Join(root, "openapi.yaml"), // above the run-from folder
	}}, sub)
	if !r.Valid {
		t.Fatalf("errors: %v", errorsOf(r))
	}
	if r.Argv[2] != "s.yaml" {
		t.Errorf("scenario should be relative to run-from, got %q", r.Argv[2])
	}
	specArg := r.Argv[len(r.Argv)-1]
	if !filepath.IsAbs(specArg) {
		t.Errorf("outside path should stay absolute, got %q", specArg)
	}
	if !has(r, clibuild.Warning, "outside the folder") {
		t.Errorf("want an 'outside' warning, got %+v", r.Messages)
	}
}

func TestBuild_Errors(t *testing.T) {
	root := repo(t)
	cases := []struct {
		name   string
		cmd    string
		values map[string]string
		want   string
	}{
		{"missing required", "run", map[string]string{}, "Scenario is required"},
		{"file not found", "run", map[string]string{"scenario-file": "nope.yaml"}, "Scenario not found: nope.yaml"},
		{"folder given for file", "run", map[string]string{"scenario-file": "scenarios"}, "must be a file"},
		{"coverage needs spec", "run", map[string]string{"scenario-file": "scenarios/s.yaml", "coverage": "true"}, "--coverage needs openapi spec"},
		{"workspace needs spec", "run", map[string]string{"scenario-file": "scenarios/s.yaml", "workspace": "."}, "--workspace needs openapi spec"},
		{"bad int", "run", map[string]string{"scenario-file": "scenarios/s.yaml", "timeout": "-5"}, "whole number"},
		{"bad url", "conform", map[string]string{"spec": "openapi.yaml", "base-url": "localhost:8080"}, "http:// or https://"},
		{"workspace is a file", "run", map[string]string{"scenario-file": "scenarios/s.yaml", "spec": "openapi.yaml", "workspace": "openapi.yaml"}, "existing folder"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := clibuild.Build(clibuild.Request{Command: c.cmd, Values: c.values}, root)
			if r.Valid {
				t.Fatalf("want invalid, got valid: %v", r.Argv)
			}
			found := false
			for _, e := range errorsOf(r) {
				if strings.Contains(strings.ToLower(e), strings.ToLower(c.want)) {
					found = true
				}
			}
			if !found {
				t.Errorf("want error containing %q, got %v", c.want, errorsOf(r))
			}
		})
	}
}

func TestBuild_CoverageSatisfiedByScenarioSpecFile(t *testing.T) {
	root := repo(t)
	withSpec := strings.Replace(scenarioYAML, "title: Lifecycle", "title: Lifecycle\nspecFile: ../openapi.yaml", 1)
	if err := os.WriteFile(filepath.Join(root, "scenarios", "s.yaml"), []byte(withSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "scenarios/s.yaml", "coverage": "true",
	}}, root)
	if !r.Valid {
		t.Fatalf("a specFile: in the scenario satisfies --coverage; errors: %v", errorsOf(r))
	}
	if !has(r, clibuild.Info, "relative to the scenario file") {
		t.Errorf("should explain specFile resolution, got %+v", r.Messages)
	}
}

func TestBuild_InvalidScenarioIsReported(t *testing.T) {
	root := repo(t)
	if err := os.WriteFile(filepath.Join(root, "scenarios", "bad.yaml"), []byte("schemaVersion: \"1\"\ntitle: Bad\nsteps: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "scenarios/bad.yaml"}}, root)
	if r.Valid || !strings.Contains(strings.Join(errorsOf(r), "|"), "no steps") {
		t.Errorf("want 'no steps' error, got valid=%v %v", r.Valid, errorsOf(r))
	}
}

func TestBuild_UndefinedVariablesWarn(t *testing.T) {
	root := repo(t)
	env := "schemaVersion: \"1\"\nname: partial\nvariables:\n  - name: token\n    value: x\n"
	if err := os.WriteFile(filepath.Join(root, "environments", "partial.yaml"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "scenarios/s.yaml", "env": "environments/partial.yaml",
	}}, root)
	if !has(r, clibuild.Warning, "{{baseUrl}}") || has(r, clibuild.Warning, "{{userId}}") {
		t.Errorf("should warn about baseUrl only (userId is extracted), got %+v", r.Messages)
	}

	r = clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{"scenario-file": "scenarios/s.yaml"}}, root)
	if !has(r, clibuild.Warning, "no environment is selected") {
		t.Errorf("want a no-environment warning, got %+v", r.Messages)
	}
}

func TestBuild_DefaultsAreNotEmitted(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "conform", Values: map[string]string{
		"spec": "openapi.yaml", "request-timeout": "10000", "timeout": "5", "fail-fast": "false",
	}}, root)
	if got := strings.Join(r.Argv, " "); got != "clrsig conform openapi.yaml" {
		t.Errorf("defaults and unknown inputs must not appear, got %q", got)
	}
}

func TestBuild_ReportFormatAndOutput(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "diff", Values: map[string]string{
		"base-spec": "openapi.yaml", "revision-spec": "https://example.com/v2.yaml",
		"format": "html", "output": "reports/diff.html",
	}}, root)
	if !r.Valid {
		t.Fatal(errorsOf(r))
	}
	if got := strings.Join(r.Argv, " "); got != "clrsig diff openapi.yaml https://example.com/v2.yaml --format html --output reports/diff.html" {
		t.Errorf("argv = %q", got)
	}
	if !has(r, clibuild.Info, "self-contained HTML report") || !has(r, clibuild.Info, "folder reports will be created") {
		t.Errorf("want notes about the HTML file and the new folder, got %+v", r.Messages)
	}
	if len(r.Suggestions) != 0 {
		t.Errorf("no suggestion once an output file is chosen, got %+v", r.Suggestions)
	}
}

func TestBuild_TextFormatIsTheDefault(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "diff", Values: map[string]string{
		"base-spec": "openapi.yaml", "revision-spec": "openapi.yaml", "format": "text",
	}}, root)
	if got := strings.Join(r.Argv, " "); got != "clrsig diff openapi.yaml openapi.yaml" {
		t.Errorf("the default format is not written out, got %q", got)
	}
	if len(r.Suggestions) != 0 {
		t.Errorf("text needs no output file, got %+v", r.Suggestions)
	}
}

func TestBuild_FormatWithoutOutputSuggestsAFile(t *testing.T) {
	root := repo(t)
	cases := []struct {
		format, want string
		warns        bool
	}{
		{"html", "reports/run-s.html", true},
		{"junit", "reports/run-s.xml", true},
		{"json", "reports/run-s.json", false},
	}
	for _, c := range cases {
		r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
			"scenario-file": "scenarios/s.yaml", "format": c.format,
		}}, root)
		if len(r.Suggestions) != 1 || r.Suggestions[0].Field != "output" || r.Suggestions[0].Value != c.want {
			t.Errorf("%s: suggestion = %+v, want %s", c.format, r.Suggestions, c.want)
		}
		if got := has(r, clibuild.Warning, "printed to the screen"); got != c.warns {
			t.Errorf("%s: screen warning = %v, want %v", c.format, got, c.warns)
		}
	}
}

func TestBuild_OutputProblemsAndNotes(t *testing.T) {
	root := repo(t)
	base := map[string]string{"base-spec": "openapi.yaml", "revision-spec": "openapi.yaml", "format": "json"}
	with := func(output string) clibuild.Result {
		v := map[string]string{"output": output}
		for k, val := range base {
			v[k] = val
		}
		return clibuild.Build(clibuild.Request{Command: "diff", Values: v}, root)
	}

	if r := with("environments"); r.Valid || !strings.Contains(strings.Join(errorsOf(r), "|"), "is a folder") {
		t.Errorf("a folder is not a valid report file: valid=%v %v", r.Valid, errorsOf(r))
	}
	if r := with("openapi.yaml"); !r.Valid || !has(r, clibuild.Warning, "already exists and will be overwritten") {
		t.Errorf("an existing file is overwritten with a warning: valid=%v %+v", r.Valid, r.Messages)
	}
	if r := with(filepath.Join(filepath.Dir(root), "elsewhere.json")); !has(r, clibuild.Warning, "outside the folder") {
		t.Errorf("an output outside the run-from folder should warn: %+v", r.Messages)
	}
	if r := with("bad"); !has(r, clibuild.Info, "JSON report") {
		t.Errorf("want a JSON note, got %+v", r.Messages)
	}
}

func TestBuild_BadFormatIsRejected(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "diff", Values: map[string]string{
		"base-spec": "openapi.yaml", "revision-spec": "openapi.yaml", "format": "pdf",
	}}, root)
	if r.Valid || !strings.Contains(strings.Join(errorsOf(r), "|"), "must be one of: text, json, junit, html") {
		t.Errorf("want a format error, got valid=%v %v", r.Valid, errorsOf(r))
	}
}

func TestBuild_ExplanationMentionsFormat(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "diff", Values: map[string]string{
		"base-spec": "openapi.yaml", "revision-spec": "openapi.yaml", "format": "junit", "output": "reports/diff.xml",
	}}, root)
	found := false
	for _, p := range r.Explanation.Parts {
		if p.Token == "--format junit" && strings.Contains(p.Meaning, "junit is for CI test dashboards") {
			found = true
		}
	}
	if !found {
		t.Errorf("the explanation should describe --format junit, got %+v", r.Explanation.Parts)
	}
}

func TestSuggestOutputAndSlug(t *testing.T) {
	cases := []struct{ command, subject, format, want string }{
		{"run", "scenarios/User Lifecycle (v2).yaml", "html", "reports/run-user-lifecycle-v2.html"},
		{"run", "/abs/path/s.yml", "junit", "reports/run-s.xml"},
		{"conform", "openapi.json", "json", "reports/conform-openapi.json"},
		{"diff", "", "html", "reports/diff.html"},
		{"run", "run.yaml", "html", "reports/run.html"},
		{"run", "x.yaml", "text", "reports/run-x.txt"},
		{"run", "ünï.yaml", "html", "reports/run-n.html"},
	}
	for _, c := range cases {
		if got := clibuild.SuggestOutput(c.command, c.subject, c.format); got != c.want {
			t.Errorf("SuggestOutput(%q, %q, %q) = %s, want %s", c.command, c.subject, c.format, got, c.want)
		}
	}
	if got := clibuild.Slug("  Hello, World!  "); got != "hello-world" {
		t.Errorf("Slug = %q", got)
	}
}

func TestBuild_ConformFlagsAndQuoting(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "conform", Values: map[string]string{
		"spec": "api v2/openapi copy.yaml", "base-url": "http://localhost:8080",
		"tags": "users,orders", "negative": "true", "request-timeout": "2500",
	}}, root)
	if !r.Valid {
		t.Fatal(errorsOf(r))
	}
	byShell := map[clibuild.Shell]string{}
	for _, rd := range r.Renderings {
		byShell[rd.Shell] = rd.Oneline
	}
	want := map[clibuild.Shell]string{
		clibuild.Bash:       `clrsig conform 'api v2/openapi copy.yaml' --base-url http://localhost:8080 --negative --tags users,orders --request-timeout 2500`,
		clibuild.PowerShell: `clrsig conform 'api v2/openapi copy.yaml' --base-url http://localhost:8080 --negative --tags 'users,orders' --request-timeout 2500`,
		clibuild.Cmd:        `clrsig conform "api v2\openapi copy.yaml" --base-url http://localhost:8080 --negative --tags "users,orders" --request-timeout 2500`,
	}
	for shell, w := range want {
		if byShell[shell] != w {
			t.Errorf("%s\n got %s\nwant %s", shell, byShell[shell], w)
		}
	}
}

func TestBuild_MultilineContinuations(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "scenarios/s.yaml", "env": "environments/ci.yaml",
	}}, root)
	for _, rd := range r.Renderings {
		cont := map[clibuild.Shell]string{clibuild.Bash: "\\", clibuild.PowerShell: "`", clibuild.Cmd: "^"}[rd.Shell]
		lines := strings.Split(rd.Multiline, "\n")
		if len(lines) != 2 {
			t.Fatalf("%s: want 2 lines, got %q", rd.Shell, rd.Multiline)
		}
		if !strings.HasSuffix(lines[0], " "+cont) {
			t.Errorf("%s: first line should end with a %q continuation: %q", rd.Shell, cont, lines[0])
		}
		if strings.HasSuffix(lines[1], cont) {
			t.Errorf("%s: last line must not continue: %q", rd.Shell, lines[1])
		}
	}
}

func TestBuild_Explanation(t *testing.T) {
	root := repo(t)
	r := clibuild.Build(clibuild.Request{Command: "run", Values: map[string]string{
		"scenario-file": "scenarios/s.yaml", "spec": "openapi.yaml", "coverage": "true",
	}}, root)
	ex := r.Explanation
	if ex.Goal != "Run a scenario" || len(ex.ExitCodes) != 3 || len(ex.Outputs) == 0 {
		t.Errorf("explanation incomplete: %+v", ex)
	}
	tokens := map[string]string{}
	for _, p := range ex.Parts {
		tokens[p.Token] = p.Meaning
	}
	for _, want := range []string{"clrsig", "run", "scenarios/s.yaml", "--spec openapi.yaml", "--coverage"} {
		if tokens[want] == "" {
			t.Errorf("no explanation for token %q; have %v", want, tokens)
		}
	}
}

func TestBuild_UnknownCommandAndBadRunFrom(t *testing.T) {
	root := repo(t)
	if r := clibuild.Build(clibuild.Request{Command: "nope"}, root); r.Valid || len(r.Messages) == 0 {
		t.Error("unknown command must be invalid")
	}
	if r := clibuild.Build(clibuild.Request{Command: "run"}, filepath.Join(root, "missing")); r.Valid {
		t.Error("a run-from folder that does not exist must be invalid")
	}
	if r := clibuild.Build(clibuild.Request{Command: "run"}, ""); r.Valid {
		t.Error("an empty run-from folder must be invalid")
	}
}

func TestDefaultRunFrom(t *testing.T) {
	root := repo(t)
	got := clibuild.DefaultRunFrom(filepath.Join(root, "scenarios"))
	if got != root {
		t.Errorf("want the repository root %s, got %s", root, got)
	}
	plain := t.TempDir()
	if got := clibuild.DefaultRunFrom(plain); got != plain {
		t.Errorf("without .git the workspace itself is used, got %s", got)
	}
}
