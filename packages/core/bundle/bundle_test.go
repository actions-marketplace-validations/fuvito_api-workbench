package bundle_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/bundle"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

const good = `
schemaVersion: "1"
title: Suite
specFile: https://api.example.com/openapi.json
environments:
  - name: Staging
    variables:
      - {name: baseUrl, value: "https://staging.example.com"}
      - {name: token, value: CHANGE_ME, secret: true}
requests:
  - file: check-health
    title: Check health
    method: GET
    url: "{{baseUrl}}/health"
    timeout: 5s
  - file: auth/login
    method: POST
    url: "{{baseUrl}}/login"
    body: {type: json, content: '{"u":"x"}'}
scenarios:
  - title: Smoke test
    steps:
      - label: Health
        requestRef: check-health
        assertions:
          - {type: status, operator: eq, expected: 200}
          - {type: schema, schema: getHealth}
      - label: Login
        requestRef: requests/auth/login.yaml
        extractions:
          - {name: sessionId, source: jsonpath, path: "$.id"}
      - label: Use session
        request:
          method: GET
          url: "{{baseUrl}}/me/{{sessionId}}"
          auth: {type: bearer, token: "{{token}}"}
  - title: Second flow
    file: flows/second
    specFile: /local/other.yaml
    steps:
      - label: One
        request: {method: GET, url: "{{baseUrl}}/x"}
`

func parse(t *testing.T, text string) *serialization.Bundle {
	t.Helper()
	b, err := bundle.ParseText(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return b
}

func errorsOf(issues []bundle.Issue) []string {
	var out []string
	for _, i := range issues {
		if i.Severity == bundle.SeverityError {
			out = append(out, i.String())
		}
	}
	return out
}

func warningsOf(issues []bundle.Issue) []string {
	var out []string
	for _, i := range issues {
		if i.Severity == bundle.SeverityWarning {
			out = append(out, i.String())
		}
	}
	return out
}

func contains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func TestParseText_Tolerance(t *testing.T) {
	fenced := "```yaml\n" + good + "\n```\n"
	for name, text := range map[string]string{
		"plain":      good,
		"fenced":     fenced,
		"fence only": "```\n" + good + "```",
		"bom":        string(rune(0xFEFF)) + good,
		"crlf":       strings.ReplaceAll(good, "\n", "\r\n"),
		"padded":     "\n\n  \n" + good + "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			if b := parse(t, text); len(b.Scenarios) != 2 {
				t.Errorf("scenarios: %d", len(b.Scenarios))
			}
		})
	}
	if b := parse(t, `{"schemaVersion":"1","scenarios":[{"title":"S","steps":[{"label":"a","request":{"method":"GET","url":"http://x"}}]}]}`); len(b.Scenarios) != 1 {
		t.Error("json should be detected")
	}
	for _, empty := range []string{"", "   \n", "```yaml\n```"} {
		if _, err := bundle.ParseText(empty); err == nil {
			t.Errorf("%q should be rejected", empty)
		}
	}
}

