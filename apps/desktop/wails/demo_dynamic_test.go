package main

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The dynamic-value demo scenarios are for experimenting, so they must pass on every run: they
// are run several times because every run generates different values.
func TestDemoWorkspace_DynamicValueScenariosPass(t *testing.T) {
	isolateHome(t)
	resetDemoState()
	srv := httptest.NewServer(newDemoMux())
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())

	dir := t.TempDir()
	if err := extractDemoWorkspace(dir, port); err != nil {
		t.Fatal(err)
	}
	wsSvc := newWS(t, dir)
	if err := wsSvc.SelectEnvironment(filepath.Join(dir, "environments", "local.yaml")); err != nil {
		t.Fatal(err)
	}
	ss := NewScenarioService(wsSvc, NewSettingsService())

	files, _ := filepath.Glob(filepath.Join(dir, "scenarios", "4-dynamic-values", "*.yaml"))
	if len(files) != 4 {
		t.Fatalf("expected 4 dynamic-value scenarios, found %d", len(files))
	}
	for _, f := range files {
		for run := 1; run <= 5; run++ {
			res := ss.RunScenario(f, "", 0)
			if res.Error != "" || !res.Passed {
				t.Errorf("%s run %d: error=%q passed=%v", filepath.Base(f), run, res.Error, res.Passed)
				for _, st := range res.Steps {
					for _, a := range st.Assertions {
						if !a.Passed {
							t.Logf("  %s: %s", st.Label, a.Message)
						}
					}
					if st.Error != "" {
						t.Logf("  %s: %s", st.Label, st.Error)
					}
				}
			}
		}
	}
}

// A single request (not a scenario) also generates dynamic values on every send.
func TestDemoWorkspace_DynamicRequestGeneratesFreshValues(t *testing.T) {
	isolateHome(t)
	resetDemoState()
	srv := httptest.NewServer(newDemoMux())
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	dir := t.TempDir()
	if err := extractDemoWorkspace(dir, port); err != nil {
		t.Fatal(err)
	}
	wsSvc := newWS(t, dir)
	if err := wsSvc.SelectEnvironment(filepath.Join(dir, "environments", "local.yaml")); err != nil {
		t.Fatal(err)
	}
	rs := NewRequestService(wsSvc, NewSettingsService())
	defer rs.close() // releases the history database so the temp folder can be removed
	file, err := wsSvc.LoadRequest(filepath.Join(dir, "requests", "8-dynamic-values", "echo-dynamic.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	send := func() ResponseOutput {
		return rs.Execute(RequestInput{Method: file.Method, URL: file.URL, Headers: file.Headers, QueryParams: file.QueryParams, Body: file.Body})
	}
	a, b := send(), send()
	if a.Error != "" || a.StatusCode != 200 || strings.Contains(a.Body, "{{$") {
		t.Fatalf("every function must be replaced before sending: %+v", a)
	}
	if a.Body == b.Body {
		t.Error("two sends must generate different values")
	}
}
