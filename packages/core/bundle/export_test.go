package bundle_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/bundle"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

func write(t *testing.T, ws, rel, content string) {
	t.Helper()
	abs := filepath.Join(ws, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedWorkspace(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "my-project")
	write(t, ws, "environments/staging.yaml", `schemaVersion: "1"
name: Staging
variables:
  - {name: baseUrl, value: "https://staging.example.com"}
  - {name: token, value: real-secret-token, secret: true}
  - {name: empty, value: "", secret: true}
`)
	write(t, ws, "environments/local.json", `{"schemaVersion":"1","name":"Local","variables":[{"name":"baseUrl","value":"http://localhost"}]}`)
	write(t, ws, "requests/check-health.yaml", "schemaVersion: \"1\"\ntitle: Check health\nmethod: GET\nurl: \"{{baseUrl}}/health\"\ntimeout: 5s\n")
	write(t, ws, "requests/auth/login.yaml", "schemaVersion: \"1\"\nmethod: POST\nurl: \"{{baseUrl}}/login\"\nbody:\n  type: json\n  content: '{\"u\":\"x\"}'\nauth:\n  type: bearer\n  token: \"{{token}}\"\n")
	write(t, ws, "requests/.hidden/skip.yaml", "method: GET\nurl: http://x\n")
	write(t, ws, "requests/broken.yaml", "method: [")
	write(t, ws, "requests/notes.txt", "not a request")
	write(t, ws, "scenarios/smoke.yaml", `schemaVersion: "1"
title: Smoke
specFile: https://api.example.com/openapi.json
steps:
  - label: Health
    requestRef: ../requests/check-health.yaml
    assertions: [{type: status, operator: eq, expected: 200}]
  - label: Login
    requestRef: auth/login.yaml
  - label: Inline
    request: {method: GET, url: "{{baseUrl}}/me"}
`)
	write(t, ws, "scenarios/flows/second.yaml", `schemaVersion: "1"
title: Second
specFile: https://api.example.com/openapi.json
steps:
  - label: One
    request: {method: GET, url: "{{baseUrl}}/x"}
    validateSpec: false
`)
	write(t, ws, "scenarios/dangling.yaml", `schemaVersion: "1"
title: Dangling
specFile: https://api.example.com/openapi.json
steps:
  - {label: A, requestRef: ../requests/missing.yaml}
`)
	return ws
}

func TestExport_CollectsEverythingAndReportsWhatItSkipped(t *testing.T) {
	res, err := bundle.Export(seedWorkspace(t), bundle.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b := res.Bundle
	if b.Title != "my-project" || b.SchemaVersion != "1" {
		t.Errorf("header: %+v", b)
	}
	if len(b.Environments) != 2 || b.Environments[0].File != "local" || b.Environments[1].File != "staging" {
		t.Errorf("environments (sorted, any extension): %+v", b.Environments)
	}
	if len(b.Requests) != 2 || b.Requests[0].File != "auth/login" || b.Requests[1].File != "check-health" {
		t.Errorf("requests (recursive, hidden skipped): %+v", b.Requests)
	}
	if len(b.Scenarios) != 3 || b.Scenarios[0].File != "dangling" || b.Scenarios[1].File != "flows/second" || b.Scenarios[2].File != "smoke" {
		t.Errorf("scenarios: %+v", b.Scenarios)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Path != "requests/broken.yaml" || res.Skipped[0].Reason == "" {
		t.Errorf("only the unparseable file should be reported: %+v", res.Skipped)
	}
}

func TestExport_RewritesRequestRefsToBundleNames(t *testing.T) {
	res, err := bundle.Export(seedWorkspace(t), bundle.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var smoke, dangling *serialization.BundleScenario
	for i := range res.Bundle.Scenarios {
		switch res.Bundle.Scenarios[i].File {
		case "smoke":
			smoke = &res.Bundle.Scenarios[i]
		case "dangling":
			dangling = &res.Bundle.Scenarios[i]
		}
	}
	if got := smoke.Scenario.Steps[0].RequestRef; got != "check-health" {
		t.Errorf("a ref relative to the scenario folder: %q", got)
	}
	if got := smoke.Scenario.Steps[1].RequestRef; got != "auth/login" {
		t.Errorf("a ref relative to requests/: %q", got)
	}
	if got := dangling.Scenario.Steps[0].RequestRef; got != "../requests/missing.yaml" {
		t.Errorf("an unresolvable ref is left alone: %q", got)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "missing.yaml") {
		t.Errorf("unresolvable refs should be reported: %v", res.Warnings)
	}
}

func TestExport_SecretsAreRedactedByDefault(t *testing.T) {
	ws := seedWorkspace(t)
	res, err := bundle.Export(ws, bundle.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.RedactedSecrets != 1 {
		t.Errorf("only the non-empty secret counts: %d", res.RedactedSecrets)
	}
	data, _ := res.Marshal()
	if strings.Contains(string(data), "real-secret-token") {
		t.Fatalf("a secret leaked into the export:\n%s", data)
	}
	if !strings.Contains(string(data), bundle.SecretPlaceholder) {
		t.Error("the placeholder should replace the secret")
	}

	kept, err := bundle.Export(ws, bundle.ExportOptions{KeepSecrets: true})
	if err != nil {
		t.Fatal(err)
	}
	keptData, _ := kept.Marshal()
	if !strings.Contains(string(keptData), "real-secret-token") || kept.RedactedSecrets != 0 {
		t.Error("KeepSecrets should leave values untouched")
	}
	if got, _ := os.ReadFile(filepath.Join(ws, "environments", "staging.yaml")); !strings.Contains(string(got), "real-secret-token") {
		t.Error("exporting must never modify the workspace")
	}
}

func TestExport_HoistsASharedSpecFile(t *testing.T) {
	ws := seedWorkspace(t)
	res, _ := bundle.Export(ws, bundle.ExportOptions{})
	if res.Bundle.SpecFile != "https://api.example.com/openapi.json" {
		t.Errorf("a spec shared by every scenario moves to the bundle: %q", res.Bundle.SpecFile)
	}
	for _, s := range res.Bundle.Scenarios {
		if s.Scenario.SpecFile != "" {
			t.Errorf("scenario %s should inherit it: %q", s.File, s.Scenario.SpecFile)
		}
	}

	write(t, ws, "scenarios/other.yaml", "schemaVersion: \"1\"\ntitle: Other\nspecFile: /local/spec.yaml\nsteps:\n  - {label: A, request: {method: GET, url: \"http://x\"}}\n")
	mixed, _ := bundle.Export(ws, bundle.ExportOptions{})
	if mixed.Bundle.SpecFile != "" {
		t.Error("different specs must stay on their scenarios")
	}
	for _, s := range mixed.Bundle.Scenarios {
		if s.File == "other" && s.Scenario.SpecFile != "/local/spec.yaml" {
			t.Errorf("own spec lost: %q", s.Scenario.SpecFile)
		}
	}

	noSpec := t.TempDir()
	write(t, noSpec, "scenarios/a.yaml", "schemaVersion: \"1\"\ntitle: A\nsteps:\n  - {label: A, request: {method: GET, url: \"http://x\"}}\n")
	plain, _ := bundle.Export(noSpec, bundle.ExportOptions{})
	if plain.Bundle.SpecFile != "" {
		t.Error("no spec stays no spec")
	}
}

func TestExport_EmptyWorkspaceIsAnError(t *testing.T) {
	if _, err := bundle.Export(t.TempDir(), bundle.ExportOptions{}); err == nil || !strings.Contains(err.Error(), "nothing") && !strings.Contains(err.Error(), "no environments") {
		t.Errorf("got %v", err)
	}
	ws := t.TempDir()
	write(t, ws, "requests/broken.yaml", "method: [")
	res, err := bundle.Export(ws, bundle.ExportOptions{})
	if err == nil || len(res.Skipped) != 1 {
		t.Errorf("a workspace with only broken files exports nothing: %v %+v", err, res)
	}
}

func TestExport_DuplicateStemsGetDistinctNames(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "requests/a.yaml", "method: GET\nurl: http://x/1\n")
	write(t, ws, "requests/a.json", `{"method":"GET","url":"http://x/2"}`)
	res, err := bundle.Export(ws, bundle.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Bundle.Requests) != 2 || res.Bundle.Requests[0].File == res.Bundle.Requests[1].File {
		t.Errorf("names must be unique: %+v", res.Bundle.Requests)
	}
	data, _ := res.Marshal()
	b, err := bundle.ParseText(string(data))
	if err != nil || len(bundle.Validate(b, bundle.Options{})) != 0 {
		t.Errorf("the export must be a valid bundle: %v %v", err, bundle.Validate(b, bundle.Options{}))
	}
}

// Export then import into a fresh workspace: the result must be equivalent files.
func TestExport_RoundTripsThroughImport(t *testing.T) {
	src := seedWorkspace(t)
	res, err := bundle.Export(src, bundle.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := res.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	b, err := bundle.ParseText(string(data))
	if err != nil {
		t.Fatalf("the export must parse strictly: %v\n%s", err, data)
	}
	if errs := errorsOf(bundle.Validate(b, bundle.Options{})); len(errs) != 0 {
		// the dangling scenario's requestRef is a path outside the bundle: a warning, not an error
		t.Fatalf("the export must validate: %v", errs)
	}

	dst := t.TempDir()
	plan, err := bundle.Prepare(b, dst, bundle.Options{}, bundle.StrategyRename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Write(dst); err != nil {
		t.Fatal(err)
	}

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		return string(b)
	}

	smoke, err := serialization.ParseScenarioFile([]byte(read("scenarios/smoke.yaml")), "smoke.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if smoke.SpecFile != "https://api.example.com/openapi.json" || len(smoke.Steps) != 3 {
		t.Errorf("scenario: %+v", smoke)
	}
	if smoke.Steps[0].RequestRef != "../requests/check-health.yaml" || smoke.Steps[1].RequestRef != "../requests/auth/login.yaml" {
		t.Errorf("requestRefs must point at the re-created requests: %q %q", smoke.Steps[0].RequestRef, smoke.Steps[1].RequestRef)
	}
	for _, ref := range []string{smoke.Steps[0].RequestRef, smoke.Steps[1].RequestRef} {
		if _, err := os.Stat(filepath.Join(dst, "scenarios", filepath.FromSlash(ref))); err != nil {
			t.Errorf("%s does not resolve from the scenario folder: %v", ref, err)
		}
	}

	second, _ := serialization.ParseScenarioFile([]byte(read("scenarios/flows/second.yaml")), "second.yaml")
	if second.Steps[0].ValidateSpec == nil || *second.Steps[0].ValidateSpec {
		t.Error("validateSpec: false must survive")
	}

	login, _ := serialization.ParseRequestFile([]byte(read("requests/auth/login.yaml")), "login.yaml")
	if login.Auth == nil || login.Auth.Bearer == nil || login.Auth.Bearer.Token != "{{token}}" || login.Body == nil {
		t.Errorf("request lost data: %+v", login)
	}
	health, _ := serialization.ParseRequestFile([]byte(read("requests/check-health.yaml")), "x.yaml")
	if health.Title != "Check health" || health.Timeout.Seconds() != 5 {
		t.Errorf("request lost data: %+v", health)
	}

	env, _ := serialization.ParseEnvironmentFile([]byte(read("environments/staging.yaml")), "staging.yaml")
	if env.Name != "Staging" || len(env.Variables) != 3 || env.Variables[1].Value != bundle.SecretPlaceholder || !env.Variables[1].Secret {
		t.Errorf("environment: %+v", env)
	}
}

func TestWriteBundle_RoundTripsAndRejectsUnknownFormats(t *testing.T) {
	b := parse(t, good)
	for _, f := range []string{"yaml", "yml", "json"} {
		data, err := serialization.WriteBundle(b, f)
		if err != nil {
			t.Fatal(err)
		}
		again, err := serialization.ParseBundle(data, f)
		if err != nil {
			t.Fatalf("%s: %v\n%s", f, err, data)
		}
		if len(again.Environments) != 1 || len(again.Requests) != 2 || len(again.Scenarios) != 2 || again.SpecFile != b.SpecFile {
			t.Errorf("%s round trip lost items: %+v", f, again)
		}
	}
	if _, err := serialization.WriteBundle(b, "toml"); err == nil {
		t.Error("unknown format")
	}
	empty := &serialization.Bundle{}
	if data, _ := serialization.WriteBundle(empty, "yaml"); !strings.Contains(string(data), `schemaVersion: "1"`) {
		t.Errorf("a missing version defaults to 1:\n%s", data)
	}
}
