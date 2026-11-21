package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
