package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

const pingSpec = `openapi: "3.0.3"
info: {title: ping-api, version: "1"}
paths:
  /ping:
    get:
      operationId: getPing
      responses:
        "200": {description: ok}
  /other:
    get:
      operationId: getOther
      responses:
        "200": {description: ok}
`

func newSS(t *testing.T) (*ScenarioService, *WorkspaceService) {
	t.Helper()
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	return NewScenarioService(ws, NewSettingsService()), ws
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func pingScenario(url, extra string) string {
	return fmt.Sprintf(`schemaVersion: "1"
title: Ping flow
steps:
  - label: ping
    request:
      method: GET
      url: %s/ping
    assertions:
      - type: status
        operator: eq
        expected: 200
    extractions:
      - name: code
        source: status
%s`, url, extra)
}

func TestToSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Hello World":     "hello-world",
		"  --A  b__c--  ": "a-b-c",
		"Ünï 42":          "ünï-42",
		"!!!":             "",
		"":                "",
	} {
		if got := toSlug(in); got != want {
			t.Errorf("toSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScenarioCRUD(t *testing.T) {
	ss, ws := newSS(t)

	if r := ss.CreateScenario("", ""); r.Error == "" {
		t.Error("name required")
	}
	if r := (&ScenarioService{workspace: newWS(t, "")}).CreateScenario("x", ""); r.Error == "" {
		t.Error("workspace required")
	}
	if r := ss.CreateScenario("x", "../escape"); r.Error == "" {
		t.Error("path traversal must be rejected")
	}

	a := ss.CreateScenario("Login Flow", "")
	if a.Error != "" || filepath.Base(a.Path) != "login-flow.yaml" {
		t.Fatalf("create: %+v", a)
	}
	if msg := ss.ValidateScenarioYAML(ss.LoadScenarioRaw(a.Path)); msg != "" {
		t.Errorf("starter scenario should validate: %s", msg)
	}
	b := ss.CreateScenario("Login Flow", "")
	if filepath.Base(b.Path) != "login-flow-2.yaml" {
		t.Errorf("collision suffix: %+v", b)
	}
	nested := ss.CreateScenario("Deep", "auth/oauth")
	if nested.Error != "" || filepath.Base(filepath.Dir(nested.Path)) != "oauth" {
		t.Errorf("nested create: %+v", nested)
	}
	if r := ss.CreateScenario("???", ""); r.Error != "" || filepath.Base(r.Path) != "scenario.yaml" {
		t.Errorf("unsluggable names fall back: %+v", r)
	}

	detail := ss.LoadScenario(a.Path)
	if detail.Error != "" || detail.Title != "Login Flow" || len(detail.Steps) != 1 {
		t.Fatalf("load: %+v", detail)
	}
	st := detail.Steps[0]
	if st.Method != "GET" || len(st.Assertions) != 1 || st.Assertions[0].Type != "status" || st.Headers == nil || st.Extractions == nil {
		t.Errorf("step mapping: %+v", st)
	}
	if d := ss.LoadScenario(filepath.Join(ws.workspacePath, "none.yaml")); d.Error == "" {
		t.Error("missing file")
	}
	if d := ss.LoadScenario(writeFile(t, filepath.Join(ws.workspacePath, "bad.yaml"), "steps: [")); d.Error == "" {
		t.Error("unparseable file")
	}

	dup := ss.DuplicateScenario(a.Path)
	dup2 := ss.DuplicateScenario(a.Path)
	if filepath.Base(dup.Path) != "login-flow-copy.yaml" || filepath.Base(dup2.Path) != "login-flow-copy-2.yaml" {
		t.Errorf("duplicate naming: %+v %+v", dup, dup2)
	}
	if r := ss.DuplicateScenario(filepath.Join(ws.workspacePath, "none.yaml")); r.Error == "" {
		t.Error("duplicate of missing file")
	}

	renamed := ss.RenameScenario(a.Path, "Sign In")
	if filepath.Base(renamed) != "sign-in.yaml" {
		t.Fatalf("rename: %q", renamed)
	}
	if got := ss.RenameScenario(renamed, "Sign In"); got != renamed {
		t.Errorf("same-name rename is a no-op: %q", got)
	}
	if got := ss.RenameScenario(renamed, "Login Flow 2"); !strings.HasPrefix(got, "error:") {
		t.Errorf("conflict should fail: %q", got)
	}
	if !strings.HasPrefix(ss.RenameScenario(renamed, ""), "error:") {
		t.Error("empty name")
	}
	if !strings.HasPrefix(ss.RenameScenario(filepath.Join(ws.workspacePath, "ghost.yaml"), "x"), "error:") {
		t.Error("renaming a missing file")
	}

	if msg := ss.DeleteScenario(renamed); msg != "" {
		t.Error(msg)
	}
	if ss.DeleteScenario(renamed) == "" {
		t.Error("deleting a missing file reports an error")
	}
}

func TestScenarioRawAndValidation(t *testing.T) {
	ss, ws := newSS(t)
	p := filepath.Join(ws.workspacePath, "scenarios", "r.yaml")
	os.MkdirAll(filepath.Dir(p), 0o755)

	if msg := ss.SaveScenarioRaw(p, "hello"); msg != "" {
		t.Fatal(msg)
	}
	if ss.LoadScenarioRaw(p) != "hello" || ss.LoadScenarioRaw(p+".none") != "" {
		t.Error("raw round trip")
	}
	if ss.SaveScenarioRaw(filepath.Join(ws.workspacePath, "no", "dir", "x"), "x") == "" {
		t.Error("save into missing dir should report")
	}

	if msg := ss.ValidateScenarioYAML("steps: ["); !strings.HasPrefix(msg, "YAML parse error") {
		t.Errorf("parse error: %q", msg)
	}
	if msg := ss.ValidateScenarioYAML("schemaVersion: \"1\"\ntitle: t\nsteps: []\n"); msg == "" {
		t.Error("a scenario without steps should have issues")
	}
}

func TestScenarioFoldersListMoveCopy(t *testing.T) {
	ss, ws := newSS(t)
	if ss.ListScenarios() != nil && len(newSSNoWS(t).ListScenarios()) != 0 {
		t.Error("no workspace → nothing")
	}

	if msg := ss.CreateScenarioFolder("auth/sub"); msg != "" {
		t.Fatal(msg)
	}
	if ss.CreateScenarioFolder("") == "" || ss.CreateScenarioFolder("../x") == "" {
		t.Error("invalid folder names")
	}
	if newSSNoWS(t).CreateScenarioFolder("x") == "" {
		t.Error("no workspace")
	}

	s1 := ss.CreateScenario("One", "")
	s2 := ss.CreateScenario("Two", "auth")
	writeFile(t, filepath.Join(ws.workspacePath, "scenarios", "notes.txt"), "ignored")
	writeFile(t, filepath.Join(ws.workspacePath, "scenarios", "broken.yaml"), "steps: [")

	var files, folders int
	byTitle := map[string]ScenarioInfoIO{}
	for _, it := range ss.ListScenarios() {
		if it.IsFolder {
			folders++
			continue
		}
		files++
		byTitle[it.Title] = it
	}
	if files != 3 || folders != 2 {
		t.Errorf("files=%d folders=%d", files, folders)
	}
	if byTitle["one"].StepCount != 1 || byTitle["one"].Folder != "" || byTitle["two"].Folder != "auth" || byTitle["broken"].StepCount != 0 {
		t.Errorf("list metadata: %+v", byTitle)
	}

	moved, err := ss.MoveScenario(s1.Path, "auth/sub")
	if err != nil || filepath.Base(filepath.Dir(moved)) != "sub" {
		t.Fatalf("move: %q %v", moved, err)
	}
	if same, err := ss.MoveScenario(moved, "auth/sub"); err != nil || same != moved {
		t.Errorf("move in place: %q %v", same, err)
	}
	copied, err := ss.CopyScenario(s2.Path, "auth/sub")
	if err != nil {
		t.Fatal(err)
	}
	again, _ := ss.CopyScenario(s2.Path, "auth/sub")
	third, _ := ss.CopyScenario(s2.Path, "auth/sub")
	if filepath.Base(copied) != "two.yaml" || filepath.Base(again) != "two-copy.yaml" || filepath.Base(third) != "two-copy-3.yaml" {
		t.Errorf("copy naming: %s %s %s", copied, again, third)
	}
	if _, err := ss.MoveScenario(s2.Path, "auth/sub"); err == nil {
		t.Error("move onto existing file should fail")
	}
	outside := writeFile(t, filepath.Join(t.TempDir(), "o.yaml"), "x")
	for name, fn := range map[string]func() error{
		"move outside":  func() error { _, e := ss.MoveScenario(outside, ""); return e },
		"copy outside":  func() error { _, e := ss.CopyScenario(outside, ""); return e },
		"move bad dest": func() error { _, e := ss.MoveScenario(s2.Path, "../.."); return e },
		"copy bad dest": func() error { _, e := ss.CopyScenario(s2.Path, "../.."); return e },
		"move no ws":    func() error { _, e := newSSNoWS(t).MoveScenario(s2.Path, ""); return e },
		"copy no ws":    func() error { _, e := newSSNoWS(t).CopyScenario(s2.Path, ""); return e },
		"copy missing": func() error {
			_, e := ss.CopyScenario(filepath.Join(ws.workspacePath, "scenarios", "ghost.yaml"), "")
			return e
		},
	} {
		if fn() == nil {
			t.Errorf("%s should fail", name)
		}
	}

	if got := ss.RenameScenarioFolder("auth", "Authentication"); got != "authentication" {
		t.Errorf("rename folder: %q", got)
	}
	if got := ss.RenameScenarioFolder("authentication", "authentication"); got != "authentication" {
		t.Errorf("same name: %q", got)
	}
	ss.CreateScenarioFolder("other")
	for name, got := range map[string]string{
		"conflict": ss.RenameScenarioFolder("other", "authentication"),
		"empty":    ss.RenameScenarioFolder("other", ""),
		"bad path": ss.RenameScenarioFolder("..", "x"),
		"dot":      ss.RenameScenarioFolder(".", "x"),
		"bad name": ss.RenameScenarioFolder("other", "!!!"),
		"missing":  ss.RenameScenarioFolder("ghost", "x"),
		"no ws":    newSSNoWS(t).RenameScenarioFolder("a", "b"),
	} {
		if !strings.HasPrefix(got, "error:") {
			t.Errorf("%s should fail, got %q", name, got)
		}
	}

	if msg := ss.DeleteScenarioFolder("authentication"); msg != "" {
		t.Error(msg)
	}
	if _, err := os.Stat(filepath.Join(ws.workspacePath, "scenarios", "authentication")); !os.IsNotExist(err) {
		t.Error("folder and contents should be removed")
	}
	if ss.DeleteScenarioFolder("..") == "" || ss.DeleteScenarioFolder(".") == "" || newSSNoWS(t).DeleteScenarioFolder("x") == "" {
		t.Error("guards")
	}
}

func newSSNoWS(t *testing.T) *ScenarioService {
	t.Helper()
	return &ScenarioService{workspace: newWS(t, "")}
}

func TestRunScenario_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Seen", r.URL.Path)
		fmt.Fprint(w, "pong")
	}))
	defer srv.Close()

	ss, ws := newSS(t)
	dir := filepath.Join(ws.workspacePath, "scenarios")
	path := writeFile(t, filepath.Join(dir, "ping.yaml"), pingScenario(srv.URL, ""))

	res := ss.RunScenario(path, "", 0)
	if res.Error != "" || !res.Passed || len(res.Steps) != 1 {
		t.Fatalf("run: %+v", res)
	}
	st := res.Steps[0]
	if st.StatusCode != 200 || st.ResponseBody != "pong" || st.Extracted["code"] != "200" || len(st.Assertions) != 1 || len(st.ResponseHeaders) == 0 {
		t.Errorf("step: %+v", st)
	}
	if res.Coverage != nil || res.SpecError != "" {
		t.Error("no spec bound → no coverage")
	}

	hist := ss.GetScenarioRunHistory(0)
	if len(hist) != 1 || hist[0].Title != "Ping flow" || !hist[0].Passed || hist[0].StepTotal != 1 || hist[0].StepPassed != 1 || len(hist[0].Steps) != 1 {
		t.Errorf("run history: %+v", hist)
	}

	skipped := ss.RunScenario(path, "", 1)
	if !skipped.Passed || !skipped.Steps[0].Skipped {
		t.Errorf("fromStep: %+v", skipped.Steps)
	}
	if h := ss.GetScenarioRunHistory(1); len(h) != 1 || h[0].StepSkipped != 1 {
		t.Errorf("limit/skipped counts: %+v", h)
	}

	if sav := ss.SaveScenarioResult(path, res); sav != "" {
		t.Fatal(sav)
	}
	if got := ss.LoadScenarioResult(path); got == nil || got.Title != "Ping flow" || len(got.Steps) != 1 {
		t.Errorf("cached result: %+v", got)
	}
	if ss.LoadScenarioResult(filepath.Join(dir, "never.yaml")) != nil {
		t.Error("no cached result")
	}
	if ss.SaveScenarioResult(filepath.Join(t.TempDir(), "x.yaml"), res) == "" {
		t.Error("outside workspace should be refused")
	}
	if newSSNoWS(t).SaveScenarioResult(path, res) == "" || newSSNoWS(t).LoadScenarioResult(path) != nil {
		t.Error("no workspace")
	}
}

