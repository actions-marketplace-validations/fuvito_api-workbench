package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
)

// dynServer echoes the query parameter "e" as {"echo": ...} and counts requests.
func dynServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"echo":%q}`, r.URL.Query().Get("e"))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

const orderScenarioYAML = `schemaVersion: "1"
title: %s
variables:
  - name: email
    value: "user-{{$randomString 10}}@example.com"
  - name: due
    value: "{{$date +7d}}"
steps:
  - label: create
    request: {method: GET, url: "{{base}}/create?e={{email}}&due={{due}}"}
  - label: fetch
    request: {method: GET, url: "{{base}}/fetch?e={{email}}"}
    assertions:
      - {type: jsonpath, path: $.echo, operator: eq, expected: "{{email}}"}
`

func orderScenarioFile(t *testing.T, title string) string {
	t.Helper()
	return writeTempFile(t, strings.ReplaceAll(strings.ToLower(title), " ", "-")+".yaml", fmt.Sprintf(orderScenarioYAML, title))
}

var (
	seedRe  = regexp.MustCompile(`seed: (\d+)`)
	emailRe = regexp.MustCompile(`var email = (\S+)`)
)

func find(t *testing.T, re *regexp.Regexp, text string) string {
	t.Helper()
	m := re.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%v not found in:\n%s", re, text)
	}
	return m[1]
}

func TestRun_ScenarioVariablesAndAssertionsUseTheSameGeneratedValue(t *testing.T) {
	srv, n := dynServer(t)
	out, err := exec(t, "run", "--var", "base="+srv.URL, orderScenarioFile(t, "Orders"))
	if err != nil {
		t.Fatalf("the assertion {{email}} must match the echoed value: %v\n%s", err, out)
	}
	email := find(t, emailRe, out)
	if !strings.HasPrefix(email, "user-") || !strings.HasSuffix(email, "@example.com") {
		t.Errorf("email = %q", email)
	}
	if !regexp.MustCompile(`var due = \d{4}-\d{2}-\d{2}`).MatchString(out) {
		t.Errorf("the generated date is listed:\n%s", out)
	}
	if n.Load() != 2 {
		t.Errorf("requests = %d", n.Load())
	}
}

func TestRun_SeedIsPrintedAndReplaysTheRun(t *testing.T) {
	srv, _ := dynServer(t)
	sc := orderScenarioFile(t, "Orders")

	first, err := exec(t, "run", "--var", "base="+srv.URL, sc)
	if err != nil {
		t.Fatal(err)
	}
	seed := find(t, seedRe, first)

	replay, err := exec(t, "run", "--var", "base="+srv.URL, "--seed", seed, sc)
	if err != nil {
		t.Fatal(err)
	}
	if find(t, emailRe, replay) != find(t, emailRe, first) {
		t.Errorf("--seed %s must replay the generated values:\n%s\n%s", seed, first, replay)
	}

	other, _ := exec(t, "run", "--var", "base="+srv.URL, "--seed", "7", sc)
	if find(t, emailRe, other) == find(t, emailRe, first) || find(t, seedRe, other) != "7" {
		t.Errorf("a different seed gives different values:\n%s", other)
	}
}

func TestRun_SeedFromTheEnvironmentVariable(t *testing.T) {
	srv, _ := dynServer(t)
	sc := orderScenarioFile(t, "Orders")
	t.Setenv("CLRSIG_SEED", "4242")
	a, _ := exec(t, "run", "--var", "base="+srv.URL, sc)
	if find(t, seedRe, a) != "4242" {
		t.Errorf("CLRSIG_SEED should set the seed:\n%s", a)
	}
	b, _ := exec(t, "run", "--var", "base="+srv.URL, "--seed", "4242", sc)
	if find(t, emailRe, a) != find(t, emailRe, b) {
		t.Error("the variable and the flag are equivalent")
	}
	c, _ := exec(t, "run", "--var", "base="+srv.URL, "--seed", "1", sc)
	if find(t, seedRe, c) != "1" {
		t.Errorf("--seed wins over CLRSIG_SEED:\n%s", c)
	}

	t.Setenv("CLRSIG_SEED", "not-a-number")
	if _, err := exec(t, "run", "--var", "base="+srv.URL, sc); err == nil || !strings.Contains(err.Error(), "CLRSIG_SEED must be a whole number") {
		t.Errorf("a bad CLRSIG_SEED is an error: %v", err)
	}
}

func TestRun_StaticScenarioPrintsNoSeed(t *testing.T) {
	srv, _ := countingServer(t)
	sc := writeTempFile(t, "s.yaml", scenarioExpecting("Static", srv.URL+"/ping", 200))
	out, err := exec(t, "run", sc)
	if err != nil || strings.Contains(out, "seed") {
		t.Errorf("existing static scenarios keep their output: %v\n%s", err, out)
	}
}

func TestRun_FolderSharesEnvironmentValuesButNotScenarioValues(t *testing.T) {
	srv, _ := dynServer(t)
	root := t.TempDir()
	scenario := func(title string) string {
		return fmt.Sprintf(`schemaVersion: "1"
