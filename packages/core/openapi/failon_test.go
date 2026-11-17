package openapi

import "testing"

func TestParseFailOn(t *testing.T) {
	for in, want := range map[string]FailOn{"": FailOnBreaking, "breaking": FailOnBreaking, "warning": FailOnWarning, "info": FailOnInfo} {
		got, err := ParseFailOn(in)
		if err != nil || got != want {
			t.Errorf("ParseFailOn(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseFailOn("error"); err == nil {
		t.Error("an unknown level is an error")
	}
}

func TestDiffFailsAtEachThreshold(t *testing.T) {
	err := DiffChange{Level: "ERR", Breaking: true}
	warn := DiffChange{Level: "WARN"}
	info := DiffChange{Level: "INFO"}

	cases := []struct {
		name    string
		changes []DiffChange
		on      FailOn
		want    bool
	}{
		{"breaking fails everywhere", []DiffChange{err}, FailOnBreaking, true},
		{"breaking fails at warning", []DiffChange{err}, FailOnWarning, true},
		{"warn passes at breaking", []DiffChange{warn}, FailOnBreaking, false},
		{"warn fails at warning", []DiffChange{warn}, FailOnWarning, true},
		{"info passes at warning", []DiffChange{info}, FailOnWarning, false},
		{"info fails at info", []DiffChange{info}, FailOnInfo, true},
		{"no changes pass at info", nil, FailOnInfo, false},
		{"mixed counts only the failing", []DiffChange{info, warn, err}, FailOnWarning, true},
	}
	for _, c := range cases {
		r := &DiffResult{Changes: c.changes}
		for _, ch := range c.changes {
			if ch.Breaking {
				r.HasBreaking = true
			}
		}
		if got := r.Fails(c.on); got != c.want {
			t.Errorf("%s: Fails(%s) = %v, want %v", c.name, c.on, got, c.want)
		}
	}

	r := &DiffResult{Changes: []DiffChange{info, warn, err}, HasBreaking: true}
	if r.FailCount(FailOnBreaking) != 1 || r.FailCount(FailOnWarning) != 2 || r.FailCount(FailOnInfo) != 3 {
		t.Errorf("counts = %d/%d/%d", r.FailCount(FailOnBreaking), r.FailCount(FailOnWarning), r.FailCount(FailOnInfo))
	}
}
