package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/redact"
	"gopkg.in/yaml.v3"
)

// BaselineSchemaVersion is the version of the baseline file format that is written. Version 1
// (operation, kind, detail text) is still read, see ParseBaseline.
const BaselineSchemaVersion = "2"

// LegacyBaselineSchemaVersion is the older format, which identified a violation by the
// validator's wording. It is read but never written.
const LegacyBaselineSchemaVersion = "1"

// Violation is one thing a conformance run found wrong, in a form that stays the same from run
// to run so it can be remembered in a baseline. Operation is the method and the path template
// from the spec ("GET /users/{id}"), never a concrete URL, so ids and hosts do not change it.
//
// A violation is identified by Operation, Check, Where and Rule. Summary is for the reader and is
// not part of the identity, so rewording a message never turns a known violation into a new one.
// Kind and Detail are the identity of version 1 baselines (the validator's message text); a run
// still computes them so an old baseline can be read.
type Violation struct {
	Operation string
	// Check is the ID of the check (CS-RESP-005, see package checks), or "other" for a violation
	// this package does not recognise.
	Check string
	// Where is the field path, header name or status the violation is about; empty when there is none.
	Where string
	// Rule narrows a check that has several (which constraint broke; the probe's description). For
	// "other" it is the validator's own text, the one wording-sensitive identity left.
	Rule string
	// Summary says what is wrong in one line. It is written to the baseline for people to read.
	Summary string

	// Kind is request (the generated request broke the spec), response (the response broke the
	// spec), error (the operation could not be executed) or probe (a negative probe was accepted).
	// Detail is the validator's message, or "<probe kind>: <description>" for a probe. Both belong
	// to version 1 files.
	Kind   string
	Detail string
}

// Key identifies the violation: two runs that find the same thing produce the same key.
func (v Violation) Key() string {
	return v.Operation + "\x00" + v.Check + "\x00" + v.Where + "\x00" + strings.Join(problemParts(v.Rule), " | ")
}

// stableText is a validator message without the schema and value it quotes. Baselines written by
// earlier versions stored messages with them; messages no longer carry them, so both are compared
// without. A password the old message quoted is masked in the new one.
func stableText(s string) string {
	return redact.Credentials(strings.TrimSpace(reQuoted.ReplaceAllString(s, "")))
}

var (
	// the blocks are indented by two spaces; what follows them (" | " and the next problem) is not
	reQuoted = regexp.MustCompile(`\n(?:Schema:\n(?:  [^\n]*(?:\n|$))+\n?)?Value:\n(?:  [^\n]*(?:\n|$))+|\nSchema:\n(?:  [^\n]*(?:\n|$))+`)
	// where the next of several problems in one OpenAPI 3.1 message starts
	reNextProblem = regexp.MustCompile(` \| (?:validation failed due to: |error at "|at ')`)
	// what a request violation says before the validator's message
	reMessageContext = regexp.MustCompile(`^.*?does not fit the spec: `)
	// the library's own path before its sentence. It repeats segments, how often depends on the
	// other errors around it; the path inside the sentence (at '/a/b') is the right one
	reLibraryPath = regexp.MustCompile(`^error at "[^"]*": `)
)

const groupedBy = "validation failed due to: "

// problemParts lists the problems a message names, sorted. Up to 0.13.2 one OpenAPI 3.1 message
// could name several, in any order: 0.13.1 and earlier stored all of them, 0.13.2 only the first.
// Comparing problem by problem reads both, and does not depend on the order.
func problemParts(message string) []string {
	text := stableText(message)
	context := reMessageContext.FindString(text)
	text = text[len(context):]
	var raw []string
	start := 0
	for _, m := range reNextProblem.FindAllStringIndex(text, -1) {
		raw = append(raw, text[start:m[0]])
		start = m[0] + len(" | ")
	}
	raw = append(raw, text[start:])
	seen := map[string]bool{}
	var out []string
	for _, part := range raw {
		for strings.HasPrefix(part, groupedBy) {
			part = strings.TrimPrefix(part, groupedBy)
		}
		part = reLibraryPath.ReplaceAllString(part, "")
		if part = context + strings.TrimSpace(part); !seen[part] {
			seen[part] = true
			out = append(out, part)
		}
	}
	sort.Strings(out)
	return out
}

