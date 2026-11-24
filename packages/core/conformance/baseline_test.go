package conformance

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

func sampleResult() *ConformResult {
	return &ConformResult{
		Secrets: []string{"s3cret"},
		Operations: []OperationResult{
			{Method: "GET", Path: "/health", StatusCode: 200},
			{Method: "GET", Path: "/users/{id}", StatusCode: 200, Violations: []openapi.ValidationError{
				{Path: "/id", Message: "property is missing"}, {Path: "/id", Message: "property is missing"}, // duplicate
			}},
			{Method: "POST", Path: "/users", StatusCode: 201, RequestViolations: []openapi.ValidationError{{Message: "body is invalid"}}},
			{Method: "GET", Path: "/slow", Error: "dial tcp 127.0.0.1:54321: connection refused"},
			{Method: "DELETE", Path: "/users/{id}", Skipped: true},
			{Method: "GET", Path: "/leaky", StatusCode: 200, Violations: []openapi.ValidationError{{Message: "token s3cret is not allowed"}}},
		},
	}
}

func TestViolationsAreStableSortedAndDeduplicated(t *testing.T) {
	got := Violations(sampleResult(), nil)
	want := []string{
		"GET /leaky  response  token *** is not allowed",
		"GET /slow  error",
		"GET /users/{id}  response  /id: property is missing",
		"POST /users  request  body is invalid",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d violations: %v", len(got), got)
	}
	for i := range want {
		legacy := got[i].Operation + "  " + got[i].Kind
		if got[i].Detail != "" {
			legacy += "  " + got[i].Detail
		}
		if legacy != want[i] {
			t.Errorf("%d: %q, want %q", i, legacy, want[i])
		}
	}

	other := sampleResult()
	other.Operations[3].Error = "i/o timeout after 10s" // a different message for the same failure
	if Violations(other, nil)[1].Key() != got[1].Key() || Violations(other, nil)[1].LegacyKey() != got[1].LegacyKey() {
		t.Error("an execution error is identified by its operation, not by its message")
	}
	if got[1].Check != "CS-RUN-003" {
		t.Errorf("an execution error is check CS-RUN-003, got %q", got[1].Check)
	}
}

func TestNegativeProbesCountOnlyWhenAccepted(t *testing.T) {
	neg := &NegativeResult{Probes: []NegativeProbeResult{
		{Method: "POST", Path: "/users", Kind: "missing_required", Description: "drop field name", StatusCode: 201},
		{Method: "POST", Path: "/users", Kind: "wrong_type", Description: "name as number", StatusCode: 400},
		{Method: "POST", Path: "/users", Kind: "no_auth", Description: "no credentials", Error: "refused"},
		{Method: "POST", Path: "/users", Kind: "no_auth", Description: "skipped", Skipped: true},
	}}
	got := Violations(nil, neg)
	if len(got) != 1 || got[0].Kind != "probe" || got[0].Detail != "missing_required: drop field name" || got[0].Check != "other" {
		t.Errorf("only the accepted probe is a violation: %v", got)
	}
}

func TestCompareSplitsKnownNewAndFixed(t *testing.T) {
	base := NewBaseline(Violations(sampleResult(), nil))

	now := sampleResult()
	now.Operations[1].Violations = nil // fixed
	now.Operations = append(now.Operations, OperationResult{Method: "PUT", Path: "/new", StatusCode: 200,
		Violations: []openapi.ValidationError{{Message: "brand new problem"}}})
	d := Compare(Violations(now, nil), base)

	if len(d.Known) != 3 || len(d.New) != 1 || len(d.Fixed) != 1 {
		t.Fatalf("known=%d new=%d fixed=%d: %+v", len(d.Known), len(d.New), len(d.Fixed), d)
	}
	if d.New[0].Operation != "PUT /new" || d.Fixed[0].Operation != "GET /users/{id}" {
		t.Errorf("%+v", d)
	}
	text := d.Text()
	for _, want := range []string{"Baseline: 3 known, 1 new, 1 fixed", "NEW    PUT /new", "FIXED  GET /users/{id}", "--write-baseline"} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q:\n%s", want, text)
		}
	}
}

