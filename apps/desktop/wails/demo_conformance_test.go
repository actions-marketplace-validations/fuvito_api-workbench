package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

// The Check Conformance page lists spec files found in the workspace. The Demo Server's scenarios
// name its spec by URL, so the workspace must also carry the spec as a file, or the page's spec
// list is empty for the Demo Server.
func TestDemoWorkspace_CheckConformancePageListsAndRunsTheSpec(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	d := NewDemoService(ws)
	res := d.StartDemo()
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	defer d.StopDemo()

	svc := NewConformanceService(ws, NewSettingsService())
	ctx := svc.GetConformContext()
	if len(ctx.Specs) != 1 || ctx.Specs[0].Label != "specs/openapi.yaml" {
		t.Fatalf("the page should list the demo spec, got %+v", ctx.Specs)
	}
	if ctx.ActiveEnvironment == "" {
		t.Fatal("the demo environment should be active")
	}

	info := svc.InspectConformSpec(ctx.Specs[0].Path)
	if info.Error != "" {
		t.Fatal(info.Error)
	}
	if want := fmt.Sprintf("http://localhost:%d", res.Port); info.ServerURL != want {
		t.Fatalf("the spec's server must be the port the Demo Server runs on: got %q, want %q", info.ServerURL, want)
	}

	// Read-only is the page's default: the demo's data is only changed when the user opts in.
	out := svc.RunConformance(ConformRunInput{SpecPath: ctx.Specs[0].Path})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if out.BlockedOperations != 0 || len(out.MissingValues) != 0 {
		t.Fatalf("the demo environment should supply every value the spec needs: missing %v", out.MissingValues)
	}
	if out.Failing != 0 || out.Covered != info.Reads || out.Passing != out.Covered {
		t.Fatalf("the Demo API should match its own spec on every read: %+v (reads %d)", out, info.Reads)
	}
}

// A scenario run and a conformance run on the Demo Server must feed one Contracts card: the
// scenarios name the spec file, and the page records its run under the name the scenarios use.
func TestDemoWorkspace_ScenarioAndConformanceRunsShareOneContractsCard(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	d := NewDemoService(ws)
	res := d.StartDemo()
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	defer d.StopDemo()

	scenarios := NewScenarioService(ws, NewSettingsService())
	run := scenarios.RunScenario(filepath.Join(res.WorkspacePath, "scenarios", "1-full-coverage", "health.yaml"), "", 0)
	if run.Error != "" || run.SpecError != "" || run.AccumulatedCoverage == nil {
		t.Fatalf("scenario run: error=%q specError=%q coverage=%v", run.Error, run.SpecError, run.AccumulatedCoverage)
	}

	svc := NewConformanceService(ws, NewSettingsService())
	spec := svc.GetConformContext().Specs[0].Path
	out := svc.RunConformance(ConformRunInput{SpecPath: spec, RecordCoverage: true})
	if out.Error != "" {
		t.Fatal(out.Error)
	}

	cards := scenarios.ListAccumulatedCoverage()
	if len(cards) != 1 {
		var names []string
		for _, c := range cards {
			names = append(names, c.SpecFile)
		}
		t.Fatalf("want one Contracts card for the Demo spec, got %d: %v (%s)", len(cards), names, out.CoverageNote)
	}
	if cards[0].TotalRuns != 2 {
		t.Fatalf("both runs belong on that card, got %d runs (scenario run counted coverage: %v; conformance: error %q, covered %d, note %q; demo port %d)",
			cards[0].TotalRuns, run.AccumulatedCoverage != nil, out.Error, out.Covered, out.CoverageNote, res.Port)
	}

	// Opening the card reloads the spec by the name the scenarios wrote, which is relative to the
	// scenario's folder, not to wherever the app was started.
	detail, err := scenarios.GetAccumulatedCoverage(cards[0].SpecFile)
	if err != nil || detail == nil || detail.Total != 24 {
		t.Fatalf("opening the Contracts card for %q: %+v, %v", cards[0].SpecFile, detail, err)
	}
	for _, format := range []string{"markdown", "html", "csv"} {
		if rep := scenarios.ExportCoverageReport(cards[0].SpecFile, format); rep.Error != "" || rep.Content == "" {
			t.Fatalf("export %s: %q", format, rep.Error)
		}
	}
}
