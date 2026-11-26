package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const writtenPw = "s3cretPw1"

// What the services hand to the screen, and what they keep, apart from the response itself and
// the extracted values: those are shown as they are.
func TestDesktopResultsDoNotRepeatAWrittenOutPassword(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME"} {
		t.Setenv(name, home)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"database":"postgres://app:` + writtenPw + `@db:5432/x"}`))
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	ws := newWS(t, dir)
	closed := "http://127.0.0.1:1/x?password=" + writtenPw
	scn := writeFile(t, filepath.Join(dir, "scenarios", "p.yaml"), `schemaVersion: "1"
title: passwords
variables:
  - {name: database, value: "redis://:`+writtenPw+`@cache:6379"}
steps:
  - label: answered
    failPolicy: continue
    request: {method: GET, url: "`+srv.URL+`", headers: [{name: X-Db, value: "{{database}}"}]}
    assertions:
      - {type: jsonpath, path: "$.database", operator: eq, expected: "x"}
  - label: no answer
    request: {method: GET, url: "`+closed+`"}
`)
	svc := NewScenarioService(ws, NewSettingsService())
	res := svc.RunScenario(scn, "", 0)
	if res.Error != "" || len(res.Steps) != 2 || res.Steps[1].Error == "" {
		t.Fatalf("%+v", res)
	}
	for i := range res.Steps {
		res.Steps[i].ResponseBody = ""
	}
	check := func(name string, v any) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), writtenPw) {
			t.Errorf("%s holds the password: %s", name, b)
		}
	}
	check("the scenario result", res)
	history := svc.GetScenarioRunHistory(0)
	if len(history) == 0 {
		t.Fatal("the run was not recorded")
	}
	check("the run history", history)

	spec := writeFile(t, filepath.Join(dir, "spec.yaml"), `openapi: "3.0.3"
info: {title: P, version: "1"}
paths:
  /thing:
    get: {responses: {"200": {description: ok}}}
`)
	out := NewConformanceService(ws, NewSettingsService()).RunConformance(ConformRunInput{SpecPath: spec, BaseURL: "http://127.0.0.1:1/x?password=" + writtenPw})
	if out.Error != "" || len(out.Operations) != 1 || out.Operations[0].Error == "" {
		t.Fatalf("%+v", out)
	}
	// the base address is the page's own input, echoed back
	check("the conformance operations", out.Operations)
}
