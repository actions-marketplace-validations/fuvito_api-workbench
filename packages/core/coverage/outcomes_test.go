package coverage

import (
	"testing"
	"time"
)

func results(pass map[string]bool) map[string]OperationResult {
	out := map[string]OperationResult{}
	for op, p := range pass {
		out[op] = OperationResult{LastPassed: p, LastRunAt: time.Now()}
	}
	return out
}

func TestOutcomesSplitsCoveredByLastResult(t *testing.T) {
	r := Report{
		Total: 14, Covered: 10,
		Exercised:        []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
		OperationResults: results(map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true, "f": true, "g": true, "h": true, "i": true, "j": false}),
	}
	got := r.Outcomes()
	want := Outcomes{Total: 14, Covered: 10, NotCovered: 4, Passing: 9, Failing: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got.Covered+got.NotCovered != got.Total || got.Passing+got.Failing != got.Covered {
		t.Errorf("tiles do not add up: %+v", got)
	}
}

func TestOutcomesFullCoverageStillShowsFailures(t *testing.T) {
	r := Report{Total: 2, Covered: 2, Exercised: []string{"a", "b"}, OperationResults: results(map[string]bool{"a": true, "b": false})}
	got := r.Outcomes()
	if got.NotCovered != 0 || got.Failing != 1 || got.Passing != 1 {
		t.Errorf("%+v", got)
	}
}

func TestOutcomesCountsCoveredWithoutAResult(t *testing.T) {
	r := Report{Total: 3, Covered: 2, Exercised: []string{"a", "b"}}
	got := r.Outcomes()
	if got.NoResult != 2 || got.Passing != 0 || got.Failing != 0 || got.NotCovered != 1 {
		t.Errorf("%+v", got)
	}
}

func TestOutcomesNeverNegative(t *testing.T) {
	// a stored operation that is no longer in the spec must not push the counts below zero
	r := Report{Total: 1, Covered: 2, Exercised: []string{"a", "b"}}
	if got := r.Outcomes(); got.NotCovered != 0 || got.NoResult != 2 {
		t.Errorf("%+v", got)
	}
}

func TestShare(t *testing.T) {
	cases := []struct{ n, total, want int }{{1, 3, 33}, {2, 3, 67}, {10, 14, 71}, {0, 0, 0}, {5, 0, 0}, {1, 2, 50}, {1, 200, 1}}
	for _, c := range cases {
		if got := Share(c.n, c.total); got != c.want {
			t.Errorf("Share(%d,%d) = %d, want %d", c.n, c.total, got, c.want)
		}
	}
}