func TestRunScenario_Errors(t *testing.T) {
	ss, ws := newSS(t)
	dir := filepath.Join(ws.workspacePath, "scenarios")

	if r := ss.RunScenario(filepath.Join(dir, "none.yaml"), "", 0); !strings.Contains(r.Error, "read file") {
		t.Errorf("missing: %+v", r)
	}
	bad := writeFile(t, filepath.Join(dir, "bad.yaml"), "steps: [")
	if r := ss.RunScenario(bad, "", 0); !strings.Contains(r.Error, "parse") {
		t.Errorf("bad: %+v", r)
	}
	ok := writeFile(t, filepath.Join(dir, "ok.yaml"), pingScenario("http://127.0.0.1:1", ""))
	if r := ss.RunScenario(ok, filepath.Join(dir, "noenv.yaml"), 0); !strings.Contains(r.Error, "read env file") {
		t.Errorf("env: %+v", r)
	}
	if r := ss.RunScenario(ok, "", 0); r.Passed || r.Steps[0].Error == "" {
		t.Errorf("unreachable server should fail the run: %+v", r)
	}
	ss.CancelScenario() // no run in flight: no-op
}

func TestRunScenario_RequestRefAndEnvOverride(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Env")
	}))
	defer srv.Close()

	ss, ws := newSS(t)
	dir := filepath.Join(ws.workspacePath, "scenarios")
	writeFile(t, filepath.Join(ws.workspacePath, "requests", "shared.yaml"), fmt.Sprintf(
		"schemaVersion: \"1\"\ntitle: shared\nmethod: GET\nurl: %s/x\nheaders:\n  - name: X-Env\n    value: \"{{who}}\"\n", srv.URL))
	envPath := writeFile(t, filepath.Join(ws.workspacePath, "environments", "alt.yaml"),
		"schemaVersion: \"1\"\nname: alt\nvariables:\n  - name: who\n    value: alt-env\n")
	path := writeFile(t, filepath.Join(dir, "ref.yaml"), `schemaVersion: "1"
title: ref
steps:
  - label: via ref
    requestRef: shared.yaml
    assertions:
      - type: status
        operator: eq
        expected: 200
  - label: dangling ref
    requestRef: nope.yaml
    failPolicy: continue
`)
	res := ss.RunScenario(path, envPath, 0)
	if gotHeader != "alt-env" {
		t.Errorf("env override not applied: %q", gotHeader)
	}
	if !res.Steps[0].Passed || res.Steps[1].Error == "" || res.Passed {
		t.Errorf("ref resolution: %+v", res.Steps)
	}
}

