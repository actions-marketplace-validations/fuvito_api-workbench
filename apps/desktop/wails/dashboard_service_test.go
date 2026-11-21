package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

func newTestDashboard(t *testing.T) (*DashboardService, *WorkspaceService) {
	t.Helper()
	isolateHome(t)
	ws := NewWorkspaceService()
	return NewDashboardService(ws, NewScenarioService(ws, NewSettingsService())), ws
}

func writeWS(t *testing.T, root, rel, content string) string {
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

const dashEnv = `schemaVersion: "1"
name: Local
variables:
  - name: baseUrl
    value: http://localhost:1
`

const dashScenario = `schemaVersion: "1"
title: Users
specFile: ../specs/api.yaml
steps:
  - label: List
    request:
      method: GET
      url: "{{baseUrl}}/users"
`

func TestDashboard_NoWorkspace(t *testing.T) {
	d, _ := newTestDashboard(t)
	o := d.GetWorkspaceOverview()
	if o.Open || o.Attention == nil || o.Specs == nil || o.EnvVars == nil {
		t.Fatalf("no workspace: %+v", o)
	}
}

func TestDashboard_OverviewOfAWorkspace(t *testing.T) {
	d, ws := newTestDashboard(t)
	root := t.TempDir()
	writeWS(t, root, "environments/local.yaml", dashEnv)
	writeWS(t, root, "requests/a.yaml", "x")
	writeWS(t, root, "scenarios/users.yaml", dashScenario)

	ws.enterWorkspace(root)
	o := d.GetWorkspaceOverview()
	if !o.Open || o.Path != root || o.Temporary || o.Requests != 1 || o.Scenarios != 1 || o.Environments != 1 {
		t.Fatalf("overview: %+v", o)
	}
	if o.EnvironmentSource != envSourceNone {
		t.Fatalf("nothing chosen or remembered yet, source = %q", o.EnvironmentSource)
	}
	ids := map[string]bool{}
	for _, a := range o.Attention {
		ids[a.ID] = true
	}
	if !ids["no-environment"] || !ids["spec-missing"] {
		t.Fatalf("want no-environment and spec-missing, got %v", ids)
	}

	if err := ws.SelectEnvironment(filepath.Join(root, "environments", "local.yaml")); err != nil {
		t.Fatal(err)
	}
	o = d.GetWorkspaceOverview()
	if o.EnvironmentSource != envSourceChosen || o.EnvironmentName != "Local" || len(o.EnvVars) != 1 {
		t.Fatalf("after choosing: %+v", o)
	}
	for _, a := range o.Attention {
		if a.ID == "no-environment" {
			t.Fatal("an environment is selected now")
		}
	}
}

func TestDashboard_EnvironmentSourceRememberedAndTemporary(t *testing.T) {
	d, ws := newTestDashboard(t)
	root := t.TempDir()
	envPath := writeWS(t, root, "environments/staging.yaml", dashEnv)

	ws.enterWorkspace(root)
	if err := ws.SelectEnvironment(envPath); err != nil {
		t.Fatal(err)
	}
	ws.CloseWorkspace()
	if got := d.GetWorkspaceOverview().EnvironmentSource; got != envSourceNone {
		t.Fatalf("closed workspace: %q", got)
	}

	ws.enterWorkspace(root)
	if got := d.GetWorkspaceOverview().EnvironmentSource; got != envSourceRemembered {
		t.Fatalf("reopened workspace should say remembered, got %q", got)
	}

	temp := filepath.Join(t.TempDir(), demoWorkspacePrefix+"x")
	writeWS(t, temp, "environments/local.yaml", dashEnv)
	ws.openWorkspaceDirect(temp)
	o := d.GetWorkspaceOverview()
	if !o.Temporary || o.EnvironmentSource != envSourceTemporary {
		t.Fatalf("temporary workspace: temporary=%v source=%q", o.Temporary, o.EnvironmentSource)
	}
}

func TestDashboard_CloseReminder(t *testing.T) {
	_, ws := newTestDashboard(t)
	if ws.GetCloseReminder() != "" {
		t.Fatal("no workspace, no reminder")
	}
	root := t.TempDir()
	envPath := writeWS(t, root, "environments/local.yaml", dashEnv)
	ws.enterWorkspace(root)
	if got := ws.GetCloseReminder(); !strings.Contains(got, "No environment will be remembered") {
		t.Fatalf("none chosen: %q", got)
	}
	if err := ws.SelectEnvironment(envPath); err != nil {
		t.Fatal(err)
	}
	if got := ws.GetCloseReminder(); !strings.Contains(got, `"Local" will be remembered`) {
		t.Fatalf("chosen: %q", got)
	}
	ws.openWorkspaceDirect(filepath.Join(t.TempDir(), conformanceWorkspacePrefix+"y"))
	if got := ws.GetCloseReminder(); !strings.Contains(got, "temporary workspace") {
		t.Fatalf("temporary: %q", got)
	}
}

func TestDashboard_ReadsOnly(t *testing.T) {
	d, ws := newTestDashboard(t)
	root := t.TempDir()
	writeWS(t, root, "scenarios/users.yaml", dashScenario)
	ws.enterWorkspace(root)
	before := dirListing(t, root)
	d.GetWorkspaceOverview()
	if after := dirListing(t, root); after != before {
		t.Fatalf("the dashboard changed the workspace:\n%s\n%s", before, after)
	}
}

func dirListing(t *testing.T, root string) string {
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

func TestCoverageTotals(t *testing.T) {
	c := coverageTotals(AccumulatedCoverageIO{
		Total: 10, Covered: 6,
		OperationResults: map[string]OperationResultIO{"a": {LastPassed: true}, "b": {LastPassed: true}, "c": {LastPassed: false}},
	})
	if c.NotCovered != 4 || c.Passing != 2 || c.Failing != 1 {
		t.Fatalf("totals: %+v", c)
	}
}

// One spec file that two scenarios spell differently shares one stored coverage entry (the store
// keys by letters and digits only), so Contracts shows one card. The page must find that entry
// whichever spelling it looks up, show it once, and warn about a failure in it.
func TestDashboard_OneSpecNamedTwoWaysShowsItsSharedCoverageOnce(t *testing.T) {
	d, ws := newTestDashboard(t)
	root := t.TempDir()
	envPath := writeWS(t, root, "environments/local.yaml", dashEnv)
	writeWS(t, root, "specs/api.yaml", "openapi: 3.0.0")
	writeWS(t, root, "scenarios/a.yaml", dashScenario)
	writeWS(t, root, "scenarios/deep/b.yaml", strings.Replace(dashScenario, "../specs/api.yaml", "../../specs/api.yaml", 1))
	ws.enterWorkspace(root)
	if err := ws.SelectEnvironment(envPath); err != nil {
		t.Fatal(err)
	}

	if err := saveStoredCoverage(root, coverage.StoredCoverage{SpecFile: "../specs/api.yaml", TotalRuns: 1, PassedRuns: 1, TotalOps: 4, Exercised: []string{"GET /a"},
		OperationResults: map[string]coverage.OperationResult{"GET /a": {LastPassed: true}}}); err != nil {
		t.Fatal(err)
	}
	// A run of the second scenario loads the same entry (same key) and adds to it.
	st := loadStoredCoverage(root, "../../specs/api.yaml")
	st.TotalRuns++
	st.FailedRuns++
	st.Exercised = append(st.Exercised, "GET /b")
	st.OperationResults["GET /b"] = coverage.OperationResult{LastPassed: false}
	if err := saveStoredCoverage(root, st); err != nil {
		t.Fatal(err)
	}

	o := d.GetWorkspaceOverview()
	if len(o.Specs) != 1 || len(o.Specs[0].Refs) != 2 {
		t.Fatalf("one spec with two spellings: %+v", o.Specs)
	}
	cov := o.Specs[0].Coverage
	if len(cov) != 1 || cov[0].Passing != 1 || cov[0].Failing != 1 || cov[0].Covered != 2 {
		t.Fatalf("the shared coverage should be shown once with both runs, got %+v", cov)
	}
	failing := 0
	for _, a := range o.Attention {
		if a.ID == "spec-failing" {
			failing++
		}
	}
	if failing != 1 {
		t.Fatalf("one warning for the one shared entry, got %d", failing)
	}
}

// The bundled sample workspaces are healthy, so the page must not warn about them: a false warning
// on the first workspace a new user opens would teach them to ignore the list.
func TestDashboard_SampleWorkspacesNeedNoAttention(t *testing.T) {
	isolateHome(t)

	demo := filepath.Join(t.TempDir(), demoWorkspacePrefix+"x")
	if err := extractDemoWorkspace(demo, 8090); err != nil {
		t.Fatal(err)
	}
	conf := t.TempDir()
	if err := writeConformanceWorkspace(conf, 8120); err != nil {
		t.Fatal(err)
	}

	for name, root := range map[string]string{"demo": demo, "conformance": conf} {
		ws := NewWorkspaceService()
		ws.openWorkspaceDirect(root)
		d := NewDashboardService(ws, NewScenarioService(ws, NewSettingsService()))
		o := d.GetWorkspaceOverview()
		if o.EnvironmentName == "" {
			t.Fatalf("%s: the sample environment should be active", name)
		}
		for _, a := range o.Attention {
			t.Errorf("%s workspace: unexpected attention item %s: %s (%s)", name, a.ID, a.What, a.Where)
		}
	}
}