func TestNoChangeMeansNothingNew(t *testing.T) {
	cur := Violations(sampleResult(), nil)
	d := Compare(cur, NewBaseline(cur))
	if len(d.New) != 0 || len(d.Fixed) != 0 || len(d.Known) != len(cur) {
		t.Errorf("%+v", d)
	}
	if strings.Contains(d.Text(), "--write-baseline") {
		t.Error("the rewrite hint is only for fixed violations")
	}
}

func TestBaselineRoundTripsInYAMLAndJSONAndIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	b := NewBaseline(Violations(sampleResult(), nil))
	for _, name := range []string{"conform-baseline.yaml", "nested/dir/baseline.json"} {
		path := filepath.Join(dir, name)
		if err := WriteBaselineFile(path, b); err != nil {
			t.Fatal(err)
		}
		first, _ := os.ReadFile(path)
		if err := WriteBaselineFile(path, b); err != nil {
			t.Fatal(err)
		}
		second, _ := os.ReadFile(path)
		if string(first) != string(second) {
			t.Errorf("%s: writing the same baseline twice must give identical bytes", name)
		}
		parsed, err := ParseBaseline(first, path)
		if err != nil || len(parsed.Violations) != len(b.Violations) || parsed.Violations[0].Key() != b.Violations[0].Key() || parsed.Violations[0].Summary != b.Violations[0].Summary || parsed.Legacy {
			t.Errorf("%s: round trip: %v %+v", name, err, parsed)
		}
		if strings.Contains(string(first), "s3cret") {
			t.Errorf("%s: a secret leaked into the baseline", name)
		}
		if _, err := os.Stat(path + ".tmp"); err == nil {
			t.Errorf("%s: temp file left behind", name)
		}
	}
	yamlData, _ := os.ReadFile(filepath.Join(dir, "conform-baseline.yaml"))
	if !strings.Contains(string(yamlData), `schemaVersion: "2"`) {
		t.Errorf("schemaVersion must be written:\n%s", yamlData)
	}
}

func TestParseBaselineRejectsBadInput(t *testing.T) {
	for name, tc := range map[string]struct{ file, text, want string }{
		"future version":   {"b.yaml", "schemaVersion: \"3\"\nviolations: []\n", "not supported"},
		"missing version":  {"b.yaml", "violations: []\n", "not supported"},
		"unknown field":    {"b.yaml", "schemaVersion: \"2\"\nviolation: []\n", "violation"},
		"unknown v1 field": {"b.yaml", "schemaVersion: \"1\"\nviolation: []\n", "violation"},
		"incomplete v1":    {"b.yaml", "schemaVersion: \"1\"\nviolations:\n  - {operation: GET /x}\n", "needs an operation and a kind"},
		"incomplete":       {"b.yaml", "schemaVersion: \"2\"\nviolations:\n  - {operation: GET /x}\n", "needs an operation and a check"},
		"v1 shape in v2":   {"b.yaml", "schemaVersion: \"2\"\nviolations:\n  - {operation: GET /x, kind: response}\n", "kind"},
		"bad json":         {"b.json", "{\"schemaVersion\": \"2\", \"extra\": 1}", "extra"},
	} {
		if _, err := ParseBaseline([]byte(tc.text), tc.file); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want error containing %q", name, err, tc.want)
		}
	}
	for _, v := range []string{"1", "2"} {
		if b, err := ParseBaseline([]byte("schemaVersion: \""+v+"\"\nviolations: []\n"), "b.yaml"); err != nil || len(b.Violations) != 0 {
			t.Errorf("an empty version %s baseline is valid: %v", v, err)
		}
	}
}

// A realistic finding, so the check ID, the field and the summary come from the diagnosis.
func missingFieldResult(message string) *ConformResult {
	return &ConformResult{Operations: []OperationResult{{
		Method: "GET", Path: "/users/1", SpecPath: "/users/{id}", StatusCode: 200,
		Violations: []openapi.ValidationError{{Message: message}},
	}}}
}