// identities are what a violation is matched by: where the validator's words are the identity, one
// for each problem they name; otherwise just the one.
func (v Violation) identities(legacy bool) []string {
	var base, text string
	switch {
	case legacy:
		base, text = v.Operation+"\x00"+v.Kind+"\x00", v.Detail
	case v.Check == "other":
		base, text = v.Operation+"\x00"+v.Check+"\x00"+v.Where+"\x00", v.Rule
	default:
		return []string{v.Key()}
	}
	var out []string
	for _, part := range problemParts(text) {
		out = append(out, base+part)
	}
	return out
}

// LegacyKey is the identity a version 1 baseline uses.
func (v Violation) LegacyKey() string {
	return v.Operation + "\x00" + v.Kind + "\x00" + stableText(v.Detail)
}

func (v Violation) String() string {
	if v.Check == "" {
		s := SpecStyle(v.Operation) + "  " + v.Kind
		if v.Detail != "" {
			s += "  " + v.Detail
		}
		return s
	}
	s := SpecStyle(v.Operation) + "  " + v.Check
	if v.Where != "" {
		s += "  " + v.Where
	}
	if v.Summary != "" {
		s += "  " + v.Summary
	}
	return s
}

var probeChecks = map[string]string{
	"drop_required_field":    "CS-PROBE-001",
	"wrong_field_type":       "CS-PROBE-002",
	"missing_required_param": "CS-PROBE-003",
	"missing_auth":           "CS-PROBE-004",
}

// ProbeCheckID is the check ID of a negative probe kind, or "" for a kind that has none.
func ProbeCheckID(kind string) string { return probeChecks[kind] }

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
	mask := func(s string) string {
		for _, sec := range secrets {
			if sec != "" {
				s = strings.ReplaceAll(s, sec, "***")
			}
		}
		return s
	}
	add := func(v Violation) {
		// a baseline is committed to the repository, so a secret never reaches it
		v.Detail, v.Where, v.Rule, v.Summary = mask(v.Detail), mask(v.Where), mask(v.Rule), mask(v.Summary)
		id := v.Key() + "\x01" + v.LegacyKey()
		if !seen[id] {
			seen[id] = true
			out = append(out, v)
		}
	}
	counts := func(f Finding) bool {
		return f.Severity == checks.Failure || f.Severity == checks.Warning && failOn == checks.Warning
	}
	fromFinding := func(op, kind, detail string, f Finding) Violation {
		v := Violation{Operation: op, Kind: kind, Detail: detail, Check: f.CheckID, Where: f.Where, Summary: f.Summary}
		if v.Check == "" {
			v.Check, v.Rule = "other", stableText(detail)
		} else if f.CheckID == "CS-RESP-007" {
			v.Rule = strings.ToLower(reConstraint.FindString(detail))
		}
		return v
	}
	if r != nil {
		for _, op := range r.Operations {
			if op.Skipped {
				continue
			}
			name := op.Method + " " + op.Path
			if op.Error != "" {
				add(Violation{Operation: name, Kind: "error", Check: "CS-RUN-003"})
			}
			for _, v := range op.RequestViolations {
				if f := diagnoseRequest(op, v).withSeverity(); counts(f) {
					add(fromFinding(name, "request", v.Error(), f))
				}
			}
			for _, v := range op.Violations {
				if f := diagnoseViolation(op, v).withSeverity(); counts(f) {
					add(fromFinding(name, "response", v.Error(), f))
				}
			}
		}
	}
	if neg != nil {
		for _, p := range neg.Probes {
			if p.Skipped || p.Error != "" || p.Passed() {
				continue
			}
			id := probeChecks[string(p.Kind)]
			if id == "" {
				id = "other"
			}
			add(Violation{Operation: p.Method + " " + p.Path, Kind: "probe", Detail: string(p.Kind) + ": " + p.Description,
				Check: id, Rule: p.Description, Summary: p.Description})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key() != out[j].Key() {
			return out[i].Key() < out[j].Key()
		}
		return out[i].LegacyKey() < out[j].LegacyKey()
	})
	return out
}

