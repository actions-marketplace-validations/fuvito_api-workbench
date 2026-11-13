package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oasdiff/oasdiff/checker"
	"github.com/oasdiff/oasdiff/diff"
	"github.com/oasdiff/oasdiff/load"
)

// DiffChange describes a single change between two spec versions.
type DiffChange struct {
	// ID is the oasdiff rule identifier (e.g. "endpoint-removed").
	ID string
	// Text is a human-readable description of the change.
	Text string
	// Method is the HTTP method of the affected operation (may be empty for spec-level changes).
	Method string
	// Path is the URL path of the affected operation (may be empty for spec-level changes).
	Path string
	// Breaking is true when the change is not backward-compatible.
	Breaking bool
	// Level is ERR (breaking), WARN, or INFO.
	Level string
}

// DiffResult holds the full set of changes between two spec versions.
type DiffResult struct {
	Changes     []DiffChange
	HasBreaking bool
}

// Diff compares two already-parsed specs and classifies every change as
// breaking or non-breaking using oasdiff's default rule set.
func Diff(base, revision *Spec) (*DiffResult, error) {
	baseSI := &load.SpecInfo{Spec: base.doc, Url: specURL(base.doc)}
	revSI := &load.SpecInfo{Spec: revision.doc, Url: specURL(revision.doc)}

	diffReport, operationsSources, err := diff.GetWithOperationsSourcesMap(diff.NewConfig(), baseSI, revSI)
	if err != nil {
		return nil, err
	}

	bcConfig := checker.NewConfig(checker.GetAllChecks())
	changes := checker.CheckBackwardCompatibilityUntilLevel(bcConfig, diffReport, operationsSources, checker.INFO)

	l := checker.NewDefaultLocalizer()
	result := &DiffResult{}
	for _, c := range changes {
		level := "INFO"
		breaking := false
		switch c.GetLevel() {
		case checker.ERR:
			level = "ERR"
			breaking = true
			result.HasBreaking = true
		case checker.WARN:
			level = "WARN"
		}
		result.Changes = append(result.Changes, DiffChange{
			ID:       c.GetId(),
			Text:     c.GetUncolorizedText(l),
			Method:   c.GetOperation(),
			Path:     c.GetPath(),
			Breaking: breaking,
			Level:    level,
		})
	}
	return result, nil
}

func specURL(doc *openapi3.T) string {
	if doc == nil || doc.Info == nil {
		return ""
	}
	return doc.Info.Title
}
