// Package checks is the list of checks Clear Signal runs, as data: each has a stable ID, a title and
// a default severity. The IDs are the ones in the catalog of what is checked (Help, What We Check,
// apps/desktop/wails/help/what-we-check.md); tests keep the code, this list and that document
// in agreement.
package checks

import (
	"fmt"
	"sort"
	"sync"
)

// Severity says what a finding does to the run.
type Severity string

const (
	// Failure changes the exit code.
	Failure Severity = "failure"
	// Warning is reported and counted but does not change the exit code unless asked.
	Warning Severity = "warning"
	// Note is information: something was not checked, or is worth knowing.
	Note Severity = "note"
)

// Areas group checks by what they look at.
const (
	AreaResponse   = "response"
	AreaRequest    = "request"
	AreaRun        = "run"
	AreaProbe      = "probe"
	AreaNotChecked = "not-checked"
	AreaSpec       = "spec"
)

// Check is one thing Clear Signal can report.
type Check struct {
	ID       string
	Title    string
	Severity Severity
	Area     string
}

var all = []Check{
	{"CS-RESP-001", "Undeclared response status", Failure, AreaResponse},
	{"CS-RESP-002", "Wrong response Content-Type", Failure, AreaResponse},
	{"CS-RESP-003", "Required response header not sent", Failure, AreaResponse},
	{"CS-RESP-004", "Response header of the wrong type", Failure, AreaResponse},
	{"CS-RESP-005", "Required body field missing", Failure, AreaResponse},
	{"CS-RESP-006", "Body field of the wrong type", Failure, AreaResponse},
	{"CS-RESP-007", "Value outside the declared constraints", Failure, AreaResponse},
	{"CS-RESP-008", "Body is not valid JSON, or is empty", Failure, AreaResponse},
	{"CS-RESP-009", "Body matches no oneOf / anyOf branch", Failure, AreaResponse},
	{"CS-RESP-010", "Undeclared extra property", Failure, AreaResponse},
	{"CS-RESP-011", "writeOnly property in a response", Failure, AreaResponse},
	{"CS-RESP-012", "Body media type that cannot be read", Note, AreaNotChecked},
	{"CS-RESP-013", "Response longer than the read limit", Failure, AreaResponse},
	{"CS-RESP-014", "Response slower than the limit", Failure, AreaResponse},
	{"CS-RESP-015", "Number beyond a 64-bit float", Failure, AreaResponse},

	{"CS-REQ-001", "Request parameter does not fit", Failure, AreaRequest},
	{"CS-REQ-002", "Request body does not fit", Failure, AreaRequest},

	{"CS-RUN-001", "Not sent: no value for a placeholder", Failure, AreaRun},
	{"CS-RUN-002", "Not sent: a dynamic value failed", Failure, AreaRun},
	{"CS-RUN-003", "No response", Failure, AreaRun},

	{"CS-PROBE-001", "Missing required field accepted", Failure, AreaProbe},
	{"CS-PROBE-002", "Wrong field type accepted", Failure, AreaProbe},
	{"CS-PROBE-003", "Missing required query parameter or header accepted", Failure, AreaProbe},
	{"CS-PROBE-004", "Missing credentials accepted", Failure, AreaProbe},
	{"CS-PROBE-006", "A probe got no answer", Failure, AreaProbe},

	{"CS-SPEC-001", "Operation without an operationId", Warning, AreaSpec},
	{"CS-SPEC-002", "Required property that is not declared", Failure, AreaSpec},
	{"CS-SPEC-003", "Example that breaks its own format", Warning, AreaSpec},

	{"CS-NC-001", "Path not in the spec", Note, AreaNotChecked},
	{"CS-NC-002", "No response content declared", Note, AreaNotChecked},
	{"CS-NC-003", "Binary or schema-less media type", Note, AreaNotChecked},
	{"CS-NC-004", "Spec validation turned off for the step", Note, AreaNotChecked},
	{"CS-NC-005", "Scenario has no spec", Note, AreaNotChecked},
	{"CS-NC-006", "The step did not complete", Note, AreaNotChecked},
	{"CS-NC-007", "Operation skipped by a filter or read-only", Note, AreaNotChecked},
	{"CS-NC-008", "Response without a Content-Type", Note, AreaNotChecked},
	{"CS-NC-009", "Declared media type without a decoder", Note, AreaNotChecked},
}

var (
	mu   sync.RWMutex
	byID = index()
)

func index() map[string]Check {
	m := make(map[string]Check, len(all))
	for _, c := range all {
		m[c.ID] = c
	}
	return m
}

// All returns every check, sorted by ID. The result is a copy.
func All() []Check {
	out := append([]Check(nil), all...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ByID finds a check by its ID.
func ByID(id string) (Check, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := byID[id]
	return c, ok
}

// SeverityOf is the default severity of a check. An ID that is not in the list (a violation that
// is not recognised has none) is a failure, so nothing that failed before can pass by accident.
func SeverityOf(id string) Severity {
	mu.RLock()
	defer mu.RUnlock()
	if c, ok := byID[id]; ok {
		return c.Severity
	}
	return Failure
}

// WithSeverity changes the severity of one check until the returned function is called. It is for
// tests and for a later per-project setting; the severities in the list are the defaults. An
// unknown ID is ignored.
func WithSeverity(id string, s Severity) (restore func()) {
	mu.Lock()
	defer mu.Unlock()
	old, ok := byID[id]
	if !ok {
		return func() {}
	}
	c := old
	c.Severity = s
	byID[id] = c
	return func() {
		mu.Lock()
		defer mu.Unlock()
		byID[id] = old
	}
}

// ParseFailOn reads the value of a --fail-on option for conformance runs: failure (the default
// when empty) or warning. A note never fails a run, so it is not accepted.
func ParseFailOn(s string) (Severity, error) {
	switch s {
	case "", string(Failure):
		return Failure, nil
	case string(Warning):
		return Warning, nil
	}
	return "", fmt.Errorf("--fail-on %q is not valid for conform: use failure (the default) or warning", s)
}
