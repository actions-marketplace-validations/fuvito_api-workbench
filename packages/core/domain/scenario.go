package domain

import "time"

type FailPolicy string

const (
	FailPolicyStop     FailPolicy = "stop"
	FailPolicyContinue FailPolicy = "continue"
	FailPolicySkip     FailPolicy = "skip"
)

type RetryPolicy struct {
	MaxAttempts int
	Delay       time.Duration
}

// ScenarioStep is one step in a Scenario.
// Either RequestRef or Request must be set, not both.
// RequestRef is a path to a saved request file relative to the workspace root.
type ScenarioStep struct {
	Name         string
	Description  string
	Disabled     bool // when true, step is skipped without being counted as a failure
	RequestRef   string
	Request      *Request
	Extractions  []Extraction
	Assertions   []Assertion
	FailPolicy   FailPolicy
	Retry        *RetryPolicy
	TimeoutMs    int
	ValidateSpec *bool // nil = inherit scenario default; explicit true/false overrides per step
}

// Scenario is a sequential, multi-step workflow used for Spec Conformance Testing.
type Scenario struct {
	SchemaVersion string
	Name          string
	Description   string
	SpecFile      string // optional path to OpenAPI spec; enables automatic response schema validation
	Steps         []ScenarioStep
}
