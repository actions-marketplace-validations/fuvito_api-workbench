package domain

// AssertionType identifies what part of the response is being validated.
// schema is used for API Contract Validation against an OpenAPI response schema.
type AssertionType string

const (
	AssertionStatus   AssertionType = "status"
	AssertionHeader   AssertionType = "header"
	AssertionJSONPath AssertionType = "jsonpath"
	AssertionSchema   AssertionType = "schema" // JSON Schema / OpenAPI contract assertion
)

type Operator string

const (
	OperatorEq       Operator = "eq"
	OperatorNe       Operator = "ne"
	OperatorContains Operator = "contains"
	OperatorMatches  Operator = "matches" // regex
	OperatorGt       Operator = "gt"
	OperatorLt       Operator = "lt"
)

// Assertion defines a single check against an HTTP response.
// Expected is any because valid values include strings, numbers, and booleans.
type Assertion struct {
	Type     AssertionType
	Name     string   // optional label for reports
	Path     string   // header name or jsonpath expression
	Operator Operator // defaults to OperatorEq when empty
	Expected any
	Schema   string // inline JSON Schema document for AssertionSchema assertions
}

// AssertionResult records whether a single Assertion passed.
type AssertionResult struct {
	Assertion Assertion
	Passed    bool
	Actual    string
	Message   string
}
