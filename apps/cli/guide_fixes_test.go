package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

// What writing the detailed guide turned up (B-137 to B-147), as fixed.

func execRoot(root *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&out)
	err := root.Execute()
	return out.String(), err
}

func runQuietly(t *testing.T, args ...string) error {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_, err := execRoot(newRootCmd(), args...)
	return err
}

// runGuide runs the whole command tree with its own home folder, so that no setting of the person
// running the tests is read or written.
func runGuide(t *testing.T, args ...string) (string, int) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	out, err := execRoot(newRootCmd(), args...)
	code := exitCodeOf(err)
	if code == -1 { // an error that main prints and exits 2 for
		return out + err.Error(), 2
	}
	return out, code
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func scenarioWith(step string) string {
	return "schemaVersion: \"1\"\ntitle: m\nsteps:\n  - label: a\n" + step + "    assertions:\n      - {type: status, operator: eq, expected: 200}\n"
}

func TestValidate_ReportsAnUnknownHTTPMethod(t *testing.T) {
	inline := writeTemp(t, "m.yaml", scenarioWith("    request: {method: GETT, url: \"http://localhost:1\"}\n"))
	out, code := runGuide(t, "validate", inline)
	if code != 1 || !strings.Contains(out, `unknown HTTP method "GETT"`) {
		t.Errorf("in the step: exit %d\n%s", code, out)
	}

	referenced := writeTemp(t, "s.yaml", scenarioWith("    requestRef: r.yaml\n"))
	request := "schemaVersion: \"1\"\ntitle: r\nmethod: FETCH\nurl: http://localhost:1\n"
	if err := os.WriteFile(filepath.Join(filepath.Dir(referenced), "r.yaml"), []byte(request), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code = runGuide(t, "validate", referenced); code != 1 || !strings.Contains(out, `unknown HTTP method "FETCH"`) {
		t.Errorf("in a referenced file: exit %d\n%s", code, out)
	}

	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE", "get"} {
		file := writeTemp(t, "ok.yaml", scenarioWith("    request: {method: "+m+", url: \"http://localhost:1\"}\n"))
		if out, code := runGuide(t, "validate", file); code != 0 {
			t.Errorf("%s: exit %d\n%s", m, code, out)
		}
	}
}

func TestValidate_AFileThatCannotBeReadExits2(t *testing.T) {
	good := writeTemp(t, "g.yaml", scenarioWith("    request: {method: GET, url: \"http://localhost:1\"}\n"))
	if out, code := runGuide(t, "validate", good, filepath.Join(t.TempDir(), "missing.yaml")); code != 2 {
		t.Errorf("a missing file: exit %d\n%s", code, out)
	}
	// a file that was read and is wrong keeps 1
	if out, code := runGuide(t, "validate", writeTemp(t, "bad.yaml", "steps: [")); code != 1 {
		t.Errorf("a file that does not parse: exit %d\n%s", code, out)
	}
}

func TestRequest_IsNotSentWithAValueNothingDefines(t *testing.T) {
	var sent int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sent++ }))
	t.Cleanup(srv.Close)
	for name, part := range map[string]string{
		"query":  "url: \"" + srv.URL + "/?t={{nowhere}}\"\n",
		"header": "url: \"" + srv.URL + "\"\nheaders: [{name: X-T, value: \"{{nowhere}}\"}]\n",
		"token":  "url: \"" + srv.URL + "\"\nauth: {type: bearer, token: \"{{nowhere}}\"}\n",
		"body":   "url: \"" + srv.URL + "\"\nbody: {type: text, content: \"{{nowhere}}\"}\n",
	} {
		file := writeTemp(t, "r.yaml", "schemaVersion: \"1\"\ntitle: r\nmethod: POST\n"+part)
		for _, format := range []string{"text", "json"} {
			out, code := runGuide(t, "request", file, "--format", format)
			if code != 1 || !strings.Contains(out, "not sent: no value for {{nowhere}}") {
				t.Errorf("%s, %s: exit %d\n%s", name, format, code, out)
			}
		}
	}
	if sent != 0 {
		t.Fatalf("%d request(s) reached the server", sent)
	}

	file := writeTemp(t, "r.yaml", "schemaVersion: \"1\"\ntitle: r\nmethod: GET\nurl: \""+srv.URL+"/?t={{here}}\"\n")
	env := writeTemp(t, "e.yaml", "schemaVersion: \"1\"\nname: e\nvariables: [{name: here, value: x}]\n")
	if out, code := runGuide(t, "request", file, "--env", env); code != 0 || sent != 1 {
		t.Errorf("a defined value: exit %d, %d sent\n%s", code, sent, out)
	}
}

