package domain

// Variable is a named value available for {{variable}} interpolation.
type Variable struct {
	Name  string
	Value string
}

type ExtractionSource string

const (
	ExtractJSONPath ExtractionSource = "jsonpath"
	ExtractHeader   ExtractionSource = "header"
	ExtractStatus   ExtractionSource = "status"
)

// Extraction defines how to pull a value from a Response and store it as a runtime variable.
// Used in scenario steps to chain request data across steps.
type Extraction struct {
	Name   string           // variable name to assign
	Source ExtractionSource
	Path   string           // jsonpath expression or header name
}
