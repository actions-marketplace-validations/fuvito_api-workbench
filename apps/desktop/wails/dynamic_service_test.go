package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunScenario_ReportsSeedAndScenarioVariables(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()

	ws := t.TempDir()
	scn := writeFile(t, filepath.Join(ws, "scenarios", "dyn.yaml"), `schemaVersion: "1"
title: Dyn
variables:
  - name: email
    value: "u-{{$randomString 6}}@example.com"
  - name: token
    value: "tok-{{$randomString 8}}"
    secret: true
steps:
  - label: ping
    request: {method: GET, url: "`+srv.URL+`/x?e={{email}}"}
`)
	res := NewScenarioService(newWS(t, ws), NewSettingsService()).RunScenario(scn, "", 0)
	if res.Error != "" || !res.Passed {
		t.Fatalf("result = %+v", res)
	}
	if res.Seed == nil {
		t.Error("random values were used, so the seed is reported")
	}
	if len(res.Variables) != 2 || !strings.HasPrefix(res.Variables[0].Value, "u-") {
		t.Errorf("variables = %+v", res.Variables)
	}
	if res.Variables[1].Value != "***" || !res.Variables[1].Secret {
		t.Errorf("a secret variable must be masked: %+v", res.Variables[1])
	}
}

func TestRunScenario_SetupErrorIsReported(t *testing.T) {
	isolateHome(t)
	ws := t.TempDir()
	scn := writeFile(t, filepath.Join(ws, "scenarios", "bad.yaml"), `schemaVersion: "1"
title: Bad
variables:
  - name: x
    value: "{{$nope}}"
steps:
  - label: a
    request: {method: GET, url: "http://127.0.0.1:1/x"}
`)
	res := NewScenarioService(newWS(t, ws), NewSettingsService()).RunScenario(scn, "", 0)
	if res.Passed || !strings.Contains(res.Error, "unknown function $nope") {
		t.Errorf("result = %+v", res)
	}
}

func TestListDynamicFunctions(t *testing.T) {
	fns := NewScenarioService(newWS(t, t.TempDir()), NewSettingsService()).ListDynamicFunctions()
	found := false
	for _, f := range fns {
		if f.Name == "calc" && f.Example != "" && f.Description != "" {
			found = true
		}
	}
	if !found || len(fns) < 20 {
		t.Errorf("functions = %+v", fns)
	}
}
