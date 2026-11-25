package workspaceinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func write(t *testing.T, root, rel, content string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const okScenario = `schemaVersion: "1"
title: Users
specFile: ../specs/api.yaml
steps:
  - label: List
    request:
      method: GET
      url: "{{baseUrl}}/users"
    assertions:
      - {type: status, operator: eq, expected: 200}
`

func ids(r Report) []string {
	var out []string
	for _, a := range r.Attention {
		out = append(out, a.ID)
	}
	return out
}

func has(r Report, id string) *Attention {
	for i := range r.Attention {
		if r.Attention[i].ID == id {
			return &r.Attention[i]
		}
	}
	return nil
}

func TestInspect_EmptyWorkspace(t *testing.T) {
	r := Inspect(t.TempDir(), nil, "")
	if r.Requests != 0 || r.Scenarios != 0 || r.Environments != 0 || len(r.Attention) != 0 {
		t.Fatalf("empty workspace: %+v", r)
	}
	if r.EnvVars == nil || r.Specs == nil || r.Attention == nil {
		t.Fatal("slices must be non-nil so the page gets [] and not null")
	}
}

func TestInspect_CountsAndHealthyWorkspace(t *testing.T) {
	root := t.TempDir()
	write(t, root, "requests/a.yaml", "x")
	write(t, root, "requests/sub/b.json", "{}")
	write(t, root, "requests/readme.md", "x")
	envPath := write(t, root, "environments/local.yaml", "x")
	write(t, root, "scenarios/users.yaml", okScenario)
	write(t, root, "specs/api.yaml", "openapi: 3.0.0")

	env := &domain.Environment{Name: "Local", Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "http://x"}}}
	r := Inspect(root, env, envPath)
	if r.Requests != 2 || r.Scenarios != 1 || r.Environments != 1 {
		t.Fatalf("counts: %+v", r)
	}
	if len(r.Attention) != 0 {
		t.Fatalf("a healthy workspace has nothing to attend to: %v", ids(r))
	}
	if len(r.Specs) != 1 || r.Specs[0].State != SpecFound {
		t.Fatalf("specs: %+v", r.Specs)
	}
	if r.EnvironmentName != "Local" {
		t.Fatalf("environment name %q", r.EnvironmentName)
	}
}

func TestInspect_NoEnvironmentWhenPlaceholdersNeedOne(t *testing.T) {
	root := t.TempDir()
	write(t, root, "environments/local.yaml", "x")
	write(t, root, "scenarios/users.yaml", okScenario)
	write(t, root, "specs/api.yaml", "x")
	r := Inspect(root, nil, "")
	a := has(r, ItemNoEnvironment)
	if a == nil || !strings.Contains(a.Where, "{{baseUrl}}") {
		t.Fatalf("want no-environment naming {{baseUrl}}, got %v", ids(r))
	}
	if r.Attention[0].ID != ItemNoEnvironment {
		t.Fatal("no-environment comes first")
	}
}

func TestInspect_NoEnvironmentIsQuietWhenNothingNeedsOne(t *testing.T) {
	root := t.TempDir()
	write(t, root, "environments/local.yaml", "x")
	write(t, root, "scenarios/s.yaml", `schemaVersion: "1"
title: S
variables:
  - name: host
    value: http://x
steps:
  - label: One
    request: {method: GET, url: "{{host}}/a"}
    extractions:
      - {name: id, source: jsonpath, path: $.id}
  - label: Two
    request: {method: GET, url: "{{host}}/a/{{id}}/{{$guid}}"}
`)
	if r := Inspect(root, nil, ""); has(r, ItemNoEnvironment) != nil {
		t.Fatalf("scenario variables, extractions and functions define everything: %v", ids(r))
	}
	empty := t.TempDir()
	write(t, empty, "scenarios/users.yaml", okScenario)
	if r := Inspect(empty, nil, ""); has(r, ItemNoEnvironment) != nil {
		t.Fatal("a workspace with no environment files must not be told to pick one")
	}
}

