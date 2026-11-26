package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// hitCounter counts every request that reaches it.
func hitCounter(t *testing.T) (string, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &calls
}

func TestConformList_SendsNothingAndSaysWhatIsNeeded(t *testing.T) {
	url, calls := hitCounter(t)
	spec := writeTempFile(t, "ph.yaml", placeholderSpec)

	out, err := exec(t, "conform", "--list", "--base-url", url, spec)
	if err != nil {
		t.Fatalf("a listing exits 0 even with missing values: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Nothing was sent.",
		"READY  GET     /health  needs nothing",
		"MISS   GET     /items/{id}  needs {{id}}  (not sent: no value for {{id}})",
		"2 operations  ·  1 ready  ·  1 would not be sent  ·  0 skipped",
		"Values used: id",
		"Missing: id.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	out, err = exec(t, "conform", "--list", "--base-url", url, "--var", "id=secret-value-7", spec)
	if err != nil || !strings.Contains(out, "Nothing is missing.") || strings.Contains(out, "secret-value-7") {
		t.Errorf("with a value nothing is missing, and values are never printed: %v\n%s", err, out)
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("%d request(s) reached the API", n)
	}
}

func TestConformList_JSON(t *testing.T) {
	url, calls := hitCounter(t)
	spec := writeTempFile(t, "ph.yaml", placeholderSpec)
	file := filepath.Join(t.TempDir(), "needs.json")
	if out, err := exec(t, "conform", "--list", "-f", "json", "-o", file, "--base-url", url, spec); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	var got struct {
		Operations []struct {
			Method, Path, SpecPath string
			Needs, Missing         []string
		}
		Values []struct {
			Name                string
			Missing, Credential bool
		}
	}
	if err := json.Unmarshal([]byte(readFile(t, file)), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Operations) != 2 || got.Operations[1].Path != "/items/{{id}}" || got.Operations[1].SpecPath != "/items/{id}" ||
		got.Operations[0].Needs == nil || got.Operations[0].Missing == nil {
		t.Errorf("operations: %+v", got.Operations)
	}
	if len(got.Values) != 1 || got.Values[0].Name != "id" || !got.Values[0].Missing || got.Values[0].Credential {
		t.Errorf("values: %+v", got.Values)
	}
	if calls.Load() != 0 {
		t.Error("the API was called")
	}
}

// A project file that turns on probes, a setup scenario, reports and a baseline for real runs does
// not make a listing send anything, write a report or fail on a file that is not there yet.
func TestConformList_IgnoresWhatTheProjectFileSetsForARun(t *testing.T) {
	url, calls := hitCounter(t)
	root := project(t, map[string]string{
		"openapi.yaml": placeholderSpec,
		"login.yaml":   "schemaVersion: \"1\"\ntitle: Login\nsteps:\n  - label: Login\n    request: {method: POST, url: \"{{baseUrl}}/login\"}\n",
		"clrsig.yaml": "schemaVersion: \"1\"\nspec: openapi.yaml\nbaseUrl: " + url + "\nreports: [junit=reports/r.xml]\n" +
			"conform:\n  negative: true\n  setup: login.yaml\n  baseline: not-recorded-yet.yaml\n  readOnly: true\n",
	})
	out, err := exec(t, "conform", "--list")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "Nothing was sent.") || !strings.Contains(out, "setup scenario") || strings.Contains(out, "Negative conformance") {
		t.Errorf("output:\n%s", out)
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("%d request(s) reached the API", n)
	}
	if _, err := os.Stat(filepath.Join(root, "reports", "r.xml")); err == nil {
		t.Error("a report was written")
	}
}

func TestConformList_RefusesFlagsItWouldIgnore(t *testing.T) {
	url, calls := hitCounter(t)
	spec := writeTempFile(t, "ph.yaml", placeholderSpec)
	for _, args := range [][]string{
		{"--negative"}, {"--setup", "login.yaml"}, {"--report", "junit=r.xml"}, {"--write-baseline", "b.yaml"},
		{"--baseline", "b.yaml"}, {"--coverage"}, {"--explain"}, {"--fail-fast"}, {"--fail-on", "warning"},
		{"--max-response-time", "5"}, {"-f", "junit"},
	} {
		out, err := exec(t, append([]string{"conform", "--list", "--base-url", url, spec}, args...)...)
		if err == nil || exitCode(err) == 1 || !strings.Contains(err.Error(), "--list") {
			t.Errorf("%v must be refused as a usage error: %v\n%s", args, err, out)
		}
	}
	if calls.Load() != 0 {
		t.Error("the API was called")
	}
}

func TestConformList_FiltersAndReadOnly(t *testing.T) {
	url, _ := hitCounter(t)
	spec := writeTempFile(t, "ph.yaml", placeholderSpec)

	out, err := exec(t, "conform", "--list", "--base-url", url, "--paths", "/health", spec)
	if err != nil || !strings.Contains(out, "SKIP   GET     /items/{id}") || !strings.Contains(out, "1 ready  ·  0 would not be sent  ·  1 skipped") || strings.Contains(out, "Missing:") {
		t.Errorf("a skipped operation's values are not missing: %v\n%s", err, out)
	}
	out, err = exec(t, "conform", "--list", "--base-url", url, "--paths", "/nowhere", spec)
	if err == nil || exitCode(err) == 1 || !strings.Contains(err.Error(), "no operation matches --paths /nowhere") {
		t.Errorf("a filter that matches nothing is an error, as for a run: %v\n%s", err, out)
	}
}

// init --spec lists what the spec needs as comments; a first conform still says "not sent", and
// uncommenting and filling the line sends the operation.
func TestInit_ListsWhatTheSpecNeeds(t *testing.T) {
	url, _ := hitCounter(t)
	root := project(t, map[string]string{"openapi.yaml": placeholderSpec})
	out, err := exec(t, "init", "--spec", "openapi.yaml", "--base-url", url)
	if err != nil || !strings.Contains(out, "The spec needs: id.") || !strings.Contains(out, "environments/local.yaml") {
		t.Fatalf("%v\n%s", err, out)
	}
	local := filepath.Join(root, "environments", "local.yaml")
	if c := readFile(t, local); !strings.Contains(c, "  #- name: id\n") || !strings.Contains(c, "# used by: GET /items/{id}") {
		t.Fatalf("local.yaml:\n%s", c)
	}
	out, err = exec(t, "conform")
	if exitCode(err) != 1 || !strings.Contains(out, "not sent: no value for {{id}}") {
		t.Fatalf("as generated, the operation is not sent: %v\n%s", err, out)
	}
	filled := strings.ReplaceAll(readFile(t, local), "  #- name: id\n  #  value: \"\"\n", "  - name: id\n    value: \"1\"\n")
	if err := os.WriteFile(local, []byte(filled), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err = exec(t, "conform"); err != nil || !strings.Contains(out, "PASSED") {
		t.Fatalf("filled in, it is sent: %v\n%s", err, out)
	}

	out, err = exec(t, "init", "--spec", "openapi.yaml")
	if err != nil || !strings.Contains(out, "was already there and was not changed") || readFile(t, local) != filled {
		t.Errorf("a second init changes nothing and says how to see the values: %v\n%s", err, out)
	}
}

func TestInit_DryRunAndURLSpecSayWhereTheValuesComeFrom(t *testing.T) {
	url, calls := hitCounter(t)
	root := project(t, map[string]string{"openapi.yaml": placeholderSpec})
	out, err := exec(t, "init", "--spec", "openapi.yaml", "--dry-run")
	if err != nil || !strings.Contains(out, "would be listed") {
		t.Errorf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "environments")); err == nil {
		t.Error("a dry run wrote something")
	}
	out, err = exec(t, "init", "--spec", url+"/openapi.yaml")
	if err != nil || !strings.Contains(out, "init does not fetch") || calls.Load() != 0 {
		t.Errorf("%v (%d calls)\n%s", err, calls.Load(), out)
	}
}
