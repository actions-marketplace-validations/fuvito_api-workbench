package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

const qualitySpec = `openapi: "3.0.3"
info: {title: quality-api, version: "3.0"}
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
paths:
  /a:
    get:
      operationId: getA
      responses:
        "200": {description: ok}
        "404": {description: missing}
  /b:
    get:
      operationId: getB
      security:
        - key: []
      responses:
        "200": {description: ok}
  /c:
    get:
      operationId: getC
      responses:
        "200": {description: ok}
`

func exitCode(err error) int {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return -1
}

func TestCoverageQuality_Gaps(t *testing.T) {
	ws := t.TempDir()
	spec := writeTempFile(t, "api.yaml", qualitySpec)
	err := coverage.Save(ws, coverage.StoredCoverage{
		SpecFile: spec, TotalOps: 3, TotalRuns: 2, PassedRuns: 2,
		Exercised: []string{"GET /a", "GET /b"},
		OperationResults: map[string]coverage.OperationResult{
			"GET /a": {LastPassed: true, StatusCodes: []int{200}},
			"GET /b": {LastPassed: true, StatusCodes: []int{200}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := execFull(t, "coverage", "quality", "--workspace", ws, spec)
	if exitCode(err) != 1 {
		t.Fatalf("gaps should yield exit code 1, got err=%v", err)
	}
	for _, want := range []string{
		"quality-api", "2 run(s)", "NEVER EXERCISED (1)", "GET /c",
		"HAPPY PATH ONLY", "GET /a", "404",
		"SECURED", "GET /b",
		"UNTESTED ERROR PATHS", "gap(s) found",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCoverageQuality_JSON(t *testing.T) {
	ws := t.TempDir()
	spec := writeTempFile(t, "api.yaml", qualitySpec)
	coverage.Save(ws, coverage.StoredCoverage{
		SpecFile: spec, TotalOps: 3, TotalRuns: 1, Exercised: []string{"GET /a"},
		OperationResults: map[string]coverage.OperationResult{"GET /a": {StatusCodes: []int{200}}},
	})

	out, err := execFull(t, "coverage", "quality", "-w", ws, "-f", "json", spec)
	if exitCode(err) != 1 {
		t.Fatalf("expected exit 1, got %v", err)
	}
	var got struct {
		SpecTitle            string
		HasGaps              bool
		NeverExercised       []string
		ErrorsNeverTriggered []struct {
			Operation string
			Missing   []int
		}
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if got.SpecTitle != "quality-api" || !got.HasGaps || len(got.NeverExercised) != 2 {
		t.Errorf("unexpected json: %+v", got)
	}
	if len(got.ErrorsNeverTriggered) != 1 || got.ErrorsNeverTriggered[0].Missing[0] != 404 {
		t.Errorf("error gaps: %+v", got.ErrorsNeverTriggered)
	}
}

func TestCoverageQuality_NoGaps(t *testing.T) {
	ws := t.TempDir()
	spec := writeTempFile(t, "api.yaml", qualitySpec)
	coverage.Save(ws, coverage.StoredCoverage{
		SpecFile: spec, TotalOps: 3, TotalRuns: 1,
		Exercised: []string{"GET /a", "GET /b", "GET /c"},
		OperationResults: map[string]coverage.OperationResult{
			"GET /a": {StatusCodes: []int{200, 404}},
			"GET /b": {StatusCodes: []int{200}, HadAuth: true},
			"GET /c": {StatusCodes: []int{200}},
		},
	})
	out, err := execFull(t, "coverage", "quality", "-w", ws, spec)
	if err != nil {
		t.Fatalf("no gaps should succeed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No gaps found") {
		t.Errorf("output: %s", out)
	}
}

func TestCoverageQuality_Errors(t *testing.T) {
	ws := t.TempDir()
	bad := writeTempFile(t, "bad.yaml", "openapi: [")
	if _, err := execFull(t, "coverage", "quality", "-w", ws, bad); err == nil || !strings.Contains(err.Error(), "parsing spec") {
		t.Errorf("expected parse error, got %v", err)
	}
	if _, err := execFull(t, "coverage", "quality", "-w", ws, "/no/such/spec.yaml"); err == nil {
		t.Error("missing spec should fail")
	}
}
