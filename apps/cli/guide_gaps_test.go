package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// What writing the detailed guide turned up. Each test pins today's behaviour, so that a fix has to
// come back to the catalog, the backlog and the guide's page.

func runQuietly(t *testing.T, args ...string) error {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root := newRootCmd()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root.Execute()
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Pins CS-GAP-021.
func TestValidate_AnUnknownHTTPMethodIsAccepted_KnownGap(t *testing.T) {
	file := writeTemp(t, "m.yaml", "schemaVersion: \"1\"\ntitle: m\nsteps:\n  - label: a\n    request: {method: GETT, url: \"http://localhost:1\"}\n    assertions:\n      - {type: status, operator: eq, expected: 200}\n")
	if err := runQuietly(t, "validate", file); err != nil {
		t.Errorf("validate reports it now (%v): update CS-GAP-021, validate.md, and close B-137", err)
	}
}

// Pins CS-GAP-022.
func TestRequest_AnUndefinedValueIsSentAsWritten_KnownGap(t *testing.T) {
	var query, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, auth = r.URL.Query().Get("t"), r.Header.Get("Authorization")
	}))
	t.Cleanup(srv.Close)
	file := writeTemp(t, "r.yaml", "schemaVersion: \"1\"\ntitle: r\nmethod: GET\nurl: \""+srv.URL+"/?t={{nowhere}}\"\nauth: {type: bearer, token: \"{{unset}}\"}\n")
	err := runQuietly(t, "request", file)
	if err != nil || query != "{{nowhere}}" || !strings.Contains(auth, "{{unset}}") {
		t.Errorf("no longer sent as written (%v, %q, %q): update CS-GAP-022, request.md, and close B-138", err, query, auth)
	}
}

// Pins CS-GAP-024: a coverage gate with nothing recorded passes.
func TestCoverageQuality_FailOnGapsPassesWithNoData_KnownGap(t *testing.T) {
	spec := writeTemp(t, "spec.yaml", generateSpec)
	if err := runQuietly(t, "coverage", "quality", spec, "--workspace", filepath.Join(t.TempDir(), "misspelled"), "--fail-on-gaps"); err != nil {
		t.Errorf("a gate with no data fails now (%v): update CS-GAP-024, coverage.md, and close B-141", err)
	}
}

// Pins CS-GAP-025: the placeholder an export leaves for a secret is sent as the secret.
func TestBundleRun_SendsTheExportPlaceholderAsTheSecret_KnownGap(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
	}))
	t.Cleanup(srv.Close)
	file := writeTemp(t, "b.yaml", "schemaVersion: \"1\"\ntitle: b\nenvironments:\n  - {file: e, name: e, variables: [{name: token, value: CHANGE_ME, secret: true}]}\nscenarios:\n  - file: a\n    title: a\n    steps:\n      - label: a\n        request: {schemaVersion: \"1\", method: GET, url: \""+srv.URL+"\", auth: {type: bearer, token: \"{{token}}\"}}\n        assertions: [{type: status, operator: eq, expected: 200}]\n")
	err := runQuietly(t, "bundle", "run", file)
	if err != nil || auth != "Bearer CHANGE_ME" {
		t.Errorf("the placeholder is no longer sent (%v, %q): update CS-GAP-025, bundle.md, and close B-142", err, auth)
	}
}