func TestRunScenario_SpecCoverageAccumulates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	ss, ws := newSS(t)
	dir := filepath.Join(ws.workspacePath, "scenarios")
	writeFile(t, filepath.Join(dir, "spec.yaml"), pingSpec)
	path := writeFile(t, filepath.Join(dir, "cov.yaml"), strings.Replace(pingScenario(srv.URL, ""), "title: Ping flow", "title: Ping flow\nspecFile: spec.yaml", 1))

	first := ss.RunScenario(path, "", 0)
	if first.Error != "" || first.SpecError != "" || first.Coverage == nil {
		t.Fatalf("run: %+v", first)
	}
	if first.Coverage.Covered != 1 || first.Coverage.Total != 2 || first.Coverage.Exercised[0] != "GET /ping" || first.Coverage.Missed[0] != "GET /other" {
		t.Errorf("coverage: %+v", first.Coverage)
	}
	if first.AccumulatedCoverage == nil || first.AccumulatedCoverage.TotalRuns != 1 {
		t.Fatalf("accumulated: %+v", first.AccumulatedCoverage)
	}
	ss.RunScenario(path, "", 0)

	list := ss.ListAccumulatedCoverage()
	if len(list) != 1 || list[0].TotalRuns != 2 {
		t.Fatalf("list: %+v", list)
	}
	if err := ss.ResetAccumulatedCoverage("spec.yaml"); err != nil {
		t.Fatal(err)
	}
	if got := ss.ListAccumulatedCoverage(); len(got) != 0 {
		t.Errorf("reset should clear: %+v", got)
	}
	if a, err := ss.GetAccumulatedCoverage("spec.yaml"); a != nil || err != nil {
		t.Errorf("no data → nil: %+v %v", a, err)
	}
	if a, err := ss.GetAccumulatedCoverage(""); a != nil || err != nil {
		t.Error("empty spec")
	}
	if ss.ResetAccumulatedCoverage("") != nil || newSSNoWS(t).ResetAccumulatedCoverage("x") != nil || newSSNoWS(t).ListAccumulatedCoverage() != nil {
		t.Error("no-op guards")
	}

	badSpec := writeFile(t, filepath.Join(dir, "bad.yaml"), strings.Replace(pingScenario(srv.URL, ""), "title: Ping flow", "title: Bad\nspecFile: missing-spec.yaml", 1))
	if r := ss.RunScenario(badSpec, "", 0); r.SpecError == "" || r.Error != "" || r.Coverage != nil {
		t.Errorf("a missing spec is reported but does not block the run: %+v", r)
	}
	writeFile(t, filepath.Join(dir, "junk-spec.yaml"), "openapi: [")
	junk := writeFile(t, filepath.Join(dir, "junk.yaml"), strings.Replace(pingScenario(srv.URL, ""), "title: Ping flow", "title: Junk\nspecFile: junk-spec.yaml", 1))
	if r := ss.RunScenario(junk, "", 0); r.SpecError == "" {
		t.Error("an unparsable spec is reported")
	}
}

