package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

// scenarioExpecting returns a one-step scenario that GETs url and expects status.
func scenarioExpecting(title, url string, status int) string {
	return fmt.Sprintf(`schemaVersion: "1"
title: %q
steps:
  - label: "step of %s"
    request:
      method: GET
      url: %q
    assertions:
      - type: status
        operator: eq
        expected: %d
`, title, title, url, status)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	return string(data)
}

// countingServer answers 200 to everything and counts the requests it receives.
func countingServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

// suite builds a folder with two passing scenarios (one nested) and a non-scenario file.
func suite(t *testing.T, url string) string {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a-first.yaml":      scenarioExpecting("First", url+"/ping", 200),
		"sub/b-second.yaml": scenarioExpecting("Second", url+"/ping", 200),
		"env.yaml":          "schemaVersion: \"1\"\nname: ci\nvariables: []\n",
	})
	return root
}

func TestRunFolder_AllPassWithCombinedReport(t *testing.T) {
	srv, _ := countingServer(t)
	root := suite(t, srv.URL)

	out, err := exec(t, "run", root)
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	for _, want := range []string{
		"(2 scenarios)", "Scenario: First", "Scenario: Second",
		"Run PASSED  2/2 scenarios passed", "skipped 1 file that is not scenarios",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Index(out, "Scenario: First") > strings.Index(out, "Scenario: Second") {
		t.Error("scenarios run in path order")
	}
}

func TestRunFolder_FailureExitsWithOne(t *testing.T) {
	srv, _ := countingServer(t)
	root := suite(t, srv.URL)
	writeFiles(t, root, map[string]string{"c-broken-expectation.yaml": scenarioExpecting("Third", srv.URL+"/ping", 404)})

	out, err := exec(t, "run", root)
	if exitCode(err) != 1 {
		t.Fatalf("a failing scenario exits 1, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Run FAILED  2/3 scenarios passed") {
		t.Errorf("summary:\n%s", out)
	}
}

func TestRunFolder_FailFastStopsAfterFirstFailure(t *testing.T) {
	srv, n := countingServer(t)
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.yaml": scenarioExpecting("A fails", srv.URL+"/ping", 404),
		"b.yaml": scenarioExpecting("B", srv.URL+"/ping", 200),
		"c.yaml": scenarioExpecting("C", srv.URL+"/ping", 200),
	})
	out, err := exec(t, "run", "--fail-fast", root)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %v\n%s", err, out)
	}
	if !strings.Contains(out, "stopped after the first failure (--fail-fast); 2 scenarios not run") {
		t.Errorf("missing fail-fast note:\n%s", out)
	}
	if got := n.Load(); got != 1 {
		t.Errorf("only the first scenario should send a request, got %d", got)
	}
	if !strings.Contains(out, "0/1 scenarios passed") {
		t.Errorf("only the scenario that ran is counted:\n%s", out)
	}
}

func TestRunFolder_NoRecursive(t *testing.T) {
	srv, _ := countingServer(t)
	root := suite(t, srv.URL)
	out, err := exec(t, "run", "--no-recursive", root)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "(1 scenario)") || strings.Contains(out, "Scenario: Second") {
		t.Errorf("only the top level should run:\n%s", out)
	}
}

func TestRunFolder_BrokenScenarioStopsEverythingBeforeAnyRequest(t *testing.T) {
	srv, n := countingServer(t)
	root := suite(t, srv.URL)
	writeFiles(t, root, map[string]string{"z-bad.yaml": "title: [unclosed\nsteps:\n  - label: x\n"})

	out, err := exec(t, "run", root)
	if err == nil || exitCode(err) == 1 {
		t.Fatalf("an unreadable scenario is a configuration error (exit 2), got %v", err)
	}
	if !strings.Contains(err.Error(), "could not be loaded, so nothing was run") || !strings.Contains(err.Error(), "z-bad.yaml") {
		t.Errorf("error should name the file and say nothing ran: %v\n%s", err, out)
	}
	if got := n.Load(); got != 0 {
		t.Errorf("no request may be sent when a scenario is broken, got %d", got)
	}
}