const missingEmail = `Error at "/email": property "email" is missing`

func TestViolationIdentityIsTheCheckNotTheWording(t *testing.T) {
	v := Violations(missingFieldResult(missingEmail), nil)
	if len(v) != 1 {
		t.Fatalf("%v", v)
	}
	if v[0].Operation != "GET /users/1" || v[0].Check != "CS-RESP-005" || v[0].Where != "email" || v[0].Summary == "" {
		t.Errorf("identity: %+v", v[0])
	}

	// the same problem described with other words (a library upgrade, a longer schema dump)
	reworded := Violations(missingFieldResult(missingEmail+"\nSchema:\n  {\"type\": \"object\"}\n\nValue:\n  {}\n"), nil)
	if reworded[0].Key() != v[0].Key() {
		t.Errorf("rewording changed the identity:\n%q\n%q", v[0].Key(), reworded[0].Key())
	}
	if reworded[0].LegacyKey() == v[0].LegacyKey() {
		t.Error("the version 1 identity does depend on the wording; that is what this format fixes")
	}

	base := NewBaseline(v)
	d := Compare(reworded, base)
	if len(d.New) != 0 || len(d.Known) != 1 || len(d.Fixed) != 0 || d.Legacy {
		t.Errorf("a reworded known violation must stay known: %+v", d)
	}

	other := Violations(missingFieldResult(`Error at "/name": property "name" is missing`), nil)
	if d := Compare(other, base); len(d.New) != 1 || len(d.Fixed) != 1 {
		t.Errorf("another field is a new violation: %+v", d)
	}
}

func TestConstraintsKeepTheirRule(t *testing.T) {
	enum := Violations(missingFieldResult(`Error at "/status": value is not one of the allowed values ["a","b"]`), nil)
	min := Violations(missingFieldResult(`Error at "/status": number must be at least 1`), nil)
	if enum[0].Check != "CS-RESP-007" || min[0].Check != "CS-RESP-007" || enum[0].Key() == min[0].Key() {
		t.Errorf("two constraints on one field are two violations: %+v %+v", enum[0], min[0])
	}
}

func TestProbeViolationsCarryTheirCheck(t *testing.T) {
	neg := &NegativeResult{Probes: []NegativeProbeResult{
		{Method: "POST", Path: "/users", Kind: openapi.ProbeMissingAuth, Description: "no credentials", StatusCode: 201},
	}}
	got := Violations(nil, neg)
	if len(got) != 1 || got[0].Check != "CS-PROBE-004" || got[0].Rule != "no credentials" {
		t.Errorf("%+v", got)
	}
}

func TestVersion1BaselineIsComparedByWording(t *testing.T) {
	cur := Violations(missingFieldResult(missingEmail), nil)
	v1 := "schemaVersion: \"1\"\nviolations:\n  - operation: GET /users/1\n    kind: response\n    detail: " + quoteYAML(cur[0].Detail) + "\n"
	b, err := ParseBaseline([]byte(v1), "old.yaml")
	if err != nil || !b.Legacy || len(b.Violations) != 1 {
		t.Fatalf("%v %+v", err, b)
	}
	d := Compare(cur, b)
	if len(d.Known) != 1 || len(d.New) != 0 || !d.Legacy || !strings.Contains(d.Text(), "old format (schemaVersion 1)") {
		t.Errorf("a version 1 baseline still accepts what it recorded: %+v\n%s", d, d.Text())
	}
	changed := Violations(missingFieldResult(missingEmail+"\nValue:\n  {}\n"), nil)
	if d := Compare(changed, b); len(d.New) != 1 {
		t.Error("a version 1 baseline breaks on a wording change, as it always did")
	}
	if _, err := b.Marshal("old.yaml"); err == nil {
		t.Error("a version 1 baseline is not written back")
	}
	if upgraded := NewBaseline(cur); upgraded.Legacy || upgraded.SchemaVersion != "2" {
		t.Errorf("%+v", upgraded)
	}
}

func quoteYAML(s string) string {
	return strconv.Quote(s)
}
