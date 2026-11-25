package reporting

import (
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// Locator says where an annotation for an operation points: a file (relative to the repository,
// with forward slashes) and a line in it. An empty file leaves the annotation on the run; a zero
// line leaves it on the file.
type Locator func(method, specPath string) (file string, line int)

// ConformGitHub renders a conformance run as GitHub Actions workflow commands, so every finding
// appears on the pull request (and in the log) with its check ID in the title and, when the spec is
// in the repository, on the line of the operation. A failure is an error, a warning a warning. It
// ends with one notice that says how the run went. Text that came from a response goes through the
// same redaction as every other report.
func ConformGitHub(result *conformance.ConformResult, neg *conformance.NegativeResult, loc Locator) string {
	var b strings.Builder
	x := newRedactor(result.Secrets, false)
	annotate := func(level, file string, line int, title, message string) {
		props := ""
		if file != "" {
			props += "file=" + escapeProperty(file)
			if line > 0 {
				props += fmt.Sprintf(",line=%d", line)
			}
		}
		if title != "" {
			if props != "" {
				props += ","
			}
			props += "title=" + escapeProperty(title)
		}
		fmt.Fprintf(&b, "::%s %s::%s\n", level, props, escapeData(message))
	}

	for _, op := range result.Operations {
		if op.Skipped {
			continue
		}
		specPath := op.ReportPath()
		file, line := "", 0
		if loc != nil {
			file, line = loc(op.Method, specPath)
		}
		for _, f := range explain(op, x) {
			level := "error"
			switch f.Severity {
			case checks.Warning:
				level = "warning"
			case checks.Note:
				level = "notice"
			}
			title := strings.TrimSpace(f.CheckID + " " + op.Method + " " + specPath)
			annotate(level, file, line, x.text(title), findingMessage(f))
		}
	}
	if neg != nil {
		for _, p := range neg.Probes {
			if p.Skipped {
				continue
			}
			var id, msg string
			switch {
			case p.Error != "":
				id, msg = "CS-PROBE-006", "The server did not answer this probe: "+x.text(p.Error)
			case !p.Passed():
				id = conformance.ProbeCheckID(string(p.Kind))
				msg = fmt.Sprintf("The server accepted an invalid request (status %d): %s. It should have refused it with a 4xx.", p.StatusCode, x.text(p.Description))
			default:
				continue
			}
			file, line := "", 0
			if loc != nil {
				file, line = loc(p.Method, p.Path)
			}
			annotate("error", file, line, strings.TrimSpace(id+" "+p.Method+" "+p.Path), msg)
		}
	}

	passed, failed, notSent, skipped := conformCounts(result)
	unsent := ""
	if notSent > 0 {
		unsent = fmt.Sprintf(", %d not sent", notSent)
	}
	summary := fmt.Sprintf("%d operations: %d passed, %d failed%s, %d skipped.", passed+failed+notSent+skipped, passed, failed, unsent, skipped)
	if line := result.Scope.Line(); line != "" {
		summary += " " + line
	}
	if w := result.WarningCount(); w > 0 {
		summary += fmt.Sprintf(" %d with warnings.", w)
	}
	annotate("notice", "", 0, "Clear Signal conform", x.text(summary))
	return b.String()
}

// findingMessage is the text of an annotation: what is wrong, why that fails and how to fix it.
func findingMessage(f conformance.Finding) string {
	var parts []string
	parts = append(parts, strings.TrimSuffix(f.Summary, "."))
	if where := f.WhereText(); where != "" {
		parts = append(parts, "Where: "+where)
	}
	if f.Why != "" {
		parts = append(parts, "Why: "+strings.TrimSuffix(f.Why, "."))
	}
	if f.Fix != "" {
		parts = append(parts, "Fix: "+strings.TrimSuffix(f.Fix, "."))
	}
	return strings.Join(parts, ". ") + "."
}

// escapeData and escapeProperty follow the workflow command rules: a command is one line, and a
// property value may not contain the characters that separate properties.
func escapeData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

func escapeProperty(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C").Replace(s)
}