func TestInspect_UnsetProcessVariableAndSecretsNeverShown(t *testing.T) {
	t.Setenv("CLRSIG_TEST_SET", "yes")
	os.Unsetenv("CLRSIG_TEST_UNSET")
	root := t.TempDir()
	env := &domain.Environment{Name: "Staging", Variables: []domain.EnvironmentVariable{
		{Name: "token", Value: "${CLRSIG_TEST_UNSET}", Secret: true},
		{Name: "user", Value: "${CLRSIG_TEST_SET}"},
		{Name: "empty", Value: ""},
		{Name: "literal", Value: "hunter2-secret", Secret: true},
	}}
	r := Inspect(root, env, filepath.Join(root, "environments", "staging.yaml"))
	a := has(r, ItemUnsetEnvVar)
	if a == nil || !strings.Contains(a.What, "CLRSIG_TEST_UNSET") || !strings.Contains(a.Where, `"token"`) {
		t.Fatalf("unset variable item: %+v", a)
	}
	got := map[string]EnvVar{}
	for _, v := range r.EnvVars {
		got[v.Name] = v
	}
	if got["token"].Set || got["token"].UnsetRef != "CLRSIG_TEST_UNSET" || !got["user"].Set || got["empty"].Set || !got["literal"].Set {
		t.Fatalf("env vars: %+v", got)
	}
	blob := a.What + a.Where + a.Why + a.Fix
	for _, v := range r.EnvVars {
		blob += v.Name + v.UnsetRef
	}
	if strings.Contains(blob, "hunter2") {
		t.Fatal("a secret value must never be in the report")
	}
}

func TestInspect_SpecStates(t *testing.T) {
	root := t.TempDir()
	outside := write(t, t.TempDir(), "other.yaml", "x")
	write(t, root, "specs/ok.yaml", "x")
	for name, ref := range map[string]string{"a": "../specs/ok.yaml", "b": "../specs/gone.yaml", "c": outside, "d": "https://example.com/openapi.yaml"} {
		write(t, root, "scenarios/"+name+".yaml", strings.Replace(okScenario, "../specs/api.yaml", filepath.ToSlash(ref), 1))
	}
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "x"}}}
	r := Inspect(root, env, "e.yaml")
	states := map[string]string{}
	for _, s := range r.Specs {
		states[s.Ref] = s.State
	}
	if states["../specs/ok.yaml"] != SpecFound || states["../specs/gone.yaml"] != SpecMissing ||
		states[filepath.ToSlash(outside)] != SpecOutside || states["https://example.com/openapi.yaml"] != SpecURL {
		t.Fatalf("states: %v", states)
	}
	miss := has(r, ItemSpecMissing)
	if miss == nil || !strings.Contains(miss.Where, "scenarios/b.yaml") || miss.Target.Kind != "scenario" {
		t.Fatalf("missing item: %+v", miss)
	}
	out := has(r, ItemSpecOutside)
	if out == nil || out.Severity != SeverityInfo {
		t.Fatalf("outside item: %+v", out)
	}
	if len(ids(r)) != 2 {
		t.Fatalf("a found spec and a URL spec are not problems: %v", ids(r))
	}
}

func TestInspect_BrokenScenariosAndRequestRefs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "requests/exists.yaml", "x")
	write(t, root, "scenarios/broken.yaml", "title: [unclosed")
	write(t, root, "scenarios/nosteps.yaml", "schemaVersion: \"1\"\ntitle: Empty\n")
	write(t, root, "scenarios/refs.yaml", `schemaVersion: "1"
title: Refs
steps:
  - label: Good
    requestRef: exists.yaml
  - label: Bad
    requestRef: gone.yaml
`)
	r := Inspect(root, nil, "")
	var invalid []string
	for _, a := range r.Attention {
		if a.ID == ItemScenarioInvalid {
			invalid = append(invalid, a.Where)
		}
	}
	if len(invalid) < 2 {
		t.Fatalf("want problems for broken.yaml and nosteps.yaml, got %v", invalid)
	}
	ref := has(r, ItemRequestRef)
	if ref == nil || !strings.Contains(ref.What, "gone.yaml") || !strings.Contains(ref.Where, `"Bad"`) {
		t.Fatalf("request ref item: %+v", ref)
	}
	for _, a := range r.Attention {
		if a.ID == ItemRequestRef && strings.Contains(a.What, "exists.yaml") {
			t.Fatal("a request found under requests/ is not missing")
		}
	}
}

func TestInspect_NeverWrites(t *testing.T) {
	root := t.TempDir()
	write(t, root, "scenarios/users.yaml", okScenario)
	before := snapshot(t, root)
	Inspect(root, nil, "")
	if after := snapshot(t, root); after != before {
		t.Fatalf("workspace changed:\n%s\n---\n%s", before, after)
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			b.WriteString(p + "|" + info.ModTime().String() + "\n")
		}
		return nil
	})
	return b.String()
}

