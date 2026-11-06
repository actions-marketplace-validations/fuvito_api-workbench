package conformance

import (
	"testing"
)

func TestComputeCoverage_allCovered(t *testing.T) {
	result := &ConformResult{
		Operations: []OperationResult{
			{Method: "GET", Path: "/health", StatusCode: 200},
			{Method: "POST", Path: "/users", StatusCode: 201},
			{Method: "GET", Path: "/users", StatusCode: 200},
		},
	}
	cov := ComputeCoverage(result)
	if cov.Total != 3 {
		t.Fatalf("Total: got %d, want 3", cov.Total)
	}
	if cov.Covered != 3 {
		t.Fatalf("Covered: got %d, want 3", cov.Covered)
	}
	if cov.NotCovered != 0 {
		t.Fatalf("NotCovered: got %d, want 0", cov.NotCovered)
	}
	if cov.Percent != 100 {
		t.Fatalf("Percent: got %.1f, want 100", cov.Percent)
	}
	if len(cov.NotCoveredOps) != 0 {
		t.Fatalf("NotCoveredOps should be empty")
	}
}

func TestComputeCoverage_partial(t *testing.T) {
	result := &ConformResult{
		Operations: []OperationResult{
			{Method: "GET", Path: "/health", StatusCode: 200},
			{Method: "DELETE", Path: "/users/{id}", Skipped: true, StatusCode: 0},
			{Method: "GET", Path: "/protected", Error: "connection refused", StatusCode: 0},
			{Method: "GET", Path: "/users", StatusCode: 200},
		},
	}
	cov := ComputeCoverage(result)
	if cov.Total != 4 {
		t.Fatalf("Total: got %d, want 4", cov.Total)
	}
	if cov.Covered != 2 {
		t.Fatalf("Covered: got %d, want 2", cov.Covered)
	}
	if cov.NotCovered != 2 {
		t.Fatalf("NotCovered: got %d, want 2", cov.NotCovered)
	}
	if cov.Percent != 50 {
		t.Fatalf("Percent: got %.1f, want 50", cov.Percent)
	}
	if len(cov.CoveredOps) != 2 {
		t.Fatalf("CoveredOps: got %d entries, want 2", len(cov.CoveredOps))
	}
	if len(cov.NotCoveredOps) != 2 {
		t.Fatalf("NotCoveredOps: got %d entries, want 2", len(cov.NotCoveredOps))
	}
}

func TestComputeCoverage_empty(t *testing.T) {
	cov := ComputeCoverage(&ConformResult{})
	if cov.Total != 0 || cov.Covered != 0 || cov.Percent != 0 {
		t.Fatalf("empty result should produce zero coverage, got %+v", cov)
	}
}

func TestComputeCoverage_noneReached(t *testing.T) {
	result := &ConformResult{
		Operations: []OperationResult{
			{Method: "GET", Path: "/a", Skipped: true},
			{Method: "GET", Path: "/b", Error: "timeout"},
		},
	}
	cov := ComputeCoverage(result)
	if cov.Covered != 0 {
		t.Fatalf("Covered: got %d, want 0", cov.Covered)
	}
	if cov.Percent != 0 {
		t.Fatalf("Percent: got %.1f, want 0", cov.Percent)
	}
}