title: %s
variables:
  - name: rid
    value: "{{runId}}"
  - name: mine
    value: "{{$randomString 8}}"
steps:
  - label: go
    request: {method: GET, url: "{{base}}/x?e={{rid}}"}
`, title)
	}
	writeFiles(t, root, map[string]string{"a.yaml": scenario("A"), "b.yaml": scenario("B")})
	env := writeTempFile(t, "env.yaml", "schemaVersion: \"1\"\nname: ci\nvariables:\n  - name: base\n    value: "+srv.URL+"\n  - name: runId\n    value: \"run-{{$randomString 6}}\"\n")

	out, err := exec(t, "run", "--env", env, "--seed", "5", root)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	rids := regexp.MustCompile(`var rid = (\S+)`).FindAllStringSubmatch(out, -1)
	mines := regexp.MustCompile(`var mine = (\S+)`).FindAllStringSubmatch(out, -1)
	if len(rids) != 2 || rids[0][1] != rids[1][1] || !strings.HasPrefix(rids[0][1], "run-") {
		t.Errorf("an environment function is generated once for the whole command: %v", rids)
	}
	if len(mines) != 2 || mines[0][1] == mines[1][1] {
		t.Errorf("a scenario variable is generated per scenario: %v", mines)
	}
	if strings.Count(out, "seed") != 1 || !strings.Contains(out, "seed 5") {
		t.Errorf("a combined report prints the seed once, in the header:\n%s", out)
	}
}

func TestRun_UnknownFunctionInAStepFailsThatStepOnly(t *testing.T) {
	srv, n := dynServer(t)
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"good.yaml": fmt.Sprintf(orderScenarioYAML, "Good"),
		"typo.yaml": `schemaVersion: "1"
title: Typo
steps:
  - label: fine
    request: {method: GET, url: "{{base}}/x"}
  - label: typo
    failPolicy: continue
    request: {method: GET, url: "{{base}}/x?e={{$randomIntt 1 5}}"}
  - label: after
    request: {method: GET, url: "{{base}}/y"}
`,
	})

	out, err := exec(t, "run", "--var", "base="+srv.URL, "-f", "json", filepath.Join(root, "typo.yaml"))
	if err == nil || exitCode(err) != 1 {
		t.Errorf("a bad function in a step is a failed step (exit 1), got %v", err)
	}
	if !strings.Contains(out, "unknown function $randomIntt") {
		t.Errorf("the report names the bad function: %s", out)
	}
	if got := n.Load(); got != 2 {
		t.Errorf("the other two steps run, the bad one is not sent: %d requests", got)
	}

	// A folder run runs the good scenario as well; the exit code is 1, not 2.
	_, err = exec(t, "run", "--var", "base="+srv.URL, root)
	if err == nil || exitCode(err) != 1 {
		t.Errorf("a folder run reports the failed step with exit 1: %v", err)
	}
}

func TestRun_BadFunctionInAScenarioVariableStopsBeforeAnythingIsSent(t *testing.T) {
	srv, n := dynServer(t)
	sc := writeTempFile(t, "v.yaml", `schemaVersion: "1"
title: V
variables:
  - name: x
    value: "{{$nope}}"
steps:
  - label: a
    request: {method: GET, url: "{{base}}/x"}