func TestRunFolder_NothingFoundIsAnError(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"env.yaml": "name: ci\n"})
	_, err := exec(t, "run", root)
	if err == nil || !strings.Contains(err.Error(), "no scenarios found") {
		t.Errorf("want 'no scenarios found', got %v", err)
	}
	_, err = exec(t, "run", filepath.Join(root, "missing"))
	if err == nil {
		t.Error("a missing path is an error")
	}
}

func TestRunFolder_MixedFilesAndFoldersAreDeduplicated(t *testing.T) {
	srv, n := countingServer(t)
	root := suite(t, srv.URL)
	out, err := exec(t, "run", root, filepath.Join(root, "a-first.yaml"), filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "(2 scenarios)") || n.Load() != 2 {
		t.Errorf("each scenario runs once (requests=%d):\n%s", n.Load(), out)
	}
}

func TestRunFolder_ReportFormats(t *testing.T) {
	srv, _ := countingServer(t)
	root := suite(t, srv.URL)

	t.Run("json", func(t *testing.T) {
		out, err := exec(t, "run", "--format", "json", root)
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		var doc struct {
			Run       string            `json:"run"`
			Passed    bool              `json:"passed"`
			Scenarios []json.RawMessage `json:"scenarios"`
		}
		if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil {
			t.Fatalf("not JSON: %v\n%s", err, out)
		}
		if !doc.Passed || len(doc.Scenarios) != 2 || doc.Run != filepath.Base(root) {
			t.Errorf("doc = %+v", doc)
		}
	})

	t.Run("junit", func(t *testing.T) {
		out, err := exec(t, "run", "--format", "junit", root)
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		var ju struct {
			Tests  int `xml:"tests,attr"`
			Suites []struct {
				Name string `xml:"name,attr"`
			} `xml:"testsuite"`
		}
		if err := xml.Unmarshal([]byte(out[strings.Index(out, "<?xml"):]), &ju); err != nil {
			t.Fatalf("not XML: %v\n%s", err, out)
		}
		if len(ju.Suites) != 2 || ju.Tests != 2 {
			t.Errorf("one test suite per scenario: %+v", ju)
		}
	})

	t.Run("html to a file", func(t *testing.T) {
		file := filepath.Join(root, "reports", "all.html")
		if _, err := exec(t, "run", "--format", "html", "--output", file, root); err != nil {
			t.Fatal(err)
		}
		data := readFile(t, file)
		for _, want := range []string{"<!DOCTYPE html>", "First", "Second", "2/2 scenarios passed"} {
			if !strings.Contains(data, want) {
				t.Errorf("html missing %q", want)
			}
		}
	})
}

func TestRunFolder_CoverageIsAggregatedOnceAndAccumulatesPerScenario(t *testing.T) {
	srv, _ := countingServer(t)
	root := suite(t, srv.URL)
	specPath := filepath.Join(root, "openapi.yaml")
	writeFiles(t, root, map[string]string{"openapi.yaml": minimalSpec})
	ws := t.TempDir()

	out, err := exec(t, "run", "--spec", specPath, "--coverage", "--workspace", ws, filepath.Join(root, "a-first.yaml"), filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Count(out, "Spec coverage") != 1 || strings.Contains(out, "Spec coverage (") {
		t.Errorf("one spec gives one plain coverage section:\n%s", out)
	}
	stored := coverage.Load(ws, specPath)
	if stored.TotalRuns != 2 || stored.PassedRuns != 2 {
		t.Errorf("each scenario counts as a run: %+v", stored)
	}
}

func TestRun_SingleFileOutputIsUnchanged(t *testing.T) {
	srv, _ := countingServer(t)
	sc := writeTempFile(t, "s.yaml", scenarioExpecting("Solo", srv.URL+"/ping", 200))
	out, err := exec(t, "run", sc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Run:") || strings.Contains(out, "scenarios passed") || !strings.HasPrefix(out, "Scenario: Solo") {
		t.Errorf("a single file keeps the single-scenario report:\n%s", out)
	}
}
