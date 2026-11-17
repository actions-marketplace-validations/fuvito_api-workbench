package main

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

const probeScenario = `schemaVersion: "1"
title: Probe
steps:
  - label: Ping
    request:
      method: GET
      url: "{{baseUrl}}/ping"
    assertions:
      - type: status
        operator: eq
        expected: 200
`

func envFileWith(t *testing.T, value string) string {
	t.Helper()
	return writeTempFile(t, "env.yaml", "schemaVersion: \"1\"\nname: ci\nvariables:\n  - name: baseUrl\n    value: \""+value+"\"\n")
}

func TestRun_ExpandsEnvironmentReferencesFromTheProcessEnvironment(t *testing.T) {
	srv, _ := countingServer(t)
	t.Setenv("CS_TEST_BASE", srv.URL)
	sc := writeTempFile(t, "s.yaml", probeScenario)

	out, err := exec(t, "run", "--env", envFileWith(t, "${CS_TEST_BASE}"), sc)
	if err != nil {
		t.Fatalf("${CS_TEST_BASE} should be read from the environment: %v\n%s", err, out)
	}
	if strings.Contains(out, "warning") {
		t.Errorf("no warning when the variable is set:\n%s", out)
	}
}

func TestRun_WarnsOnceAboutAnUnsetEnvironmentVariable(t *testing.T) {
	sc := writeTempFile(t, "s.yaml", probeScenario)
	out, err := exec(t, "run", "--env", envFileWith(t, "${CS_DEFINITELY_UNSET}"), sc)
	if exitCode(err) != 1 {
		t.Fatalf("the run itself fails (the URL stays unresolved), got %v\n%s", err, out)
	}
	want := `warning: environment variable CS_DEFINITELY_UNSET is not set (the environment variable "baseUrl" refers to it as ${CS_DEFINITELY_UNSET})`
	if strings.Count(out, want) != 1 {
		t.Errorf("want the warning exactly once:\n%s", out)
	}
}

func TestRun_VarOverridesAnEnvironmentValue(t *testing.T) {
	srv, n := countingServer(t)
	sc := writeTempFile(t, "s.yaml", probeScenario)

	out, err := exec(t, "run", "--env", envFileWith(t, "http://127.0.0.1:1"), "--var", "baseUrl="+srv.URL, sc)
	if err != nil {
		t.Fatalf("--var should win over the environment file: %v\n%s", err, out)
	}
	if n.Load() != 1 {
		t.Errorf("requests = %d", n.Load())
	}
}

func TestRun_VarWorksWithoutAnEnvironmentFile(t *testing.T) {
	srv, _ := countingServer(t)
	sc := writeTempFile(t, "s.yaml", probeScenario)
	if out, err := exec(t, "run", "--var", "baseUrl="+srv.URL, sc); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestRun_VarOverridesAreRepeatableAndOverrideAnUnsetReference(t *testing.T) {
	srv, _ := countingServer(t)
	sc := writeTempFile(t, "s.yaml", strings.Replace(probeScenario, `"{{baseUrl}}/ping"`, `"{{scheme}}://{{host}}/ping"`, 1))
	u := strings.TrimPrefix(srv.URL, "http://")
	out, err := exec(t, "run", "--var", "scheme=http", "--var", "host="+u, sc)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestRun_MalformedVarIsAnError(t *testing.T) {
	sc := writeTempFile(t, "s.yaml", probeScenario)
	for _, bad := range []string{"novalue", "=x", "  =x"} {
		_, err := exec(t, "run", "--var", bad, sc)
		if err == nil || !strings.Contains(err.Error(), "must look like name=value") {
			t.Errorf("--var %q: want a format error, got %v", bad, err)
		}
	}
}

func TestBundleRun_VarOverridesTheBundleEnvironment(t *testing.T) {
	srv, n := countingServer(t)
	path := writeTempFile(t, "b.yaml", strings.Replace(bundleFor("http://127.0.0.1:1"), "expected: 404", "expected: 200", 1))
	out, err := execBundle(t, "bundle", "run", "--var", "baseUrl="+srv.URL, path)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if n.Load() == 0 {
		t.Error("the overridden base URL should have been used")
	}
}

func TestApplyVars(t *testing.T) {
	orig := &domain.Environment{Name: "ci", Variables: []domain.EnvironmentVariable{
		{Name: "token", Value: "old", Secret: true},
		{Name: "baseUrl", Value: "http://a"},
	}}
	got, err := applyVars(orig, []string{"token=new", "extra=1", "baseUrl=http://b=c"})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]domain.EnvironmentVariable{}
	for _, v := range got.Variables {
		byName[v.Name] = v
	}
	if byName["token"].Value != "new" || !byName["token"].Secret {
		t.Errorf("an override keeps the secret flag: %+v", byName["token"])
	}
	if byName["baseUrl"].Value != "http://b=c" {
		t.Errorf("only the first = splits name and value: %+v", byName["baseUrl"])
	}
	if byName["extra"].Value != "1" || byName["extra"].Secret {
		t.Errorf("a new variable is added: %+v", byName["extra"])
	}
	if orig.Variables[0].Value != "old" || len(orig.Variables) != 2 {
		t.Error("the original environment must not change")
	}

	fromNil, err := applyVars(nil, []string{"a=b"})
	if err != nil || len(fromNil.Variables) != 1 || fromNil.Name != "command line" {
		t.Errorf("nil environment: %+v %v", fromNil, err)
	}
}