func TestAccumulatedCoverage_AbsoluteSpec(t *testing.T) {
	ss, ws := newSS(t)
	spec := writeFile(t, filepath.Join(ws.workspacePath, "specs", "api.yaml"), pingSpec)
	if err := saveStoredCoverage(ws.workspacePath, coverage.StoredCoverage{SpecFile: spec, TotalRuns: 1, TotalOps: 2, Exercised: []string{"GET /ping"}}); err != nil {
		t.Fatal(err)
	}
	acc, err := ss.GetAccumulatedCoverage(spec)
	if err != nil || acc == nil {
		t.Fatalf("%v %v", acc, err)
	}
	if acc.Covered != 1 || acc.Total != 2 || len(acc.Missed) != 1 {
		t.Errorf("accumulated view: %+v", acc)
	}

	os.Remove(spec)
	if _, err := ss.GetAccumulatedCoverage(spec); err == nil {
		t.Error("missing spec file should surface an error")
	}
	writeFile(t, spec, "openapi: [")
	if _, err := ss.GetAccumulatedCoverage(spec); err == nil {
		t.Error("unparsable spec should surface an error")
	}
}

func TestRunFolder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()

	ss, ws := newSS(t)
	dir := filepath.Join(ws.workspacePath, "requests", "batch")
	req := func(name, title, path string) {
		writeFile(t, filepath.Join(dir, name), fmt.Sprintf("schemaVersion: \"1\"\ntitle: %s\nmethod: GET\nurl: %s%s\n", title, srv.URL, path))
	}
	req("a.yaml", "Good", "/ok")
	req("b.yaml", "", "/fail")
	writeFile(t, filepath.Join(dir, "c.yaml"), "method: [")
	writeFile(t, filepath.Join(dir, "d.yaml"), "schemaVersion: \"1\"\nmethod: GET\nurl: http://127.0.0.1:1\n")
	writeFile(t, filepath.Join(dir, "skip.txt"), "x")
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)

	got := ss.RunFolder(dir)
	if len(got) != 4 {
		t.Fatalf("expected 4 results (txt and dirs skipped): %+v", got)
	}
	if got[0].Name != "Good" || !got[0].Passed || got[0].StatusCode != 200 || got[0].Method != "GET" {
		t.Errorf("good: %+v", got[0])
	}
	if got[1].Name != "b" || got[1].Passed || got[1].StatusCode != 500 {
		t.Errorf("failing status: %+v", got[1])
	}
	if !strings.HasPrefix(got[2].Error, "parse:") {
		t.Errorf("parse error: %+v", got[2])
	}
	if got[3].Error == "" || got[3].Passed {
		t.Errorf("network error: %+v", got[3])
	}
	if r := ss.RunFolder(filepath.Join(dir, "missing")); len(r) != 1 || r[0].Error == "" {
		t.Errorf("missing folder: %+v", r)
	}
}

func TestLoadSpecBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gone" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, pingSpec)
	}))
	defer srv.Close()

	if _, f, err := loadSpecBytes(srv.URL+"/spec.json", ""); err != nil || f != "json" {
		t.Errorf("json url: %q %v", f, err)
	}
	if _, f, err := loadSpecBytes(srv.URL+"/spec", ""); err != nil || f != "yaml" {
		t.Errorf("extensionless url defaults to yaml: %q %v", f, err)
	}
	if _, _, err := loadSpecBytes(srv.URL+"/gone", ""); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("http error: %v", err)
	}
	if _, _, err := loadSpecBytes("http://127.0.0.1:1/x", ""); err == nil {
		t.Error("unreachable url")
	}

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "s.yml"), pingSpec)
	writeFile(t, filepath.Join(dir, "noext"), pingSpec)
	if _, f, err := loadSpecBytes("s.yml", dir); err != nil || f != "yml" {
		t.Errorf("relative file: %q %v", f, err)
	}
	if _, f, err := loadSpecBytes(filepath.Join(dir, "noext"), ""); err != nil || f != "yaml" {
		t.Errorf("absolute, no ext: %q %v", f, err)
	}
	if _, _, err := loadSpecBytes("missing.yaml", dir); err == nil {
		t.Error("missing file")
	}
}