`)
	_, err := exec(t, "run", "--var", "base="+srv.URL, sc)
	if err == nil || exitCode(err) == 1 || !strings.Contains(err.Error(), "unknown function $nope") {
		t.Errorf("a bad scenario variable is a configuration error (exit 2): %v", err)
	}
	if n.Load() != 0 {
		t.Errorf("no request may be sent, got %d", n.Load())
	}
}

func TestRun_JSONAndJUnitCarryTheSeed(t *testing.T) {
	srv, _ := dynServer(t)
	sc := orderScenarioFile(t, "Orders")

	out, err := exec(t, "run", "--var", "base="+srv.URL, "--seed", "99", "-f", "json", sc)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Seed      *int64 `json:"seed"`
		Variables []struct{ Name, Value string }
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || doc.Seed == nil || *doc.Seed != 99 || len(doc.Variables) != 2 {
		t.Errorf("json = %+v (%v)", doc, err)
	}

	out, _ = exec(t, "run", "--var", "base="+srv.URL, "--seed", "99", "-f", "junit", sc)
	var ju struct {
		Suites []struct {
			Props []struct {
				Name  string `xml:"name,attr"`
				Value string `xml:"value,attr"`
			} `xml:"properties>property"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal([]byte(out[strings.Index(out, "<?xml"):]), &ju); err != nil || len(ju.Suites[0].Props) != 1 || ju.Suites[0].Props[0].Value != "99" {
		t.Errorf("junit seed property: %+v (%v)", ju, err)
	}
}

func TestRun_FailedAssertionShowsTheResolvedValue(t *testing.T) {
	srv, _ := dynServer(t)
	sc := writeTempFile(t, "s.yaml", `schemaVersion: "1"
title: Mismatch
variables:
  - name: want
    value: "expected-{{$randomInt 100 999}}"
steps:
  - label: a
    request: {method: GET, url: "{{base}}/x?e=different"}
    assertions:
      - {type: jsonpath, path: $.echo, operator: eq, expected: "{{want}}"}
`)
	out, err := exec(t, "run", "--var", "base="+srv.URL, "--seed", "3", sc)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %v\n%s", err, out)
	}
	want := find(t, regexp.MustCompile(`var want = (\S+)`), out)
	if !strings.Contains(out, want) || !strings.Contains(out, "different") {
		t.Errorf("the failure should show the generated expectation %q and the actual value:\n%s", want, out)
	}
}

func TestValidate_CatchesDynamicMistakes(t *testing.T) {
	sc := writeTempFile(t, "s.yaml", `schemaVersion: "1"
title: Bad
variables:
  - name: a
    value: "{{b}}"
  - name: b
    value: "{{$date +7x}}"
steps:
  - label: a
    request: {method: GET, url: "http://x/{{$guidd}}"}
`)
	out, err := exec(t, "validate", sc)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %v\n%s", err, out)
	}
	for _, want := range []string{"defined after it", "is not an offset", "unknown function $guidd"} {
		if !strings.Contains(out, want) {
			t.Errorf("validate output missing %q:\n%s", want, out)
		}
	}
}

func TestBundleRun_ScenarioVariablesAndSeed(t *testing.T) {
	srv, _ := dynServer(t)
	bundle := fmt.Sprintf(`schemaVersion: "1"
title: Dyn
environments:
  - name: local
    variables:
      - {name: base, value: "%s"}
scenarios:
  - title: One
    variables:
      - {name: id, value: "id-{{$randomString 8}}"}
    steps:
      - label: a
        request: {method: GET, url: "{{base}}/x?e={{id}}"}
        assertions:
          - {type: jsonpath, path: $.echo, operator: eq, expected: "{{id}}"}
`, srv.URL)
	path := writeTempFile(t, "b.yaml", bundle)
	a, err := execBundle(t, "bundle", "run", "--seed", "8", path)
	if err != nil {
		t.Fatalf("%v\n%s", err, a)
	}
	b, _ := execBundle(t, "bundle", "run", "--seed", "8", path)
	idRe := regexp.MustCompile(`var id = (\S+)`)
	if find(t, idRe, a) != find(t, idRe, b) || !strings.Contains(a, "seed: 8") {
		t.Errorf("a bundle run replays with --seed:\n%s\n%s", a, b)
	}
}