func TestValidate_GoodBundleHasNoErrors(t *testing.T) {
	issues := bundle.Validate(parse(t, good), bundle.Options{})
	if errs := errorsOf(issues); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidate_Errors(t *testing.T) {
	cases := map[string]struct{ doc, want string }{
		"missing schemaVersion": {"scenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GET, url: http://x}}\n", "schemaVersion is required"},
		"wrong schemaVersion":   {"schemaVersion: \"2\"\nscenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GET, url: http://x}}\n", "must be \"1\""},
		"empty bundle":          {"schemaVersion: \"1\"\ntitle: nothing\n", "contains no environments"},
		"env without name":      {"schemaVersion: \"1\"\nenvironments:\n  - variables: []\n", "name is required"},
		"duplicate env":         {"schemaVersion: \"1\"\nenvironments:\n  - {name: A, variables: []}\n  - {name: a, variables: []}\n", "duplicate environment name"},
		"duplicate variable":    {"schemaVersion: \"1\"\nenvironments:\n  - name: A\n    variables: [{name: x, value: 1}, {name: x, value: 2}]\n", `variable "x" is defined twice`},
		"unnamed variable":      {"schemaVersion: \"1\"\nenvironments:\n  - name: A\n    variables: [{name: '', value: 1}]\n", "has no name"},
		"request without file":  {"schemaVersion: \"1\"\nrequests:\n  - {method: GET, url: http://x}\n", `"file" is required`},
		"duplicate request":     {"schemaVersion: \"1\"\nrequests:\n  - {file: a, method: GET, url: http://x}\n  - {file: a.yaml, method: GET, url: http://y}\n", "already used"},
		"bad method":            {"schemaVersion: \"1\"\nrequests:\n  - {file: a, method: FETCH, url: http://x}\n", "invalid or missing HTTP method"},
		"bad body type":         {"schemaVersion: \"1\"\nrequests:\n  - {file: a, method: GET, url: http://x, body: {type: xml, content: x}}\n", "unknown body type"},
		"bad auth type":         {"schemaVersion: \"1\"\nrequests:\n  - {file: a, method: GET, url: http://x, auth: {type: oauth}}\n", "unknown auth type"},
		"bad apikey location":   {"schemaVersion: \"1\"\nrequests:\n  - {file: a, method: GET, url: http://x, auth: {type: apikey, key: k, value: v, location: cookie}}\n", "location must be header or query"},
		"path escape":           {"schemaVersion: \"1\"\nrequests:\n  - {file: ../evil, method: GET, url: http://x}\n", `must not contain ".."`},
		"scenario no title":     {"schemaVersion: \"1\"\nscenarios:\n  - file: s\n    steps:\n      - {label: a, request: {method: GET, url: http://x}}\n", "scenario title is required"},
		"scenario no steps":     {"schemaVersion: \"1\"\nscenarios:\n  - {title: S, steps: []}\n", "scenario has no steps"},
		"duplicate scenario":    {"schemaVersion: \"1\"\nscenarios:\n  - {title: S, steps: [{label: a, request: {method: GET, url: http://x}}]}\n  - {title: s, steps: [{label: a, request: {method: GET, url: http://x}}]}\n", "already used"},
		"step no label":         {"schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - request: {method: GET, url: http://x}\n", "step label is required"},
		"unknown requestRef":    {"schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - {label: a, requestRef: missing}\n", `requestRef "missing" does not match any request`},
		"bad assertion op":      {"schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - label: a\n        request: {method: GET, url: http://x}\n        assertions: [{type: status, operator: approx, expected: 1}]\n", "unknown assertion operator"},
		"inline bad method":     {"schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GRAB, url: http://x}}\n", "invalid or missing HTTP method"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			errs := errorsOf(bundle.Validate(parse(t, c.doc), bundle.Options{}))
			if !contains(errs, c.want) {
				t.Errorf("want an error containing %q, got %v", c.want, errs)
			}
		})
	}
}

func TestValidate_ErrorsAreLocated(t *testing.T) {
	doc := "schemaVersion: \"1\"\nscenarios:\n  - title: Flow\n    steps:\n      - {label: Ok, request: {method: GET, url: http://x}}\n      - {label: Broken, requestRef: nowhere}\n"
	errs := errorsOf(bundle.Validate(parse(t, doc), bundle.Options{}))
	if !contains(errs, `scenarios[0] "Flow" › step 2 "Broken"`) {
		t.Errorf("errors should say where: %v", errs)
	}
}

func TestValidate_Warnings(t *testing.T) {
	doc := `
schemaVersion: "1"
environments:
  - {name: E, variables: [{name: baseUrl, value: "http://x"}]}
scenarios:
  - title: S
    steps:
      - label: Uses unknown variable
        request: {method: GET, url: "{{baseUrl}}/{{ghost}}"}
        timeoutMs: 100
        retry: {maxAttempts: 2}
        assertions: [{type: schema, schema: getThing}]
      - label: Extracts
        request: {method: GET, url: "{{baseUrl}}/a"}
        extractions: [{name: later, source: status}]
      - label: Uses extracted
        request: {method: GET, url: "{{baseUrl}}/{{later}}"}
      - label: Uses it before it exists
        request: {method: GET, url: "{{baseUrl}}/{{evenLater}}"}
        extractions: [{name: evenLater, source: status}]
      - label: External ref
        requestRef: ../requests/other.yaml
`
	issues := bundle.Validate(parse(t, doc), bundle.Options{})
	if errs := errorsOf(issues); len(errs) != 0 {
		t.Fatalf("warnings only, got errors: %v", errs)
	}
	w := warningsOf(issues)
	for _, want := range []string{
		"{{ghost}} is not defined",
		`schema assertion "getThing" needs a spec`, "{{evenLater}} is not defined",
		`requestRef "../requests/other.yaml" is not part of this bundle`,
	} {
		if !contains(w, want) {
			t.Errorf("missing warning %q in %v", want, w)
		}
	}
	if contains(w, "{{later}}") || contains(w, "{{baseUrl}}") {
		t.Errorf("defined and extracted variables must not warn: %v", w)
	}

	// A spec on the bundle, or supplied at import time, removes the schema warning.
	withSpec := bundle.Validate(parse(t, doc), bundle.Options{SpecFile: "/spec.yaml"})
	if contains(warningsOf(withSpec), "needs a spec") {
		t.Error("override spec should satisfy schema assertions")
	}
}

