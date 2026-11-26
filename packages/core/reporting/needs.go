package reporting

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// The needs listing holds names only, never values, so there is nothing in it to redact.

// NeedsText lists, for clrsig conform --list, every operation with the values its request takes.
func NeedsText(l *conformance.NeedsList) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n", l.SpecTitle, l.SpecVersion)
	fmt.Fprintln(&b, "Nothing was sent. The values each operation needs:")
	fmt.Fprintln(&b)

	ready, blocked, skipped := 0, 0, 0
	for _, op := range l.Operations {
		status, note := "READY", ""
		switch {
		case op.Skipped:
			status = "SKIP"
			skipped++
		case op.Blocked != "":
			status = "MISS"
			blocked++
			note = "  (" + op.Blocked + ")"
		default:
			ready++
		}
		needs := "nothing"
		if len(op.Needs) > 0 {
			needs = placeholders(op.Needs)
		}
		fmt.Fprintf(&b, "  %-5s  %-7s %s  needs %s%s\n", status, op.Method, op.SpecPath, needs, note)
	}

	fmt.Fprintf(&b, "\n%d operations  ·  %d ready  ·  %d would not be sent  ·  %d skipped\n", len(l.Operations), ready, blocked, skipped)
	needs, missing := l.Names()
	if len(needs) > 0 {
		fmt.Fprintf(&b, "Values used: %s\n", strings.Join(needs, ", "))
	}
	switch {
	case len(missing) > 0:
		fmt.Fprintf(&b, "Missing: %s. Add them to the environment, or pass them as overrides (name=value).\n", strings.Join(missing, ", "))
	case blocked > 0:
		fmt.Fprintln(&b, "A dynamic value failed (see above).")
	case ready > 0:
		fmt.Fprintln(&b, "Nothing is missing.")
	}
	return b.String()
}

func placeholders(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "{{" + n + "}}"
	}
	return strings.Join(out, ", ")
}

// JSONNeeds is the JSON form of clrsig conform --list.
type JSONNeeds struct {
	SpecTitle   string            `json:"specTitle"`
	SpecVersion string            `json:"specVersion"`
	Operations  []JSONNeedsOp     `json:"operations"`
	Values      []JSONNeededValue `json:"values"`
}

// JSONNeedsOp is one operation; path and specPath are as in a conformance report.
type JSONNeedsOp struct {
	OperationID string   `json:"operationId,omitempty"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	SpecPath    string   `json:"specPath"`
	Skipped     bool     `json:"skipped,omitempty"`
	Needs       []string `json:"needs"`
	Missing     []string `json:"missing"`
	Blocked     string   `json:"blocked,omitempty"`
}

// JSONNeededValue is one value the operations of the run use.
type JSONNeededValue struct {
	Name       string `json:"name"`
	Missing    bool   `json:"missing"`
	Credential bool   `json:"credential"`
}

// IsCredential reports whether a needed value should be treated as a secret: it authenticates a
// request somewhere, or its name says it is one.
func IsCredential(l *conformance.NeedsList, name string) bool {
	for _, op := range l.Operations {
		if slices.Contains(op.Credentials, name) {
			return true
		}
	}
	return IsSecretName(name)
}

// NeedsJSON returns the listing as JSON.
func NeedsJSON(l *conformance.NeedsList) ([]byte, error) {
	list := func(s []string) []string {
		if s == nil {
			return []string{}
		}
		return s
	}
	out := JSONNeeds{SpecTitle: l.SpecTitle, SpecVersion: l.SpecVersion, Operations: []JSONNeedsOp{}, Values: []JSONNeededValue{}}
	for _, op := range l.Operations {
		out.Operations = append(out.Operations, JSONNeedsOp{
			OperationID: op.OperationID, Method: op.Method, Path: op.Path, SpecPath: op.SpecPath, Skipped: op.Skipped,
			Needs: list(op.Needs), Missing: list(op.Missing), Blocked: op.Blocked,
		})
	}
	needs, missing := l.Names()
	for _, n := range needs {
		out.Values = append(out.Values, JSONNeededValue{Name: n, Missing: slices.Contains(missing, n), Credential: IsCredential(l, n)})
	}
	return json.MarshalIndent(out, "", "  ")
}
