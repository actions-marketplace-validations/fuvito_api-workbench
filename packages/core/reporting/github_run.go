package reporting

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// GitHubRun is one scenario run to annotate: the file it came from (relative to the repository,
// with forward slashes; empty leaves annotations on the run) and the line of each step in it.
type GitHubRun struct {
	Result    *scenario.Result
	File      string
	StepLines []int
}

// RunGitHub renders scenario runs as GitHub Actions workflow commands: an error for each step that
// failed (a request that could not be sent, a failed assertion or extraction, a response that breaks
// the spec, with the check ID of the spec problem in the title), on the line of the step in the
// scenario file, and one closing notice with the counts. Text that came from a response is redacted
// like every other report.
func RunGitHub(runs []GitHubRun) string {
	var b strings.Builder
	annotate := func(level, file string, line int, title, message string) {
		props := ""
		if file != "" {
			props += "file=" + escapeProperty(file)
			if line > 0 {
				props += fmt.Sprintf(",line=%d", line)
			}
		}
		if props != "" {
			props += ","
		}
		props += "title=" + escapeProperty(title)
		fmt.Fprintf(&b, "::%s %s::%s\n", level, props, escapeData(message))
	}

	passedScenarios := 0
	for _, run := range runs {
		r := run.Result
		x := redactorFor(r)
		name := r.Scenario.Title
		if r.Passed {
			passedScenarios++
		}
		if r.SetupError != "" {
			annotate("error", run.File, 0, name, "The scenario could not start: "+x.text(r.SetupError))
		}
		for i, sr := range r.Steps {
			if sr.Skipped {
				continue
			}
			line := 0
			if i < len(run.StepLines) {
				line = run.StepLines[i]
			}
			title := name + ": " + sr.Label
			if sr.Error != "" {
				annotate("error", run.File, line, title, x.text(sr.Error))
			}
			for _, ar := range sr.Assertions {
				if ar.Passed {
					continue
				}
				msg := assertionLabel(ar.Assertion)
				if ar.Message != "" {
					msg += ": " + ar.Message
				}
				annotate("error", run.File, line, title, x.text(msg))
			}
			if sr.Spec.State == scenario.SpecViolations && sr.Request != nil {
				op := conformance.OperationResult{Method: string(sr.Request.Method), Path: pathOf(sr.Request.URL), Violations: sr.Spec.Violations}
				if sr.Response != nil {
					op.StatusCode = sr.Response.StatusCode
				}
				for _, f := range explain(op, x) {
					annotate("error", run.File, line, strings.TrimSpace(f.CheckID+" "+title), findingMessage(f))
				}
			}
		}
	}

	summary := fmt.Sprintf("%d scenarios: %d passed, %d failed.", len(runs), passedScenarios, len(runs)-passedScenarios)
	annotate("notice", "", 0, "Clear Signal run", summary)
	return b.String()
}

func pathOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		return u.Path
	}
	return raw
}
