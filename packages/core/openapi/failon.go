package openapi

import "fmt"

// FailOn is how serious a spec change must be before a diff counts as failed.
type FailOn string

const (
	// FailOnBreaking fails only on breaking changes (level ERR). This is the default.
	FailOnBreaking FailOn = "breaking"
	// FailOnWarning also fails on potentially breaking changes (level WARN).
	FailOnWarning FailOn = "warning"
	// FailOnInfo fails on any change at all, including informational ones.
	FailOnInfo FailOn = "info"
)

// ParseFailOn reads a --fail-on value; the empty string means the default.
func ParseFailOn(s string) (FailOn, error) {
	switch FailOn(s) {
	case "", FailOnBreaking:
		return FailOnBreaking, nil
	case FailOnWarning:
		return FailOnWarning, nil
	case FailOnInfo:
		return FailOnInfo, nil
	}
	return "", fmt.Errorf("--fail-on must be one of: breaking, warning, info (got %q)", s)
}

// Fails reports whether this change makes a diff fail at the given threshold.
func (c DiffChange) Fails(on FailOn) bool {
	if c.Breaking {
		return true
	}
	switch on {
	case FailOnInfo:
		return true
	case FailOnWarning:
		return c.Level == "ERR" || c.Level == "WARN"
	}
	return false
}

// FailCount is the number of changes that fail at the given threshold.
func (r *DiffResult) FailCount(on FailOn) int {
	n := 0
	for _, c := range r.Changes {
		if c.Fails(on) {
			n++
		}
	}
	return n
}

// Fails reports whether the diff fails at the given threshold. At the default threshold this is
// the same as HasBreaking.
func (r *DiffResult) Fails(on FailOn) bool {
	if on == FailOnBreaking || on == "" {
		return r.HasBreaking
	}
	return r.FailCount(on) > 0
}
