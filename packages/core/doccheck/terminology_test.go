package doccheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var reBareTerm = regexp.MustCompile(`(?i)contract[- ]test`)

// CLAUDE.md section 1: the product's work is API Contract Validation and Spec Conformance Testing.
// The bare phrase suggests Pact, so it is allowed only on a line that says it is talking about Pact
// or consumer-driven testing.
func TestNoBareContractTestingTerm(t *testing.T) {
	check := func(rel, text string) {
		for i, line := range strings.Split(text, "\n") {
			// A file name is not the term.
			lower := strings.ReplaceAll(strings.ToLower(line), "contract-testing-roadmap.md", "")
			if reBareTerm.MatchString(lower) && !strings.Contains(lower, "pact") && !strings.Contains(lower, "consumer-driven") {
				t.Errorf("%s:%d uses the bare phrase; write \"API Contract Validation\" or \"Spec Conformance Testing\": %s", rel, i+1, strings.TrimSpace(line))
			}
		}
	}
	currentDocs(t, check)
	// Text the app itself serves.
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "apps", "desktop", "wails", "demo_server.go"))
	if err != nil {
		t.Fatal(err)
	}
	check("apps/desktop/wails/demo_server.go", string(data))
}