func TestInspect_SelectedEnvironmentThatDoesNotDefineAPlaceholder(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/api.yaml", "x")
	write(t, root, "scenarios/users.yaml", `schemaVersion: "1"
title: Users
specFile: ../specs/api.yaml
steps:
  - label: List
    request:
      method: GET
      url: "{{baseUrl}}/users"
      headers:
        - {name: Authorization, value: "Bearer {{token}}"}
`)
	envPath := filepath.Join(root, "environments", "staging.yaml")
	env := &domain.Environment{Name: "Staging", Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "http://x"}}}

	r := Inspect(root, env, envPath)
	a := has(r, ItemUndefinedValue)
	if a == nil || !strings.Contains(a.What, "{{token}}") || strings.Contains(a.What, "{{baseUrl}}") || !strings.Contains(a.Where, "scenarios/users.yaml") {
		t.Fatalf("want the environment named as lacking {{token}} only, got %+v", a)
	}
	if a.Target.Kind != "environment" || a.Target.Path != envPath {
		t.Fatalf("target: %+v", a.Target)
	}

	env.Variables = append(env.Variables, domain.EnvironmentVariable{Name: "token", Value: "t", Secret: true})
	if r := Inspect(root, env, envPath); has(r, ItemUndefinedValue) != nil {
		t.Fatalf("every placeholder is defined now: %v", ids(r))
	}
}

func TestInspect_OnlyWhatIsSentCountsAsAPlaceholderUse(t *testing.T) {
	root := t.TempDir()
	write(t, root, "requests/login.yaml", "schemaVersion: \"1\"\nname: Login\nmethod: POST\nurl: \"{{baseUrl}}/login\"\nbody:\n  type: json\n  content: '{\"user\":\"{{fromRequestFile}}\"}'\n")
	write(t, root, "scenarios/s.yaml", `schemaVersion: "1"
title: "About {{inTitle}}"
description: "uses {{inDescription}}"
steps:
  - label: "Login with {{inLabel}}"
    notes: "set {{inNotes}} first"
    requestRef: login.yaml
  - label: Disabled
    disabled: true
    request: {method: GET, url: "{{baseUrl}}/{{inDisabledStep}}"}
`)
	env := &domain.Environment{Name: "E", Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "x"}}}
	r := Inspect(root, env, filepath.Join(root, "environments", "e.yaml"))
	a := has(r, ItemUndefinedValue)
	if a == nil || !strings.Contains(a.What, "{{fromRequestFile}}") {
		t.Fatalf("a placeholder inside the referenced request file must count: %+v", a)
	}
	for _, not := range []string{"inTitle", "inDescription", "inLabel", "inNotes", "inDisabledStep"} {
		if strings.Contains(a.What, not) {
			t.Errorf("{{%s}} is never sent and must not count", not)
		}
	}
}

func TestInspect_OneSpecNamedDifferentlyKeepsEverySpelling(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/api.yaml", "x")
	write(t, root, "scenarios/a.yaml", strings.Replace(okScenario, "../specs/api.yaml", "../specs/api.yaml", 1))
	write(t, root, "scenarios/deep/b.yaml", strings.Replace(okScenario, "../specs/api.yaml", "../../specs/api.yaml", 1))
	write(t, root, "scenarios/deep/c.yaml", strings.Replace(okScenario, "../specs/api.yaml", "../../specs/api.yaml", 1))
	r := Inspect(root, nil, "")
	if len(r.Specs) != 1 {
		t.Fatalf("one file is one spec row, got %d: %+v", len(r.Specs), r.Specs)
	}
	sp := r.Specs[0]
	if len(sp.Refs) != 2 || sp.Refs[0] != "../specs/api.yaml" || sp.Refs[1] != "../../specs/api.yaml" || len(sp.Scenarios) != 3 {
		t.Fatalf("spec: %+v", sp)
	}
}

func TestInspect_ASpecReachedThroughALinkOutOfTheWorkspaceIsOutside(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	write(t, elsewhere, "api.yaml", "x")
	if err := os.Symlink(elsewhere, filepath.Join(root, "linked")); err != nil {
		t.Skip("symbolic links are not available here:", err)
	}
	write(t, root, "scenarios/a.yaml", strings.Replace(okScenario, "../specs/api.yaml", "../linked/api.yaml", 1))
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{{Name: "baseUrl", Value: "x"}}}
	r := Inspect(root, env, "e.yaml")
	if len(r.Specs) != 1 || r.Specs[0].State != SpecOutside {
		t.Fatalf("a link that leaves the workspace must be reported as outside: %+v", r.Specs)
	}
}

func TestInspect_ProjectFile(t *testing.T) {
	root := t.TempDir()
	if r := Inspect(root, nil, ""); r.ProjectFile.Found || len(r.ProjectFile.Keys) != 0 || len(r.Attention) != 0 {
		t.Fatalf("no file is not a problem: %+v", r)
	}

	write(t, root, "specs/api.yaml", "x")
	write(t, root, "environments/local.yaml", "x")
	write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\nspec: specs/api.yaml\nenvironment: environments/local.yaml\nbaseUrl: http://localhost:8080\nconform:\n  readOnly: true\n")
	r := Inspect(root, nil, "")
	pf := r.ProjectFile
	if !pf.Found || pf.Path != "clrsig.yaml" || pf.Error != "" || strings.Join(pf.Keys, ",") != "spec,environment,baseUrl,conform.readOnly" {
		t.Fatalf("a good file: %+v", pf)
	}
	if has(r, ItemProjectFileInvalid) != nil || has(r, ItemProjectFilePath) != nil {
		t.Errorf("a good file raises nothing: %v", ids(r))
	}
}

