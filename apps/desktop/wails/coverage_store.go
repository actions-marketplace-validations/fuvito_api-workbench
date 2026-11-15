package main

import (
	"time"

	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// OperationResultIO is the Wails IPC representation of a per-operation last-run result.
type OperationResultIO struct {
	LastPassed bool   `json:"lastPassed"`
	LastRunAt  string `json:"lastRunAt,omitempty"`
}

// AccumulatedCoverageIO is the Wails IPC representation of accumulated spec coverage.
type AccumulatedCoverageIO struct {
	SpecFile         string                       `json:"specFile"`
	Covered          int                          `json:"covered"`
	Total            int                          `json:"total"`
	Percent          float64                      `json:"percent"`
	TotalRuns        int                          `json:"totalRuns"`
	PassedRuns       int                          `json:"passedRuns"`
	FailedRuns       int                          `json:"failedRuns"`
	LastRunAt        string                       `json:"lastRunAt,omitempty"`
	Exercised        []string                     `json:"exercised"`
	Missed           []string                     `json:"missed"`
	OperationResults map[string]OperationResultIO `json:"operationResults,omitempty"`
}

// coverageKey / coverageDirPath / loadStoredCoverage / saveStoredCoverage /
// mergeExercised are thin aliases kept here so scenario_service.go call sites
// remain unchanged while the real logic lives in packages/core/coverage.

func coverageKey(specFile string) string              { return coverage.Key(specFile) }
func coverageDirPath(workspacePath string) string     { return coverage.DirPath(workspacePath) }
func loadStoredCoverage(wsPath, specFile string) coverage.StoredCoverage {
	return coverage.Load(wsPath, specFile)
}
func saveStoredCoverage(wsPath string, cov coverage.StoredCoverage) error {
	return coverage.Save(wsPath, cov)
}
func mergeExercised(existing, newOps []string) []string {
	return coverage.MergeExercised(existing, newOps)
}

// buildAccumulatedCoverageIO converts a coverage.StoredCoverage + loaded spec into
// the Wails IPC type, computing missed operations from the spec.
func buildAccumulatedCoverageIO(stored coverage.StoredCoverage, spec *openapi.Spec) AccumulatedCoverageIO {
	all := spec.AllOperations()
	allOps := make([]string, len(all))
	for i, op := range all {
		allOps[i] = op.String()
	}
	r := coverage.BuildReport(stored, allOps)

	lastRun := ""
	if !r.LastRunAt.IsZero() {
		lastRun = r.LastRunAt.Format(time.RFC3339)
	}
	missed := r.Missed
	if missed == nil {
		missed = []string{}
	}
	var opResults map[string]OperationResultIO
	if len(r.OperationResults) > 0 {
		opResults = make(map[string]OperationResultIO, len(r.OperationResults))
		for k, v := range r.OperationResults {
			or_ := OperationResultIO{LastPassed: v.LastPassed}
			if !v.LastRunAt.IsZero() {
				or_.LastRunAt = v.LastRunAt.Format(time.RFC3339)
			}
			opResults[k] = or_
		}
	}
	return AccumulatedCoverageIO{
		SpecFile:         r.SpecFile,
		Covered:          r.Covered,
		Total:            r.Total,
		Percent:          r.Percent,
		TotalRuns:        r.TotalRuns,
		PassedRuns:       r.PassedRuns,
		FailedRuns:       r.FailedRuns,
		LastRunAt:        lastRun,
		Exercised:        r.Exercised,
		Missed:           missed,
		OperationResults: opResults,
	}
}

// reportToIO converts a coverage.Report (from ListAll, which has no Missed) to the IPC type.
func reportToIO(r coverage.Report) AccumulatedCoverageIO {
	lastRun := ""
	if !r.LastRunAt.IsZero() {
		lastRun = r.LastRunAt.Format(time.RFC3339)
	}
	exercised := r.Exercised
	if exercised == nil {
		exercised = []string{}
	}
	missed := r.Missed
	if missed == nil {
		missed = []string{}
	}
	var opResults map[string]OperationResultIO
	if len(r.OperationResults) > 0 {
		opResults = make(map[string]OperationResultIO, len(r.OperationResults))
		for k, v := range r.OperationResults {
			or_ := OperationResultIO{LastPassed: v.LastPassed}
			if !v.LastRunAt.IsZero() {
				or_.LastRunAt = v.LastRunAt.Format(time.RFC3339)
			}
			opResults[k] = or_
		}
	}
	return AccumulatedCoverageIO{
		SpecFile:         r.SpecFile,
		Covered:          r.Covered,
		Total:            r.Total,
		Percent:          r.Percent,
		TotalRuns:        r.TotalRuns,
		PassedRuns:       r.PassedRuns,
		FailedRuns:       r.FailedRuns,
		LastRunAt:        lastRun,
		Exercised:        exercised,
		Missed:           missed,
		OperationResults: opResults,
	}
}