// Baseline is the saved list of violations a team has accepted for now. A baseline read from a
// version 1 file has Legacy set: its violations carry only Operation, Kind and Detail.
type Baseline struct {
	SchemaVersion string
	Violations    []Violation
	Legacy        bool
}

// baselineFile is what is written and read for version 2.
type baselineFile struct {
	SchemaVersion string          `yaml:"schemaVersion" json:"schemaVersion"`
	Violations    []violationFile `yaml:"violations" json:"violations"`
}

type violationFile struct {
	Operation string `yaml:"operation" json:"operation"`
	Check     string `yaml:"check" json:"check"`
	Where     string `yaml:"where,omitempty" json:"where,omitempty"`
	Rule      string `yaml:"rule,omitempty" json:"rule,omitempty"`
	Summary   string `yaml:"summary,omitempty" json:"summary,omitempty"`
}

// legacyBaselineFile is version 1.
type legacyBaselineFile struct {
	SchemaVersion string                `yaml:"schemaVersion" json:"schemaVersion"`
	Violations    []legacyViolationFile `yaml:"violations" json:"violations"`
}

type legacyViolationFile struct {
	Operation string `yaml:"operation" json:"operation"`
	Kind      string `yaml:"kind" json:"kind"`
	Detail    string `yaml:"detail,omitempty" json:"detail,omitempty"`
}

// NewBaseline records violations in the current format. It carries no timestamp, so writing an
// unchanged baseline again produces an identical file. Violations that share an identity are
// recorded once.
func NewBaseline(violations []Violation) *Baseline {
	seen := map[string]bool{}
	out := []Violation{}
	for _, v := range violations {
		if !seen[v.Key()] {
			seen[v.Key()] = true
			out = append(out, v)
		}
	}
	return &Baseline{SchemaVersion: BaselineSchemaVersion, Violations: out}
}

// ParseBaseline reads a baseline; the format follows the file extension (.json, otherwise YAML).
// Unknown fields and unsupported versions are errors, so a typo is never silently ignored.
// Version 1 files are read as they are (Legacy is set); a newer version than this program knows
// is refused.
func ParseBaseline(data []byte, filename string) (*Baseline, error) {
	var probe struct {
		SchemaVersion string `yaml:"schemaVersion" json:"schemaVersion"`
	}
	// the version decides the shape, so read it first and without strictness about the rest
	if isJSONFile(filename) {
		_ = json.Unmarshal(data, &probe)
	} else {
		_ = yaml.Unmarshal(data, &probe)
	}
	switch probe.SchemaVersion {
	case LegacyBaselineSchemaVersion:
		var f legacyBaselineFile
		if err := decodeStrict(data, filename, &f); err != nil {
			return nil, err
		}
		b := &Baseline{SchemaVersion: f.SchemaVersion, Legacy: true}
		for i, v := range f.Violations {
			if v.Operation == "" || v.Kind == "" {
				return nil, fmt.Errorf("baseline violation %d needs an operation and a kind", i+1)
			}
			b.Violations = append(b.Violations, Violation{Operation: v.Operation, Kind: v.Kind, Detail: redact.Credentials(v.Detail)})
		}
		return b, nil
	case BaselineSchemaVersion:
		var f baselineFile
		if err := decodeStrict(data, filename, &f); err != nil {
			return nil, err
		}
		b := &Baseline{SchemaVersion: f.SchemaVersion}
		for i, v := range f.Violations {
			if v.Operation == "" || v.Check == "" {
				return nil, fmt.Errorf("baseline violation %d needs an operation and a check", i+1)
			}
			b.Violations = append(b.Violations, Violation{Operation: v.Operation, Check: v.Check, Where: v.Where,
				// files written by earlier versions may hold a password; it is not printed again
				Rule: redact.Credentials(v.Rule), Summary: redact.Credentials(v.Summary)})
		}
		return b, nil
	}
	return nil, fmt.Errorf("baseline schemaVersion %q is not supported (this version of clrsig reads %q and %q; a newer file needs a newer clrsig)",
		probe.SchemaVersion, LegacyBaselineSchemaVersion, BaselineSchemaVersion)
}

