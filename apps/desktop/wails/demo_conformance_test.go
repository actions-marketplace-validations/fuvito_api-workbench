package main

import (
	"fmt"
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
