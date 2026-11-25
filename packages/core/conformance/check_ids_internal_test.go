package conformance

import (
	"os"
	"regexp"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/checks"
)

// Every finding kind declared in diagnose.go either maps to a check in package checks or is the
// one kind that is deliberately not recognised. A new kind without a check fails here.
func TestEveryFindingKindMapsToACheck(t *testing.T) {
	src, err := os.ReadFile("diagnose.go")
	if err != nil {
		t.Fatal(err)
	}
	kinds := regexp.MustCompile(`(?m)^\tFinding\w+\s+= "([a-z-]+)"`).FindAllStringSubmatch(string(src), -1)
	if len(kinds) < 15 {
		t.Fatalf("found only %d finding kinds in diagnose.go; the pattern needs updating", len(kinds))
	}
	for _, m := range kinds {
		kind := m[1]
		if kind == FindingOther {
			if id := checkIDFor(Finding{Kind: kind}); id != "" {
				t.Errorf("%q must have no check ID, got %s", kind, id)
			}
			continue
		}
		id := checkIDFor(Finding{Kind: kind, Part: "request body"})
		if id == "" {
			t.Errorf("finding kind %q has no check ID", kind)
			continue
		}
		if _, ok := checks.ByID(id); !ok {
			t.Errorf("finding kind %q maps to %s, which is not in package checks", kind, id)
		}
	}
	for _, part := range []string{"request body", "request parameter"} {
		id := checkIDFor(Finding{Kind: FindingRequest, Part: part})
		if _, ok := checks.ByID(id); !ok {
			t.Errorf("request finding for %q maps to %q, which is not in package checks", part, id)
		}
	}
}

func TestDiagnoseSetsSeverityFromTheCatalog(t *testing.T) {
	for _, f := range Diagnose(OperationResult{Method: "GET", Path: "/x", Error: "dial tcp: connection refused"}) {
		if f.CheckID == "" || f.Severity != checks.SeverityOf(f.CheckID) || f.Severity == "" {
			t.Errorf("finding %+v: severity %q, check %q", f.Kind, f.Severity, f.CheckID)
		}
	}
	// a violation that is not recognised is a failure
	if got := checks.SeverityOf(checkIDFor(Finding{Kind: FindingOther})); got != checks.Failure {
		t.Errorf("unrecognised finding severity = %s, want failure", got)
	}
}
