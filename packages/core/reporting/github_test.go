package reporting

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

func githubFixture() *conformance.ConformResult {
	missing := `Error at "/email": property "email" is missing`
	return &conformance.ConformResult{
		SpecTitle: "Shop", SpecVersion: "1", BaseURL: "http://x",
		Secrets: []string{"sk-live-ABCDEFGH12345"},
		Operations: []conformance.OperationResult{
			{Method: "GET", Path: "/ok", SpecPath: "/ok", StatusCode: 200},
			{Method: "GET", Path: "/users/1", SpecPath: "/users/{id}", StatusCode: 200, Violations: []openapi.ValidationError{{Message: missing}}},
			{Method: "DELETE", Path: "/users/1", SpecPath: "/users/{id}", Skipped: true},
			{Method: "GET", Path: "/down", SpecPath: "/down", Error: "dial tcp: connection refused with sk-live-ABCDEFGH12345"},
		},
	}
}

func locator(method, path string) (string, int) {
	return "api/openapi.yaml", map[string]int{"GET /users/{id}": 42}[strings.ToUpper(method)+" "+path]
}

func TestConformGitHub_OneAnnotationPerFindingWithTheCheckAndTheLine(t *testing.T) {
	out := ConformGitHub(githubFixture(), nil, locator)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var errors, notices []string
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "::error "):
			errors = append(errors, l)
		case strings.HasPrefix(l, "::notice "):
			notices = append(notices, l)
		}
	}
	if len(errors) != 2 || len(notices) != 1 {
		t.Fatalf("a failing operation and a dead one are errors, plus one summary notice:\n%s", out)
	}
	if !strings.HasPrefix(errors[0], "::error file=api/openapi.yaml,line=42,title=CS-RESP-005 GET /users/{id}::") {
		t.Errorf("the check ID, the operation and the spec line are in the command: %s", errors[0])
	}
	for _, want := range []string{`"email" is missing`, "Why:", "Fix:"} {
		if !strings.Contains(errors[0], want) {
			t.Errorf("the message says what, why and how to fix; missing %q: %s", want, errors[0])
		}
	}
	if !strings.Contains(notices[0], "4 operations: 1 passed, 2 failed, 1 skipped.") {
		t.Errorf("summary: %s", notices[0])
	}
	if strings.Contains(out, "DELETE") {
		t.Errorf("a skipped operation is not annotated:\n%s", out)
	}
}

func TestConformGitHub_SecretsAndLineBreaksNeverEscape(t *testing.T) {
	out := ConformGitHub(githubFixture(), nil, locator)
	if strings.Contains(out, "sk-live-ABCDEFGH12345") {
		t.Errorf("a secret in an error text must be masked:\n%s", out)
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasPrefix(l, "::error ") && !strings.HasPrefix(l, "::notice ") && !strings.HasPrefix(l, "::warning ") {
			t.Errorf("every line is one workflow command, got %q", l)
		}
	}
}

func TestConformGitHub_Escaping(t *testing.T) {
	if got := escapeData("50%\nnext\r"); got != "50%25%0Anext%0D" {
		t.Errorf("data: %q", got)
	}
	if got := escapeProperty("a:b,c%d\n"); got != "a%3Ab%2Cc%25d%0A" {
		t.Errorf("property: %q", got)
	}
}

func TestConformGitHub_WarningsAndNotesAreNotErrors(t *testing.T) {
	t.Cleanup(checks.WithSeverity("CS-RESP-005", checks.Warning))
	out := ConformGitHub(githubFixture(), nil, nil)
	if !strings.Contains(out, "::warning title=CS-RESP-005 GET /users/{id}::") {
		t.Errorf("a warning-level check is a warning annotation; without a locator the annotation stays on the run:\n%s", out)
	}
	if strings.Contains(out, "file=") {
		t.Errorf("no file without a locator:\n%s", out)
	}
}

func TestConformGitHub_ProbesAndAPassingRun(t *testing.T) {
	r := &conformance.ConformResult{Operations: []conformance.OperationResult{{Method: "GET", Path: "/ok", StatusCode: 200}}}
	neg := &conformance.NegativeResult{Probes: []conformance.NegativeProbeResult{
		{Method: "POST", Path: "/users", Kind: openapi.ProbeMissingAuth, Description: "no credentials", StatusCode: 201},
		{Method: "POST", Path: "/users", Kind: openapi.ProbeDropRequiredField, Description: "drop name", StatusCode: 400},
	}}
	out := ConformGitHub(r, neg, nil)
	if strings.Count(out, "::error ") != 1 || !strings.Contains(out, "title=CS-PROBE-004 POST /users::The server accepted an invalid request (status 201): no credentials") {
		t.Errorf("only an accepted probe is an error:\n%s", out)
	}
	if clean := ConformGitHub(r, nil, nil); strings.Contains(clean, "::error") || !strings.Contains(clean, "::notice title=Clear Signal conform::1 operations: 1 passed, 0 failed, 0 skipped.") {
		t.Errorf("a passing run is one notice:\n%s", clean)
	}
}

func TestConformGitHub_ScopeLineIsInTheSummary(t *testing.T) {
	r := &conformance.ConformResult{
		Operations: []conformance.OperationResult{{Method: "GET", Path: "/ok", Skipped: true}},
		Scope:      &conformance.Scope{ChangedAgainst: "origin/main", Unchanged: 1},
	}
	if out := ConformGitHub(r, nil, nil); !strings.Contains(out, "Changed against origin/main: no operation changed") {
		t.Errorf("%s", out)
	}
}