func decodeStrict(data []byte, filename string, v any) error {
	if isJSONFile(filename) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(v); err != nil {
			return fmt.Errorf("parsing baseline: %w", err)
		}
		return nil
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("parsing baseline: %w", err)
	}
	return nil
}

// Marshal writes the baseline in the current format (version 2), in the format of filename's
// extension. A baseline read from a version 1 file cannot be written back: record a new one.
func (b *Baseline) Marshal(filename string) ([]byte, error) {
	if b.Legacy {
		return nil, fmt.Errorf("a version 1 baseline is not rewritten in place; record a new one with --write-baseline")
	}
	f := baselineFile{SchemaVersion: BaselineSchemaVersion, Violations: []violationFile{}}
	for _, v := range b.Violations {
		f.Violations = append(f.Violations, violationFile{Operation: v.Operation, Check: v.Check, Where: v.Where, Rule: v.Rule, Summary: v.Summary})
	}
	if isJSONFile(filename) {
		out, err := json.MarshalIndent(f, "", "  ")
		return append(out, '\n'), err
	}
	return yaml.Marshal(f)
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
	// Legacy is set when the baseline is a version 1 file: it was compared by the validator's
	// wording, which works as before but breaks when that wording changes.
	Legacy bool
}

// Compare splits the current violations against a baseline. A version 2 baseline is compared by
// check, a version 1 baseline by the wording it recorded. Every operation counts as checked.
func Compare(current []Violation, b *Baseline) BaselineDiff {
	return CompareChecked(current, b, nil)
}

// CheckedOperations lists the operations a run really sent, written as a baseline names them. A
// baseline entry for any other operation says nothing about this run: it is neither known nor
// fixed.
func CheckedOperations(r *ConformResult, neg *NegativeResult) map[string]bool {
	checked := map[string]bool{}
	if r != nil {
		for _, op := range r.Operations {
			if !op.Skipped && op.Blocked == "" {
				checked[op.Method+" "+op.Path] = true
			}
		}
	}
	if neg != nil {
		for _, p := range neg.Probes {
			if !p.Skipped {
				checked[p.Method+" "+p.Path] = true
			}
		}
	}
	return checked
}

// CompareChecked is Compare for a run that did not check every operation (a filter, read-only
// mode, a change set): a baseline entry counts as fixed only when its operation was checked. A nil
// checked means every operation was.
func CompareChecked(current []Violation, b *Baseline, checked map[string]bool) BaselineDiff {
	key := Violation.Key
	if b.Legacy {
		key = Violation.LegacyKey
	}
	inBaseline := map[string]bool{}
	for _, v := range b.Violations {
		for _, id := range v.identities(b.Legacy) {
			inBaseline[id] = true
		}
	}
	seen, now := map[string]bool{}, map[string]bool{}
	d := BaselineDiff{Legacy: b.Legacy}
	for _, v := range current {
		if seen[key(v)] {
			continue
		}
		seen[key(v)] = true
		// known only when every problem it names was recorded
		known := true
		for _, id := range v.identities(b.Legacy) {
			now[id] = true
			known = known && inBaseline[id]
		}
		if known {
			d.Known = append(d.Known, v)
		} else {
			d.New = append(d.New, v)
		}
	}
	for _, v := range b.Violations {
		gone := true
		for _, id := range v.identities(b.Legacy) {
			gone = gone && !now[id]
		}
		if gone && (checked == nil || checked[v.Operation]) {
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
	if d.Legacy {
		b.WriteString("This baseline uses the old format (schemaVersion 1), which identifies a violation by the validator's wording. It still works; rewrite it with --write-baseline to upgrade it.\n")
	}
	return b.String()
}
