package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fuvito/api-workbench/packages/core/workspaceinfo"
)

// DashboardService backs the Workspace page. It only reads: the checks live in
// packages/core/workspaceinfo, and this adds what only the app knows (where the environment choice
// came from, stored coverage and the scenario run log).
type DashboardService struct {
	ws       *WorkspaceService
	scenario *ScenarioService
}

func NewDashboardService(ws *WorkspaceService, scenario *ScenarioService) *DashboardService {
	return &DashboardService{ws: ws, scenario: scenario}
}

// DashSpecIO is a spec the workspace's scenarios name, with its last stored coverage.
type DashSpecIO struct {
	Ref       string   `json:"ref"`
	Path      string   `json:"path,omitempty"`
	State     string   `json:"state"`
	Scenarios []string `json:"scenarios"`
	// Refs are the spellings scenarios used for this spec; stored coverage is keyed by spelling.
	Refs []string `json:"refs"`
	// Coverage has one entry per spelling that has recorded runs (usually one, empty when no run
	// has recorded this spec yet).
	Coverage []DashCoverageIO `json:"coverage"`
}

// DashCoverageIO carries the four Contracts totals: Covered + NotCovered = Total, Passing + Failing = Covered.
type DashCoverageIO struct {
	Ref        string `json:"ref"`
	Total      int    `json:"total"`
	Covered    int    `json:"covered"`
	NotCovered int    `json:"notCovered"`
	Passing    int    `json:"passing"`
	Failing    int    `json:"failing"`
	LastRunAt  string `json:"lastRunAt,omitempty"`
}

// DashLastRunIO summarises the scenario run log for this workspace. The log is stored on disk, so
// it survives a restart.
type DashLastRunIO struct {
	At       string `json:"at"`
	Scenario string `json:"scenario"`
	Passed   bool   `json:"passed"`
	// Scenarios counts the scenarios that have a run in the log; FailingNow how many of those
	// failed the last time they ran.
	Scenarios  int `json:"scenarios"`
	FailingNow int `json:"failingNow"`
}

// DashEnvVarIO never carries a value.
type DashEnvVarIO struct {
	Name     string `json:"name"`
	Secret   bool   `json:"secret"`
	Set      bool   `json:"set"`
	UnsetRef string `json:"unsetRef,omitempty"`
}

// WorkspaceOverviewIO is everything the Workspace page shows.
type WorkspaceOverviewIO struct {
	Open      bool   `json:"open"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Temporary bool   `json:"temporary"`

	Requests     int `json:"requests"`
	Scenarios    int `json:"scenarios"`
	Environments int `json:"environments"`

	EnvironmentName string `json:"environmentName,omitempty"`
	EnvironmentPath string `json:"environmentPath,omitempty"`
	// EnvironmentSource is remembered, chosen, temporary or none.
	EnvironmentSource string         `json:"environmentSource"`
	EnvVars           []DashEnvVarIO `json:"envVars"`

	Specs     []DashSpecIO              `json:"specs"`
	LastRun   *DashLastRunIO            `json:"lastRun,omitempty"`
	Attention []workspaceinfo.Attention `json:"attention"`
	// ProjectFile is the clrsig.yaml in the workspace folder, if there is one.
	ProjectFile workspaceinfo.ProjectFile `json:"projectFile"`

	SnapshotAt string `json:"snapshotAt"`
}

// GetWorkspaceOverview inspects the open workspace. It reads files and the stores but never
// changes anything.
func (d *DashboardService) GetWorkspaceOverview() WorkspaceOverviewIO {
	out := WorkspaceOverviewIO{
		EnvVars: []DashEnvVarIO{}, Specs: []DashSpecIO{}, Attention: []workspaceinfo.Attention{}, ProjectFile: workspaceinfo.ProjectFile{Keys: []string{}},
		SnapshotAt: time.Now().Format(time.RFC3339),
	}
	root := d.ws.workspacePath
	if root == "" {
		out.EnvironmentSource = envSourceNone
		return out
	}
	out.Open = true
	out.Path = root
	out.Name = filepath.Base(root)
	out.Temporary = isTempWorkspace(root)
	out.EnvironmentSource = d.ws.activeEnvSource
	if out.EnvironmentSource == "" {
		out.EnvironmentSource = envSourceNone
	}

	rep := workspaceinfo.Inspect(root, d.ws.activeEnv, d.ws.activeEnvPath)
	out.Requests, out.Scenarios, out.Environments = rep.Requests, rep.Scenarios, rep.Environments
	out.EnvironmentName, out.EnvironmentPath = rep.EnvironmentName, rep.EnvironmentPath
	for _, v := range rep.EnvVars {
		out.EnvVars = append(out.EnvVars, DashEnvVarIO{Name: v.Name, Secret: v.Secret, Set: v.Set, UnsetRef: v.UnsetRef})
	}
	out.Attention = rep.Attention
	out.ProjectFile = rep.ProjectFile

	// Look coverage up by the store's own key, as the store does: spellings that differ only in
	// punctuation (../specs/a.yaml and ../../specs/a.yaml) share one stored entry and one card.
	stored := map[string]AccumulatedCoverageIO{}
	for _, c := range d.scenario.ListAccumulatedCoverage() {
		stored[coverageKey(c.SpecFile)] = c
	}
	for _, sp := range rep.Specs {
		item := DashSpecIO{Ref: sp.Ref, Refs: sp.Refs, Path: sp.Path, State: sp.State, Scenarios: sp.Scenarios, Coverage: []DashCoverageIO{}}
		seen := map[string]bool{}
		for _, ref := range sp.Refs {
			key := coverageKey(ref)
			c, ok := stored[key]
			if !ok || seen[key] {
				continue
			}
			seen[key] = true
			ref = c.SpecFile // the spelling the Contracts card is shown under
			cov := coverageTotals(c)
			cov.Ref = ref
			item.Coverage = append(item.Coverage, cov)
			if cov.Failing > 0 {
				out.Attention = append(out.Attention, workspaceinfo.Attention{
					ID: "spec-failing", Severity: workspaceinfo.SeverityWarning,
					What:   fmt.Sprintf("%d of %d covered operations failed the last time they ran.", cov.Failing, cov.Covered),
					Where:  fmt.Sprintf("spec %s", ref),
					Why:    "The API did not match the spec, or an assertion failed, in the most recent run.",
					Fix:    "Open Contracts to see which operations failed, then open their scenario's Result tab.",
					Target: workspaceinfo.Target{Kind: "contracts", Path: ref},
				})
			}
		}
		out.Specs = append(out.Specs, item)
	}

	out.LastRun = d.lastRun()
	return out
}

func coverageTotals(c AccumulatedCoverageIO) DashCoverageIO {
	t := DashCoverageIO{Total: c.Total, Covered: c.Covered, LastRunAt: c.LastRunAt}
	t.NotCovered = c.Total - c.Covered
	for _, r := range c.OperationResults {
		if r.LastPassed {
			t.Passing++
		} else {
			t.Failing++
		}
	}
	return t
}

func (d *DashboardService) lastRun() *DashLastRunIO {
	runs := d.scenario.GetScenarioRunHistory(500)
	if len(runs) == 0 {
		return nil
	}
	last := &DashLastRunIO{At: runs[0].ExecutedAt, Scenario: runs[0].Title, Passed: runs[0].Passed}
	seen := map[string]bool{}
	for _, r := range runs {
		if seen[r.ScenarioPath] {
			continue
		}
		seen[r.ScenarioPath] = true
		last.Scenarios++
		if !r.Passed {
			last.FailingNow++
		}
	}
	return last
}