func TestValidate_VariableCheckSkippedWithoutEnvironments(t *testing.T) {
	doc := "schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GET, url: \"{{anything}}\"}}\n"
	if w := warningsOf(bundle.Validate(parse(t, doc), bundle.Options{})); contains(w, "not defined") {
		t.Errorf("the workspace environment may supply variables: %v", w)
	}
}

func prepare(t *testing.T, ws string, doc string, opts bundle.Options, s bundle.Strategy) *bundle.Plan {
	t.Helper()
	p, err := bundle.Prepare(parse(t, doc), ws, opts, s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func entryFor(p *bundle.Plan, rel string) *bundle.Entry {
	for i := range p.Entries {
		if p.Entries[i].RelPath == rel {
			return &p.Entries[i]
		}
	}
	return nil
}

func TestPrepare_LayoutAndRewrites(t *testing.T) {
	ws := t.TempDir()
	p := prepare(t, ws, good, bundle.Options{}, bundle.StrategyRename)
	if p.HasErrors() {
		t.Fatalf("%v", p.Issues)
	}
	want := []string{
		"requests/check-health.yaml", "requests/auth/login.yaml", "environments/staging.yaml",
		"scenarios/smoke-test.yaml", "scenarios/flows/second.yaml",
	}
	if len(p.Entries) != len(want) {
		t.Fatalf("entries: %+v", p.Entries)
	}
	for _, rel := range want {
		e := entryFor(p, rel)
		if e == nil || e.Action != bundle.ActionCreate || e.FinalPath != rel || len(e.Content) == 0 {
			t.Errorf("%s: %+v", rel, e)
		}
	}

	smoke := string(entryFor(p, "scenarios/smoke-test.yaml").Content)
	for _, s := range []string{"requestRef: ../requests/check-health.yaml", "requestRef: ../requests/auth/login.yaml", "specFile: https://api.example.com/openapi.json"} {
		if !strings.Contains(smoke, s) {
			t.Errorf("scenario should contain %q:\n%s", s, smoke)
		}
	}
	second := string(entryFor(p, "scenarios/flows/second.yaml").Content)
	if !strings.Contains(second, "specFile: /local/other.yaml") {
		t.Errorf("a scenario's own specFile wins over the bundle default:\n%s", second)
	}
	if env := string(entryFor(p, "environments/staging.yaml").Content); !strings.Contains(env, "secret: true") {
		t.Errorf("secrets must be preserved:\n%s", env)
	}
}

func TestPrepare_SpecFileOverrideReplacesEveryScenario(t *testing.T) {
	p := prepare(t, t.TempDir(), good, bundle.Options{SpecFile: "C:/specs/api.yaml"}, bundle.StrategyRename)
	for _, rel := range []string{"scenarios/smoke-test.yaml", "scenarios/flows/second.yaml"} {
		if c := string(entryFor(p, rel).Content); !strings.Contains(c, "specFile: C:/specs/api.yaml") {
			t.Errorf("%s should use the override:\n%s", rel, c)
		}
	}
}

func TestPrepare_NothingIsPlannedWhenThereAreErrors(t *testing.T) {
	p := prepare(t, t.TempDir(), "schemaVersion: \"1\"\nrequests:\n  - {method: GET, url: http://x}\n", bundle.Options{}, bundle.StrategyRename)
	if !p.HasErrors() || len(p.Entries) != 0 {
		t.Errorf("%+v", p)
	}
	if _, err := p.Write(t.TempDir()); err == nil {
		t.Error("writing an invalid plan must fail")
	}
}

func TestPrepare_UnknownStrategy(t *testing.T) {
	if _, err := bundle.Prepare(parse(t, good), t.TempDir(), bundle.Options{}, "merge"); err == nil {
		t.Error("unknown strategy should be rejected")
	}
}

func seedExisting(t *testing.T, ws string) {
	t.Helper()
	for _, rel := range []string{"requests/check-health.yaml", "scenarios/smoke-test.yaml", "environments/staging.yaml"} {
		abs := filepath.Join(ws, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(abs), 0o755)
		os.WriteFile(abs, []byte("existing "+rel), 0o644)
	}
}

func TestPrepareAndWrite_ConflictStrategies(t *testing.T) {
	t.Run("rename keeps both and repoints requestRefs at the renamed request", func(t *testing.T) {
		ws := t.TempDir()
		seedExisting(t, ws)
		p := prepare(t, ws, good, bundle.Options{}, bundle.StrategyRename)

		req := entryFor(p, "requests/check-health.yaml")
		if req.Action != bundle.ActionRename || req.FinalPath != "requests/check-health-2.yaml" {
			t.Errorf("request: %+v", req)
		}
		scn := entryFor(p, "scenarios/smoke-test.yaml")
		if scn.FinalPath != "scenarios/smoke-test-2.yaml" || !strings.Contains(string(scn.Content), "requestRef: ../requests/check-health-2.yaml") {
			t.Errorf("scenario must reference the renamed request: %+v\n%s", scn, scn.Content)
		}

		res, err := p.Write(ws)
		if err != nil {
			t.Fatal(err)
		}
		if res.Renamed != 3 || res.Created != 2 || res.Skipped != 0 || res.Overwritten != 0 || len(res.Written) != 5 {
			t.Errorf("result: %+v", res)
		}
		if b, _ := os.ReadFile(filepath.Join(ws, "requests", "check-health.yaml")); string(b) != "existing requests/check-health.yaml" {
			t.Error("the existing file must be untouched")
		}
		if _, err := os.Stat(filepath.Join(ws, "requests", "check-health-2.yaml")); err != nil {
			t.Error("renamed file should exist")
		}
	})

	t.Run("a second import keeps counting up", func(t *testing.T) {
		ws := t.TempDir()
		for i := 0; i < 3; i++ {
			if _, err := prepare(t, ws, good, bundle.Options{}, bundle.StrategyRename).Write(ws); err != nil {
				t.Fatal(err)
			}
		}
		for _, rel := range []string{"scenarios/smoke-test.yaml", "scenarios/smoke-test-2.yaml", "scenarios/smoke-test-3.yaml"} {
			if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
				t.Errorf("%s: %v", rel, err)
			}
		}
	})

	t.Run("skip leaves existing files alone and points refs at them", func(t *testing.T) {
		ws := t.TempDir()
		seedExisting(t, ws)
		p := prepare(t, ws, good, bundle.Options{}, bundle.StrategySkip)
		res, err := p.Write(ws)
		if err != nil {
			t.Fatal(err)
		}
		if res.Skipped != 3 || res.Created != 2 {
			t.Errorf("result: %+v", res)
		}
		if b, _ := os.ReadFile(filepath.Join(ws, "scenarios", "smoke-test.yaml")); string(b) != "existing scenarios/smoke-test.yaml" {
			t.Error("skipped file was modified")
		}
	})

	t.Run("overwrite replaces existing files", func(t *testing.T) {
		ws := t.TempDir()
		seedExisting(t, ws)
		res, err := prepare(t, ws, good, bundle.Options{}, bundle.StrategyOverwrite).Write(ws)
		if err != nil {
			t.Fatal(err)
		}
		if res.Overwritten != 3 || res.Created != 2 {
			t.Errorf("result: %+v", res)
		}
		if b, _ := os.ReadFile(filepath.Join(ws, "environments", "staging.yaml")); !strings.Contains(string(b), "baseUrl") {
			t.Error("file should have been replaced")
		}
	})

	t.Run("default strategy is rename", func(t *testing.T) {
		ws := t.TempDir()
		seedExisting(t, ws)
		p, err := bundle.Prepare(parse(t, good), ws, bundle.Options{}, "")
		if err != nil || entryFor(p, "environments/staging.yaml").Action != bundle.ActionRename {
			t.Errorf("%v %+v", err, p)
		}
	})
}

func TestWrite_NoTempFilesAndImportedFilesParse(t *testing.T) {
	ws := t.TempDir()
	res, err := prepare(t, ws, good, bundle.Options{}, bundle.StrategyRename).Write(ws)
	if err != nil {
		t.Fatal(err)
	}
	filepath.Walk(ws, func(p string, info os.FileInfo, _ error) error {
		if strings.HasSuffix(p, ".tmp") {
			t.Errorf("temp file left behind: %s", p)
		}
		return nil
	})

	for _, rel := range res.Written {
		data, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case strings.HasPrefix(rel, "requests/"):
			_, err = serialization.ParseRequestFile(data, rel)
		case strings.HasPrefix(rel, "environments/"):
			_, err = serialization.ParseEnvironmentFile(data, rel)
		default:
			_, err = serialization.ParseScenarioFile(data, rel)
		}
		if err != nil {
			t.Errorf("%s does not parse as a normal file: %v", rel, err)
		}
	}
}
