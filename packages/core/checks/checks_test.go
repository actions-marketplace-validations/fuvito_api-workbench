package checks_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/checks"
)

var (
	reID     = regexp.MustCompile(`^CS-(RESP|REQ|RUN|PROBE|NC|SPEC)-\d{3}$`)
	reDocRow = regexp.MustCompile(`(?m)^\| (CS-[A-Z]+-\d{3}) \|`)
)

// docOnly lists the rows of the catalog that describe a behaviour, a coverage signal, a diff or a
// known gap rather than a finding in a run. A new row must be either in checks.All() or here,
// on purpose.
var docOnly = map[string]bool{
	"CS-REQ-003":   true, // what a generated request carries
	"CS-RUN-004":   true, // spec cannot be read (the run does not start)
	"CS-RUN-005":   true,
	"CS-RUN-006":   true, // new violation against a baseline
	"CS-RUN-007":   true,
	"CS-RUN-008":   true,
	"CS-RUN-009":   true,
	"CS-RUN-010":   true,
	"CS-RUN-011":   true, // what reports quote from a response
	"CS-PROBE-005": true, // probes follow the filters
}

func TestIDsAreUniqueAndWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range checks.All() {
		if !reID.MatchString(c.ID) {
			t.Errorf("%q does not follow the ID scheme", c.ID)
		}
		if seen[c.ID] {
			t.Errorf("%s is defined twice", c.ID)
		}
		seen[c.ID] = true
		if c.Title == "" {
			t.Errorf("%s has no title", c.ID)
		}
		switch c.Severity {
		case checks.Failure, checks.Warning, checks.Note:
		default:
			t.Errorf("%s has severity %q", c.ID, c.Severity)
		}
		if c.Area == "" {
			t.Errorf("%s has no area", c.ID)
		}
	}
}

func TestEveryNotCheckedRowIsANote(t *testing.T) {
	for _, c := range checks.All() {
		if c.Area == checks.AreaNotChecked && c.Severity != checks.Note {
			t.Errorf("%s is a not-checked outcome and must be a note, not %s", c.ID, c.Severity)
		}
	}
}

func TestSeverityOfUnknownIsFailure(t *testing.T) {
	if got := checks.SeverityOf(""); got != checks.Failure {
		t.Errorf("unknown ID: got %s, want failure", got)
	}
	if got := checks.SeverityOf("CS-NC-001"); got != checks.Note {
		t.Errorf("CS-NC-001: got %s, want note", got)
	}
}

func TestAllReturnsACopy(t *testing.T) {
	a := checks.All()
	a[0].Title = "changed"
	if checks.All()[0].Title == "changed" {
		t.Error("All() exposes the internal list")
	}
}

// The catalog document and the list must agree in both directions.
func TestListAndCatalogDocumentAgree(t *testing.T) {
	path := filepath.Join("..", "..", "..", "apps", "desktop", "wails", "help", "what-we-check.md")
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	inDoc := map[string]bool{}
	for _, m := range reDocRow.FindAllStringSubmatch(string(doc), -1) {
		inDoc[m[1]] = true
	}
	inList := map[string]bool{}
	for _, c := range checks.All() {
		inList[c.ID] = true
		if !inDoc[c.ID] {
			t.Errorf("%s is in checks.All() but has no row in what-we-check.md", c.ID)
		}
	}
	var missing []string
	for id := range inDoc {
		if inList[id] || docOnly[id] {
			continue
		}
		switch {
		case regexp.MustCompile(`^CS-(COV|DIFF|IMP|GAP)-`).MatchString(id):
			continue
		}
		missing = append(missing, id)
	}
	sort.Strings(missing)
	for _, id := range missing {
		t.Errorf("%s is in what-we-check.md but neither in checks.All() nor in the documentation-only list of this test", id)
	}
}
