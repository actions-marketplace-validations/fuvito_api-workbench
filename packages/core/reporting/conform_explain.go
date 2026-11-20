package reporting

import (
	"strings"

	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// explain returns the plain-language findings for an operation with every text field redacted:
// the "got" value and the validator's message come from the server's response, so they can contain
// secrets exactly like the raw violation lines do.
func explain(op conformance.OperationResult, x redactor) []conformance.Finding {
	findings := conformance.Diagnose(op)
	for i := range findings {
		f := &findings[i]
		f.Part, f.Where, f.Spec = x.text(f.Part), x.text(f.Where), x.text(f.Spec)
		f.Summary, f.Expected = x.text(f.Summary), x.text(f.Expected)
		f.Got, f.Why, f.Fix, f.Detail = x.text(f.Got), x.text(f.Why), x.text(f.Fix), x.text(f.Detail)
	}
	return findings
}

// explanationLine is the one-line form used by the text, HTML, Markdown and JUnit reports.
func explanationLine(f conformance.Finding) string {
	var b strings.Builder
	b.WriteString(f.Summary)
	b.WriteString(". Why: ")
	b.WriteString(strings.TrimSuffix(f.Why, "."))
	if where := f.WhereText(); where != "" {
		b.WriteString(". Where: ")
		b.WriteString(where)
		if f.Spec != "" {
			b.WriteString(" (spec: " + f.Spec + ")")
		}
	}
	b.WriteString(". Fix: ")
	b.WriteString(strings.TrimSuffix(f.Fix, "."))
	b.WriteString(".")
	return b.String()
}
