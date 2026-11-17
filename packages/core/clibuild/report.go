package clibuild

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/clispec"
)

// ReportsDir is the folder suggested for generated report files.
const ReportsDir = "reports"

var formatExt = map[string]string{"html": "html", "junit": "xml", "json": "json", "text": "txt"}

var formatNote = map[string]string{
	"text":  "a plain-text report",
	"json":  "a machine-readable JSON report",
	"junit": "a JUnit XML report for CI test dashboards",
	"html":  "a self-contained HTML report you can open in a browser",
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a name into a safe file-name part: lower case, letters and digits, single dashes.
func Slug(name string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// SuggestOutput proposes a report path such as reports/run-user-lifecycle.html. subject is
// what the report is about (a scenario or spec file name); it may be empty.
func SuggestOutput(command, subject, format string) string {
	ext := formatExt[format]
	if ext == "" {
		ext = "txt"
	}
	name := command
	if slug := Slug(strings.TrimSuffix(filepath.Base(subject), filepath.Ext(subject))); slug != "" && slug != command && subject != "" {
		name = command + "-" + slug
	}
	return ReportsDir + "/" + name + "." + ext
}

// outputNotes tells the user what will happen to the file they chose.
func outputNotes(field, text, root string, add func(Message)) {
	full := text
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, filepath.FromSlash(text))
	}
	if _, err := os.Stat(full); err == nil {
		add(Message{Field: field, Severity: Warning, Text: text + " already exists and will be overwritten"})
		return
	}
	if dir := filepath.Dir(full); dir != root {
		if _, err := os.Stat(dir); err != nil {
			add(Message{Field: field, Severity: Info, Text: "the folder " + filepath.ToSlash(relOrSelf(root, dir)) + " will be created"})
		}
	}
}

func relOrSelf(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// checkReport looks at the format and output choices together and proposes a file name when
// a non-text report would otherwise be printed to the screen.
func checkReport(cmd clispec.Command, values map[string]string, add func(Message)) []Suggestion {
	if _, ok := cmd.Input("format"); !ok {
		return nil
	}
	format := values["format"]
	if format == "" {
		format = "text"
	}
	output := values["output"]

	switch {
	case output != "" && format == "text":
		add(Message{Field: "output", Severity: Info, Text: "a plain-text report will be written to this file; nothing is printed to the screen"})
	case output != "":
		add(Message{Field: "output", Severity: Info, Text: fmt.Sprintf("%s will be written to this file", formatNote[format])})
	case format == "html" || format == "junit":
		add(Message{Field: "output", Severity: Warning, Text: fmt.Sprintf("without a file, the whole %s report is printed to the screen; choose where to save it", strings.ToUpper(format))})
	}

	if output != "" || format == "text" {
		return nil
	}
	subject := ""
	for _, name := range []string{"scenario-file", "spec", "base-spec"} {
		if v := values[name]; v != "" && !strings.HasPrefix(v, "http") {
			subject = v
			break
		}
	}
	return []Suggestion{{
		Field: "output",
		Value: SuggestOutput(cmd.Name, subject, format),
		Why:   "keeps reports together, named after the command and what it checks",
	}}
}
