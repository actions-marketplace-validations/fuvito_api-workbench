package conformance

import (
	"os"
	"path/filepath"
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
		if got[i].String() != want[i] {
			t.Errorf("%d: %q, want %q", i, got[i], want[i])
		}
	}

	other := sampleResult()
	other.Operations[3].Error = "i/o timeout after 10s" // a different message for the same failure
	if Violations(other, nil)[1].Key() != got[1].Key() {
		t.Error("an execution error is identified by its operation, not by its message")
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
	if len(got) != 1 || got[0].Kind != "probe" || got[0].Detail != "missing_required: drop field name" {
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
		if err != nil || len(parsed.Violations) != len(b.Violations) || parsed.Violations[0] != b.Violations[0] {
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
	if !strings.Contains(string(yamlData), `schemaVersion: "1"`) {
		t.Errorf("schemaVersion must be written:\n%s", yamlData)
	}
}

func TestParseBaselineRejectsBadInput(t *testing.T) {
	for name, tc := range map[string]struct{ file, text, want string }{
		"future version":  {"b.yaml", "schemaVersion: \"2\"\nviolations: []\n", "not supported"},
		"missing version": {"b.yaml", "violations: []\n", "not supported"},
		"unknown field":   {"b.yaml", "schemaVersion: \"1\"\nviolation: []\n", "violation"},
		"incomplete":      {"b.yaml", "schemaVersion: \"1\"\nviolations:\n  - {operation: GET /x}\n", "needs an operation and a kind"},
		"bad json":        {"b.json", "{\"schemaVersion\": \"1\", \"extra\": 1}", "extra"},
	} {
		if _, err := ParseBaseline([]byte(tc.text), tc.file); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want error containing %q", name, err, tc.want)
		}
	}
	if b, err := ParseBaseline([]byte("schemaVersion: \"1\"\nviolations: []\n"), "b.yaml"); err != nil || len(b.Violations) != 0 {
		t.Errorf("an empty baseline is valid: %v", err)
	}
}
