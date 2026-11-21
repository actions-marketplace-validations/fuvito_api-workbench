package coverage_test

import (
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

func makeReport(exercised []string, missed []string, opResults map[string]coverage.OperationResult) coverage.Report {
	return coverage.Report{
		Exercised:        exercised,
		Missed:           missed,
		OperationResults: opResults,
	}
}

func TestComputeQuality_NeverExercised(t *testing.T) {
	report := makeReport(
		[]string{"GET /users"},
		[]string{"DELETE /users/{id}", "POST /users"},
		nil,
	)
	q := coverage.ComputeQuality(report, nil, nil)
	if len(q.NeverExercised) != 2 {
		t.Errorf("NeverExercised: got %v, want 2 entries", q.NeverExercised)
	}
}

func TestComputeQuality_HappyPathOnly(t *testing.T) {
	report := makeReport(
		[]string{"GET /users", "POST /items"},
		nil,
		map[string]coverage.OperationResult{
			"GET /users":  {LastPassed: true, LastRunAt: time.Now(), StatusCodes: []int{200}},
			"POST /items": {LastPassed: true, LastRunAt: time.Now(), StatusCodes: []int{200, 404}},
		},
	)
	declared := map[string][]int{
		"GET /users":  {200, 400, 404},
		"POST /items": {200, 404},
	}
	q := coverage.ComputeQuality(report, declared, nil)

	// GET /users: only 200 seen, but 400 and 404 declared → happy-path only
	if len(q.HappyPathOnly) != 1 || q.HappyPathOnly[0] != "GET /users" {
		t.Errorf("HappyPathOnly: got %v, want [GET /users]", q.HappyPathOnly)
	}
	// POST /items: 404 seen → not happy-path only
	for _, op := range q.HappyPathOnly {
		if op == "POST /items" {
			t.Errorf("POST /items should not be happy-path-only (404 was observed)")
		}
	}
}

func TestComputeQuality_ErrorsNeverTriggered(t *testing.T) {
	report := makeReport(
		[]string{"PUT /users/{id}"},
		nil,
		map[string]coverage.OperationResult{
			"PUT /users/{id}": {StatusCodes: []int{200}},
		},
	)
	declared := map[string][]int{
		"PUT /users/{id}": {200, 400, 404, 422},
	}
	q := coverage.ComputeQuality(report, declared, nil)

	if len(q.ErrorsNeverTriggered) != 1 {
		t.Fatalf("ErrorsNeverTriggered: got %d entries, want 1", len(q.ErrorsNeverTriggered))
	}
	gap := q.ErrorsNeverTriggered[0]
	if gap.Operation != "PUT /users/{id}" {
		t.Errorf("gap operation: got %q, want PUT /users/{id}", gap.Operation)
	}
	wantMissing := []int{400, 404, 422}
	if len(gap.Missing) != len(wantMissing) {
		t.Errorf("missing codes: got %v, want %v", gap.Missing, wantMissing)
	}
	for i, c := range gap.Missing {
		if c != wantMissing[i] {
			t.Errorf("missing[%d]: got %d, want %d", i, c, wantMissing[i])
		}
	}
}

func TestComputeQuality_SecuredNeverAuthed(t *testing.T) {
	report := makeReport(
		[]string{"GET /protected", "DELETE /items/{id}"},
		nil,
		map[string]coverage.OperationResult{
			"GET /protected":     {HadAuth: false, StatusCodes: []int{200}},
			"DELETE /items/{id}": {HadAuth: true, StatusCodes: []int{204}},
		},
	)
	secured := map[string]bool{
		"GET /protected":     true,
		"DELETE /items/{id}": true,
	}
	q := coverage.ComputeQuality(report, nil, secured)

	if len(q.SecuredNeverAuthed) != 1 || q.SecuredNeverAuthed[0] != "GET /protected" {
		t.Errorf("SecuredNeverAuthed: got %v, want [GET /protected]", q.SecuredNeverAuthed)
	}
}

func TestComputeQuality_AllClean(t *testing.T) {
	report := makeReport(
		[]string{"POST /users"},
		nil,
		map[string]coverage.OperationResult{
			"POST /users": {HadAuth: true, StatusCodes: []int{201, 400, 422}},
		},
	)
	declared := map[string][]int{"POST /users": {201, 400, 422}}
	secured := map[string]bool{"POST /users": true}

	q := coverage.ComputeQuality(report, declared, secured)

	if len(q.NeverExercised) != 0 {
		t.Errorf("NeverExercised: want 0, got %v", q.NeverExercised)
	}
	if len(q.HappyPathOnly) != 0 {
		t.Errorf("HappyPathOnly: want 0, got %v", q.HappyPathOnly)
	}
	if len(q.SecuredNeverAuthed) != 0 {
		t.Errorf("SecuredNeverAuthed: want 0, got %v", q.SecuredNeverAuthed)
	}
	if len(q.ErrorsNeverTriggered) != 0 {
		t.Errorf("ErrorsNeverTriggered: want 0, got %v", q.ErrorsNeverTriggered)
	}
}

func TestComputeQuality_NeverExercisedSecuredNotReported(t *testing.T) {
	// A secured operation that was never exercised should NOT appear in
	// SecuredNeverAuthed — it already shows up in NeverExercised.
	report := makeReport(nil, []string{"DELETE /admin"}, nil)
	secured := map[string]bool{"DELETE /admin": true}

	q := coverage.ComputeQuality(report, nil, secured)

	if len(q.SecuredNeverAuthed) != 0 {
		t.Errorf("never-exercised secured op should not appear in SecuredNeverAuthed, got %v", q.SecuredNeverAuthed)
	}
	if len(q.NeverExercised) != 1 {
		t.Errorf("NeverExercised: want 1, got %v", q.NeverExercised)
	}
}

func TestMergeStatusCodes(t *testing.T) {
	a := []int{200, 404}
	b := []int{200, 422, 500}
	got := coverage.MergeStatusCodes(a, b)
	want := []int{200, 404, 422, 500}
	if len(got) != len(want) {
		t.Fatalf("MergeStatusCodes: got %v, want %v", got, want)
	}
	for i, v := range got {
		if v != want[i] {
			t.Errorf("[%d]: got %d, want %d", i, v, want[i])
		}
	}
}