func TestCoverageQuality_AGateWithNoDataFails(t *testing.T) {
	spec := writeTemp(t, "spec.yaml", generateSpec)
	empty := t.TempDir()
	for _, format := range []string{"text", "json", "csv", "markdown", "html"} {
		out, code := runGuide(t, "coverage", "quality", spec, "-w", empty, "-f", format, "--fail-on-gaps")
		if code != 1 || !strings.Contains(out, "cannot pass when no coverage was recorded") {
			t.Errorf("%s with the flag: exit %d\n%s", format, code, out)
		}
		if out, code := runGuide(t, "coverage", "quality", spec, "-w", empty, "-f", format); code != 0 {
			t.Errorf("%s without the flag: exit %d\n%s", format, code, out)
		}
	}

	out, _ := runGuide(t, "coverage", "quality", spec, "-w", empty, "-f", "json")
	var doc struct {
		TotalRuns *int `json:"totalRuns"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || doc.TotalRuns == nil || *doc.TotalRuns != 0 {
		t.Errorf("JSON was asked for (%v):\n%s", err, out)
	}
	if out, _ := runGuide(t, "coverage", "list", "-w", empty, "-f", "json"); strings.TrimSpace(out) != "[]" {
		t.Errorf("an empty list as JSON:\n%s", out)
	}
}

func TestCoverage_AMisspelledWorkspaceIsAnError(t *testing.T) {
	spec := writeTemp(t, "spec.yaml", generateSpec)
	missing := filepath.Join(t.TempDir(), "misspelled")
	for _, args := range [][]string{
		{"coverage", "quality", spec, "-w", missing, "--fail-on-gaps"},
		{"coverage", "quality", spec, "-w", missing},
		{"coverage", "list", "-w", missing},
		{"coverage", "reset", spec, "-w", missing},
		{"coverage", "list", "-w", spec}, // a file is not a workspace
	} {
		if out, code := runGuide(t, args...); code != 2 || !strings.Contains(out, "is not a folder") {
			t.Errorf("%v: exit %d\n%s", args[1:], code, out)
		}
	}
}

func recordedWorkspace(t *testing.T) (ws, spec string) {
	t.Helper()
	ws, spec = t.TempDir(), writeTemp(t, "spec.yaml", generateSpec)
	if err := coverage.Save(ws, coverage.StoredCoverage{
		SpecFile: spec, Exercised: []string{"GET /a"}, TotalOps: 2, TotalRuns: 1, PassedRuns: 1, LastRunAt: time.Now(),
		OperationResults: map[string]coverage.OperationResult{"GET /a": {LastPassed: true, StatusCodes: []int{200}}},
	}); err != nil {
		t.Fatal(err)
	}
	return ws, spec
}

func TestCoverageQuality_GapsFailOnlyWithTheFlagInEveryFormat(t *testing.T) {
	ws, spec := recordedWorkspace(t)
	for _, format := range []string{"text", "json", "csv", "markdown", "html"} {
		if out, code := runGuide(t, "coverage", "quality", spec, "-w", ws, "-f", format); code != 0 {
			t.Errorf("%s without the flag: exit %d\n%s", format, code, out)
		}
		if out, code := runGuide(t, "coverage", "quality", spec, "-w", ws, "-f", format, "--fail-on-gaps"); code != 1 {
			t.Errorf("%s with the flag: exit %d\n%s", format, code, out)
		}
	}
}

func TestCoverage_OutputIsWrittenInEveryFormat(t *testing.T) {
	ws, spec := recordedWorkspace(t)
	for _, format := range []string{"text", "json", "csv", "markdown", "html"} {
		for name, args := range map[string][]string{"quality": {"coverage", "quality", spec}, "list": {"coverage", "list"}} {
			file := filepath.Join(t.TempDir(), "reports", "out")
			out, code := runGuide(t, append(args, "-w", ws, "-f", format, "-o", file)...)
			data, err := os.ReadFile(file)
			if code != 0 || err != nil || len(data) == 0 {
				t.Errorf("%s %s: exit %d, %v, %d bytes\n%s", name, format, code, err, len(data), out)
			}
		}
	}
}

func TestCoverageReset_SaysWhenThereWasNothingToReset(t *testing.T) {
	ws, spec := recordedWorkspace(t)
	if out, code := runGuide(t, "coverage", "reset", "other.yaml", "-w", ws); code != 0 || !strings.Contains(out, "nothing was reset") {
		t.Errorf("another spec: exit %d\n%s", code, out)
	}
	if out, code := runGuide(t, "coverage", "reset", spec, "-w", ws); code != 0 || !strings.Contains(out, "Coverage reset for") || coverage.Has(ws, spec) {
		t.Errorf("recorded: exit %d\n%s", code, out)
	}
	if out, _ := runGuide(t, "coverage", "reset", spec, "-w", ws); !strings.Contains(out, "nothing was reset") {
		t.Errorf("a second time:\n%s", out)
	}
}

func TestBundleRun_DoesNotSendTheExportPlaceholder(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("Authorization"))
	}))
	t.Cleanup(srv.Close)
	file := writeTemp(t, "b.yaml", "schemaVersion: \"1\"\ntitle: b\nenvironments:\n  - {file: e, name: e, variables: [{name: token, value: CHANGE_ME, secret: true}]}\nscenarios:\n  - file: a\n    title: a\n    steps:\n      - label: a\n        request: {schemaVersion: \"1\", method: GET, url: \""+srv.URL+"\", auth: {type: bearer, token: \"{{token}}\"}}\n        assertions: [{type: status, operator: eq, expected: 200}]\n")

	out, code := runGuide(t, "bundle", "run", file)
	if code != 1 || len(got) != 0 {
		t.Fatalf("exit %d, sent %v\n%s", code, got, out)
	}
	for _, want := range []string{`"token" still holds the CHANGE_ME`, "--var token=", "not sent: no value for {{token}}"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}

	if out, code := runGuide(t, "bundle", "run", file, "--var", "token=real"); code != 0 || len(got) != 1 || got[0] != "Bearer real" {
		t.Errorf("with the value given: exit %d, sent %v\n%s", code, got, out)
	}
}

func TestTelemetry_SaysEverythingThatIsSent(t *testing.T) {
	for _, command := range []string{"enable", "status"} {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		if _, err := execRoot(newRootCmd(), "telemetry", "enable"); err != nil {
			t.Fatal(err)
		}
		out, err := execRoot(newRootCmd(), "telemetry", command)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"random identifier", "version", "operating system", "processor type", "the time", "IP address"} {
			if !strings.Contains(out, want) {
				t.Errorf("telemetry %s does not name %q:\n%s", command, want, out)
			}
		}
		if strings.Contains(strings.ToLower(out), "anonymous") {
			t.Errorf("telemetry %s calls it anonymous:\n%s", command, out)
		}
	}
}