func TestTemplates(t *testing.T) {
	ss, _ := newSS(t)
	list := ss.ListTemplates()
	if len(list) == 0 {
		t.Fatal("no templates bundled")
	}
	cats := map[string]int{}
	for _, tpl := range list {
		if tpl.ID == "" || tpl.Title == "" || tpl.StepCount == 0 || tpl.Category == "" {
			t.Errorf("incomplete template: %+v", tpl)
		}
		if len(tpl.Description) > 163 {
			t.Errorf("description not truncated: %d", len(tpl.Description))
		}
		cats[tpl.Category]++
	}
	if cats["Conformance"] == 0 {
		t.Errorf("categories: %v", cats)
	}

	for _, tpl := range list {
		r := ss.CreateFromTemplate(tpl.ID, "")
		if r.Error != "" {
			t.Errorf("%s: %s", tpl.ID, r.Error)
			continue
		}
		if msg := ss.ValidateScenarioYAML(ss.LoadScenarioRaw(r.Path)); msg != "" {
			t.Errorf("template %s does not validate once copied: %s", tpl.ID, msg)
		}
	}

	id := list[0].ID
	named := ss.CreateFromTemplate(id, "  My Custom Run  ")
	again := ss.CreateFromTemplate(id, "My Custom Run")
	if filepath.Base(named.Path) != "my-custom-run.yaml" || filepath.Base(again.Path) != "my-custom-run-2.yaml" {
		t.Errorf("naming: %+v %+v", named, again)
	}
	if d := ss.LoadScenario(named.Path); d.Title != "My Custom Run" {
		t.Errorf("title: %q", d.Title)
	}
	if ss.CreateFromTemplate("", "x").Error == "" || ss.CreateFromTemplate("nope", "x").Error == "" || newSSNoWS(t).CreateFromTemplate(id, "x").Error == "" {
		t.Error("error cases")
	}

	for id, wantCat := range map[string]string{"jsonplaceholder-crud": "Public APIs", "github-public-api": "Public APIs", "user-lifecycle": "Conformance", "zzz": "Other"} {
		if c, _, _ := templateMeta(id); c != wantCat {
			t.Errorf("templateMeta(%s) = %s", id, c)
		}
	}
}

func TestCreateScenario_CollisionStaysInFolder(t *testing.T) {
	ss, ws := newSS(t)
	a := ss.CreateScenario("Dup", "auth")
	b := ss.CreateScenario("Dup", "auth")
	if filepath.Dir(a.Path) != filepath.Dir(b.Path) || filepath.Base(b.Path) != "dup-2.yaml" {
		t.Errorf("collision suffix must stay in the target folder: %s vs %s", a.Path, b.Path)
	}
	if _, err := os.Stat(filepath.Join(ws.workspacePath, "scenarios", "dup-2.yaml")); err == nil {
		t.Error("stray copy written to the scenarios root")
	}
}

func TestCreateScenario_SpecialCharactersInTitle(t *testing.T) {
	ss, _ := newSS(t)
	for _, name := range []string{"Risk: colon # hash", `Quote "inside"`, "- leading dash", "multi" + string(rune(10)) + "line"} {
		r := ss.CreateScenario(name, "")
		if r.Error != "" {
			t.Fatalf("%q: %s", name, r.Error)
		}
		d := ss.LoadScenario(r.Path)
		if d.Error != "" || d.Title != name {
			t.Errorf("%q round-trip: title=%q err=%q", name, d.Title, d.Error)
		}
	}
}