func TestInspect_ProjectFileProblems(t *testing.T) {
	root := t.TempDir()
	write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\nbaseURL: x\n")
	r := Inspect(root, nil, "")
	a := has(r, ItemProjectFileInvalid)
	if a == nil || r.ProjectFile.Error == "" || !strings.Contains(a.Why, "baseURL") || a.Fix == "" || a.Target.Kind != "projectfile" {
		t.Fatalf("an unusable file says what, where, why and how to fix: %+v / %+v", r.ProjectFile, a)
	}

	root = t.TempDir()
	write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\nspec: specs/gone.yaml\nenvironment: https://example.com/x\n")
	r = Inspect(root, nil, "")
	a = has(r, ItemProjectFilePath)
	if a == nil || !strings.Contains(a.What, "specs/gone.yaml") || !strings.Contains(a.Where, "key spec") {
		t.Fatalf("a spec that is not there is named: %v %+v", ids(r), a)
	}

	root = t.TempDir()
	write(t, root, "clrsig.yaml", "schemaVersion: \"1\"\n")
	write(t, root, "clrsig.json", `{"schemaVersion":"1"}`)
	if r := Inspect(root, nil, ""); has(r, ItemProjectFileInvalid) == nil {
		t.Errorf("two project files in the folder is a problem: %v", ids(r))
	}
}

func TestInspect_ProjectFileIsOnlyReadInsideTheWorkspace(t *testing.T) {
	outer := t.TempDir()
	write(t, outer, "clrsig.yaml", "schemaVersion: \"1\"\n")
	root := filepath.Join(outer, "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if r := Inspect(root, nil, ""); r.ProjectFile.Found {
		t.Errorf("a file above the workspace is not read: %+v", r.ProjectFile)
	}
}

func TestInspect_WeakScenariosAreInfoItems(t *testing.T) {
	root := t.TempDir()
	write(t, root, "scenarios/weak.yaml", `schemaVersion: "1"
title: Weak
steps:
  - label: Ping
    request: {method: GET, url: "http://x/ping"}
  - label: Login
    request: {method: POST, url: "http://x/login"}
    extractions:
      - {name: spare, source: jsonpath, path: $.spare}
`)
	write(t, root, "scenarios/good.yaml", `schemaVersion: "1"
title: Good
steps:
  - label: Ping
    request: {method: GET, url: "http://x/ping"}
    assertions:
      - {type: status, operator: eq, expected: 200}
`)
	r := Inspect(root, nil, "")
	a := has(r, ItemScenarioWeak)
	if a == nil {
		t.Fatalf("a weak scenario is listed: %v", ids(r))
	}
	if a.Severity != SeverityInfo || a.Where != "scenarios/weak.yaml" || a.Target.Kind != "scenario" {
		t.Errorf("%+v", a)
	}
	if !strings.Contains(a.What, "2 things to improve") || !strings.Contains(a.Why, "[step 1] Ping: has no assertions") || !strings.Contains(a.Why, `extracts "spare"`) || !strings.Contains(a.Fix, "clrsig validate --strict") {
		t.Errorf("what, why and fix: %+v", a)
	}
	count := 0
	for _, it := range r.Attention {
		if it.ID == ItemScenarioWeak {
			count++
		}
	}
	if count != 1 {
		t.Errorf("one item per weak scenario, none for the good one: %d", count)
	}
}

// A value a step extracts and a later step uses only inside its request file (requestRef) is used.
func TestInspect_AnExtractionUsedInAReferencedRequestIsNotWeak(t *testing.T) {
	root := t.TempDir()
	write(t, root, "requests/me.yaml", "schemaVersion: \"1\"\nname: Me\nmethod: GET\nurl: \"http://x/me\"\nheaders:\n  - name: Authorization\n    value: \"Bearer {{token}}\"\n")
	write(t, root, "scenarios/s.yaml", `schemaVersion: "1"
title: Refs
steps:
  - label: Login
    request: {method: POST, url: "http://x/login"}
    extractions:
      - {name: token, source: jsonpath, path: $.token}
  - label: Me
    requestRef: me.yaml
    assertions:
      - {type: status, operator: eq, expected: 200}
`)
	if a := has(Inspect(root, nil, ""), ItemScenarioWeak); a != nil {
		t.Errorf("the token is used by the referenced request: %+v", a)
	}
}
