package main

import (
	"fmt"
	"time"

	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

const contractsFormatUsage = "Output format: text, json, csv, markdown, or html"

// isContractsFormat reports whether f is one of the file-style Contracts report formats, which
// are produced by the shared Go renderers (the desktop Contracts export uses the same ones).
func isContractsFormat(f string) bool {
	for _, k := range reporting.ContractsFormats {
		if f == k {
			return true
		}
	}
	return false
}

func contractsReport(format, spec string, r coverage.Report, q *coverage.QualityReport) ([]byte, error) {
	now := time.Now()
	switch format {
	case "csv":
		return reporting.ContractsCSV(r, q), nil
	case "markdown":
		return reporting.ContractsMarkdown(spec, r, q, now), nil
	case "html":
		b, err := reporting.ContractsHTML(spec, r, q, now)
		if err != nil {
			return nil, fmt.Errorf("generating HTML report: %w", err)
		}
		return b, nil
	}
	return nil, fmt.Errorf("unknown format %q", format)
}

func contractsListReport(format string, reports []coverage.Report) ([]byte, error) {
	now := time.Now()
	switch format {
	case "csv":
		return reporting.ContractsListCSV(reports), nil
	case "markdown":
		return reporting.ContractsListMarkdown(reports, now), nil
	case "html":
		b, err := reporting.ContractsListHTML(reports, now)
		if err != nil {
			return nil, fmt.Errorf("generating HTML report: %w", err)
		}
		return b, nil
	}
	return nil, fmt.Errorf("unknown format %q", format)
}

func qualityHasGaps(q *coverage.QualityReport) bool {
	return len(q.NeverExercised) > 0 || len(q.HappyPathOnly) > 0 ||
		len(q.SecuredNeverAuthed) > 0 || len(q.ErrorsNeverTriggered) > 0
}
