package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"gopkg.in/yaml.v3"
)

// BaselineSchemaVersion is the version of the baseline file format.
const BaselineSchemaVersion = "1"

// Violation is one thing a conformance run found wrong, in a form that stays the same from run
// to run so it can be remembered in a baseline. Operation is the method and the path template
// from the spec ("GET /users/{id}"), never a concrete URL, so ids and hosts do not change it.
type Violation struct {
	Operation string `yaml:"operation" json:"operation"`
	// Kind is request (the generated request broke the spec), response (the response broke the
	// spec), error (the operation could not be executed) or probe (a negative probe was accepted).
	Kind string `yaml:"kind" json:"kind"`
	// Detail names the field and rule for request and response violations, and the probe for a
	// probe. It is empty for an execution error, whose text (a port, a timeout) varies per run.
	Detail string `yaml:"detail,omitempty" json:"detail,omitempty"`
}

// Key identifies the violation: two runs that find the same thing produce the same key.
func (v Violation) Key() string { return v.Operation + "\x00" + v.Kind + "\x00" + v.Detail }

func (v Violation) String() string {
	s := v.Operation + "  " + v.Kind
	if v.Detail != "" {
		s += "  " + v.Detail
	}
	return s
}

// Violations lists what a run found wrong, sorted and without duplicates. neg may be nil.
// Skipped operations and passing probes are not violations.
func Violations(r *ConformResult, neg *NegativeResult) []Violation {
	return ViolationsFor(r, neg, checks.Failure)
}

// ViolationsFor is Violations with the severity that counts: failure lists only findings that
// fail a run, warning also lists the warnings (what a run with --fail-on warning must accept).
func ViolationsFor(r *ConformResult, neg *NegativeResult, failOn checks.Severity) []Violation {
	seen := map[string]bool{}
	var out []Violation
	var secrets []string
	if r != nil {
		secrets = r.Secrets
	}
	add := func(v Violation) {
		// a baseline is committed to the repository, so a secret never reaches it
		for _, sec := range secrets {
			if sec != "" {
				v.Detail = strings.ReplaceAll(v.Detail, sec, "***")
			}
		}
		if !seen[v.Key()] {
			seen[v.Key()] = true
			out = append(out, v)
		}
	}
	if r != nil {
		for _, op := range r.Operations {
			if op.Skipped {
				continue
			}
			name := op.Method + " " + op.Path
			if op.Error != "" {
				add(Violation{Operation: name, Kind: "error"})
			}
			counts := func(f Finding) bool {
				return f.Severity == checks.Failure || f.Severity == checks.Warning && failOn == checks.Warning
			}
			for _, v := range op.RequestViolations {
				if counts(diagnoseRequest(op, v).withSeverity()) {
					add(Violation{Operation: name, Kind: "request", Detail: v.Error()})
				}
			}
			for _, v := range op.Violations {
				if counts(diagnoseViolation(op, v).withSeverity()) {
					add(Violation{Operation: name, Kind: "response", Detail: v.Error()})
				}
			}
		}
	}
	if neg != nil {
		for _, p := range neg.Probes {
			if p.Skipped || p.Error != "" || p.Passed() {
				continue
			}
			add(Violation{Operation: p.Method + " " + p.Path, Kind: "probe", Detail: string(p.Kind) + ": " + p.Description})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// Baseline is the saved list of violations a team has accepted for now.
type Baseline struct {
	SchemaVersion string      `yaml:"schemaVersion" json:"schemaVersion"`
	Violations    []Violation `yaml:"violations" json:"violations"`
}

// NewBaseline records violations. It carries no timestamp, so writing an unchanged baseline
// again produces an identical file.
func NewBaseline(violations []Violation) *Baseline {
	return &Baseline{SchemaVersion: BaselineSchemaVersion, Violations: append([]Violation{}, violations...)}
}

// ParseBaseline reads a baseline; the format follows the file extension (.json, otherwise YAML).
// Unknown fields and unsupported versions are errors, so a typo is never silently ignored.
func ParseBaseline(data []byte, filename string) (*Baseline, error) {
	var b Baseline
	if isJSONFile(filename) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&b); err != nil {
			return nil, fmt.Errorf("parsing baseline: %w", err)
		}
	} else {
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&b); err != nil {
			return nil, fmt.Errorf("parsing baseline: %w", err)
		}
	}
	if b.SchemaVersion != BaselineSchemaVersion {
		return nil, fmt.Errorf("baseline schemaVersion %q is not supported (this version of clrsig reads %q)", b.SchemaVersion, BaselineSchemaVersion)
	}
	for i, v := range b.Violations {
		if v.Operation == "" || v.Kind == "" {
			return nil, fmt.Errorf("baseline violation %d needs an operation and a kind", i+1)
		}
	}
	return &b, nil
}

// Marshal writes the baseline in the format of filename's extension.
func (b *Baseline) Marshal(filename string) ([]byte, error) {
	if isJSONFile(filename) {
		out, err := json.MarshalIndent(b, "", "  ")
		return append(out, '\n'), err
	}
	return yaml.Marshal(b)
}

// WriteBaselineFile writes the baseline atomically, creating parent folders.
func WriteBaselineFile(path string, b *Baseline) error {
	data, err := b.Marshal(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func isJSONFile(name string) bool { return strings.EqualFold(filepath.Ext(name), ".json") }

// BaselineDiff is the outcome of comparing a run with a baseline.
type BaselineDiff struct {
	Known []Violation // found now and already in the baseline: accepted for now
	New   []Violation // found now but not in the baseline: these fail the run
	Fixed []Violation // in the baseline but no longer found
}

// Compare splits the current violations against a baseline.
func Compare(current []Violation, b *Baseline) BaselineDiff {
	inBaseline := map[string]bool{}
	for _, v := range b.Violations {
		inBaseline[v.Key()] = true
	}
	now := map[string]bool{}
	var d BaselineDiff
	for _, v := range current {
		now[v.Key()] = true
		if inBaseline[v.Key()] {
			d.Known = append(d.Known, v)
		} else {
			d.New = append(d.New, v)
		}
	}
	for _, v := range b.Violations {
		if !now[v.Key()] {
			d.Fixed = append(d.Fixed, v)
		}
	}
	return d
}

// Text renders the comparison for a terminal or a CI log.
func (d BaselineDiff) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Baseline: %d known, %d new, %d fixed\n", len(d.Known), len(d.New), len(d.Fixed))
	for _, v := range d.New {
		fmt.Fprintf(&b, "  NEW    %s\n", v)
	}
	for _, v := range d.Fixed {
		fmt.Fprintf(&b, "  FIXED  %s\n", v)
	}
	if len(d.Fixed) > 0 {
		b.WriteString("Fixed violations stay in the baseline until it is rewritten with --write-baseline.\n")
	}
	return b.String()
}
