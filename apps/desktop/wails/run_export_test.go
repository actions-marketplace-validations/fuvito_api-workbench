package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const exportSecret = "s3cr3t-value-1234"

func exportServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"echo":%q,"access_token":"jwt-abcdef-0001"}`, exportSecret)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func exportScenario(title, url, expected string) string {
	return fmt.Sprintf(`schemaVersion: "1"
title: %q
steps:
  - label: call
    request: {method: GET, url: %q}
    extractions:
      - {name: access_token, source: jsonpath, path: $.access_token}
    assertions:
      - {type: jsonpath, path: $.echo, operator: eq, expected: %s}
`, title, url, expected)
}

func TestExportRunReport(t *testing.T) {
	srv := exportServer(t)
	ss, ws := newSS(t)
	a := writeFile(t, filepath.Join(ws.workspacePath, "scenarios", "a.yaml"), exportScenario("Alpha", srv.URL, exportSecret))
	b := writeFile(t, filepath.Join(ws.workspacePath, "scenarios", "b.yaml"), exportScenario("Beta", srv.URL, "something-else"))
	if res := ss.RunScenario(a, "", 0); res.Error != "" {
		t.Fatal(res.Error)
	}
	ss.RunScenario(b, "", 0)

	paths := []string{a, b}
	check := func(format string, wants ...string) string {
		t.Helper()
		out := ss.ExportRunReport(format, paths)
		if out.Error != "" {
			t.Fatalf("%s: %s", format, out.Error)
		}
		for _, w := range wants {
			if !strings.Contains(out.Content, w) {
				t.Errorf("%s report missing %q:\n%s", format, w, out.Content)
			}
		}
		return out.Content
	}
	check("html", "<!DOCTYPE html>", "Alpha", "Beta", "1/2 scenarios passed")
	check("junit", "<testsuite ", "Alpha", "Beta")
	check("json", `"run": "Run results"`, `"passed": false`)
	check("markdown", "# Run results", "❌ **FAILED**", "Alpha", "Beta")
	check("text", "Run: Run results", "(2 scenarios)", "Run FAILED  1/2 scenarios passed")
}

func TestExportRunReportKeepsSecretsOut(t *testing.T) {
	srv := exportServer(t)
	ss, ws := newSS(t)
	env := writeFile(t, filepath.Join(ws.workspacePath, "environments", "ci.yaml"),
		"schemaVersion: \"1\"\nname: ci\nvariables:\n  - name: apiKey\n    value: "+exportSecret+"\n    secret: true\n")
	sc := writeFile(t, filepath.Join(ws.workspacePath, "scenarios", "s.yaml"), exportScenario("Leaky", srv.URL, "other"))
	ss.RunScenario(sc, env, 0)

	for _, format := range []string{"html", "junit", "json", "markdown", "text"} {
		out := ss.ExportRunReport(format, []string{sc})
		if out.Error != "" {
			t.Fatalf("%s: %s", format, out.Error)
		}
		if strings.Contains(out.Content, exportSecret) || strings.Contains(out.Content, "jwt-abcdef-0001") {
			t.Errorf("%s report leaked a secret:\n%s", format, out.Content)
		}
	}
}

func TestExportRunReportErrors(t *testing.T) {
	ss, ws := newSS(t)
	if out := ss.ExportRunReport("html", []string{"nothing-was-run.yaml"}); !strings.Contains(out.Error, "no results to export") {
		t.Errorf("nothing run: %+v", out)
	}
	srv := exportServer(t)
	sc := writeFile(t, filepath.Join(ws.workspacePath, "scenarios", "a.yaml"), exportScenario("Alpha", srv.URL, exportSecret))
	ss.RunScenario(sc, "", 0)
	if out := ss.ExportRunReport("pdf", []string{sc}); !strings.Contains(out.Error, "unknown report format") {
		t.Errorf("unknown format: %+v", out)
	}
	// a scenario that was not run is simply left out
	out := ss.ExportRunReport("text", []string{sc, "never-run.yaml"})
	if out.Error != "" || !strings.Contains(out.Content, "(1 scenario)") {
		t.Errorf("unrun scenarios are skipped: %+v", out)
	}
}
