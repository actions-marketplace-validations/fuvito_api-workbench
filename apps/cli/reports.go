package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// reportFormats are the formats run, conform, diff and bundle run can write.
var reportFormats = []string{"text", "json", "junit", "html", "markdown"}

const formatUsage = "Output format: text, json, junit, html, or markdown"

func validFormat(f string) bool {
	for _, k := range reportFormats {
		if f == k {
			return true
		}
	}
	return false
}

// reportPlan is the primary report plus any extra files requested with --report.
type reportPlan struct {
	format string
	extras []extraReport
}

type extraReport struct {
	format string
	file   string
}

// planReports checks --format and every --report value (format=file) so a typo is reported
// before a single request is sent.
func planReports(format string, reports []string, outputFile string) (reportPlan, error) {
	return planReportsWith(nil, format, reports, outputFile)
}

// planReportsWith is planReports for a command that also offers formats meant for the screen only
// (primary): they are accepted as --format but not as a --report file.
func planReportsWith(primary []string, format string, reports []string, outputFile string) (reportPlan, error) {
	plan := reportPlan{format: strings.ToLower(format)}
	isPrimary := false
	for _, p := range primary {
		isPrimary = isPrimary || p == plan.format
	}
	if !validFormat(plan.format) && !isPrimary {
		return plan, fmt.Errorf("unknown --format %q (use one of: %s)", format, strings.Join(append(append([]string(nil), reportFormats...), primary...), ", "))
	}
	seen := map[string]bool{}
	if outputFile != "" {
		seen[filepath.Clean(outputFile)] = true
	}
	for _, raw := range reports {
		f, file, ok := strings.Cut(raw, "=")
		f, file = strings.ToLower(strings.TrimSpace(f)), strings.TrimSpace(file)
		if !ok || f == "" || file == "" {
			return plan, fmt.Errorf("--report %q must look like format=file, for example junit=reports/run.xml", raw)
		}
		if !validFormat(f) {
			return plan, fmt.Errorf("--report %q: unknown format %q (use one of: %s)", raw, f, strings.Join(reportFormats, ", "))
		}
		if seen[filepath.Clean(file)] {
			return plan, fmt.Errorf("--report %q: %s is already used by another report or by --output", raw, file)
		}
		seen[filepath.Clean(file)] = true
		plan.extras = append(plan.extras, extraReport{format: f, file: file})
	}
	return plan, nil
}

// emit writes the primary report (to outputFile, or the screen when empty) and then each extra
// report to its own file. render produces the report in one format.
func (p reportPlan) emit(cmd *cobra.Command, outputFile string, render func(format string) ([]byte, error)) error {
	primary, err := render(p.format)
	if err != nil {
		return err
	}
	if err := writeReport(cmd, outputFile, primary); err != nil {
		return err
	}
	for _, e := range p.extras {
		data, err := render(e.format)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(e.file), 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		if err := os.WriteFile(e.file, data, 0o644); err != nil {
			return fmt.Errorf("writing %s report: %w", e.format, err)
		}
	}
	return nil
}
