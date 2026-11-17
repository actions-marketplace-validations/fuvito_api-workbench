package main

import (
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

var fencedYAML = regexp.MustCompile("(?s)```yaml\n(# file: (\\S+)\n.*?)```")

// exampleFiles returns the files embedded in help/example-scenario.md, keyed by the
// path given on each block's "# file:" line.
func exampleFiles(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	md := strings.ReplaceAll(exampleScenarioMarkdown, "\r\n", "\n") // git may check files out with CRLF
	for _, m := range fencedYAML.FindAllStringSubmatch(md, -1) {
		files[m[2]] = m[1]
	}
	for _, want := range []string{"environments/example.yaml", "requests/check-health.yaml", "scenarios/user-onboarding.yaml"} {
		if files[want] == "" {
			t.Fatalf("example-scenario.md is missing the block for %s", want)
		}
	}
	return files
}

func TestExampleScenario_IsServedByHelpService(t *testing.T) {
	if got := NewHelpService().GetExampleScenario(); got != exampleScenarioMarkdown || len(got) < 1000 {
		t.Fatalf("help service should return the embedded example (%d bytes)", len(got))
	}
}

func TestExampleScenario_FilesParseAndValidate(t *testing.T) {
	files := exampleFiles(t)

	env, err := serialization.ParseEnvironmentFile([]byte(files["environments/example.yaml"]), "example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	secrets := 0
	for _, v := range env.Variables {
		if v.Secret {
			secrets++
		}
	}
	if env.Name != "example" || secrets != 3 {
		t.Errorf("environment: %+v (secrets=%d)", env, secrets)
	}

	if _, err := serialization.ParseRequestFile([]byte(files["requests/check-health.yaml"]), "check-health.yaml"); err != nil {
		t.Fatal(err)
	}

	sc, err := serialization.ParseScenarioFile([]byte(files["scenarios/user-onboarding.yaml"]), "user-onboarding.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if issues := scenario.Validate(sc, ""); len(issues) != 0 {
		t.Errorf("the example must be a valid scenario: %v", issues)
	}
}

// The example is the reference for "uses the format to the full extent", so fail
// if it stops demonstrating any documented feature.
func TestExampleScenario_CoversTheWholeFormat(t *testing.T) {
	sc := exampleFiles(t)["scenarios/user-onboarding.yaml"]
	for _, feature := range []string{
		"schemaVersion:", "title:", "description:", "specFile:", "label:", "notes:", "requestRef:", "request:",
		"disabled: true", "validateSpec: false", "failPolicy: stop", "failPolicy: continue", "failPolicy: skip",
		"type: status", "type: jsonpath", "type: header", "type: schema", "name: Login response",
		"operator: eq", "operator: ne", "operator: gt", "operator: lt", "operator: contains", "operator: matches",
		"source: jsonpath", "source: header", "source: status",
		"type: bearer", "type: basic", "type: apikey", "location: header", "location: query",
		"type: json", "type: raw", "type: form-urlencoded", "type: multipart",
		"queryParams:", "headers:", "timeout: ", "{{token}}", "{{userId}}",
	} {
		if !strings.Contains(sc, feature) {
			t.Errorf("example scenario no longer demonstrates %q", feature)
		}
	}
}

// Runs the documented files, verbatim, against the demo server.
func TestExampleScenario_RunsAgainstDemoServer(t *testing.T) {
	isolateHome(t)
	files := exampleFiles(t)

	resetDemoState()
	srv := httptest.NewServer(newDemoMux())
	defer srv.Close()

	ws := t.TempDir()
	write := func(rel, content string) string {
		return writeFile(t, filepath.Join(ws, filepath.FromSlash(rel)), content)
	}
	specPath := write("demo-openapi.yaml", string(demoOpenAPIYAML))

	env := strings.Replace(files["environments/example.yaml"], "http://localhost:8090", srv.URL, 1)
	envPath := write("environments/example.yaml", env)
	write("requests/check-health.yaml", files["requests/check-health.yaml"])
	scn := strings.Replace(files["scenarios/user-onboarding.yaml"], "/absolute/path/to/demo-openapi.yaml", filepath.ToSlash(specPath), 1)
	scnPath := write("scenarios/user-onboarding.yaml", scn)

	wsSvc := newWS(t, ws)
	if err := wsSvc.SelectEnvironment(envPath); err != nil {
		t.Fatal(err)
	}
	res := NewScenarioService(wsSvc, NewSettingsService()).RunScenario(scnPath, "", 0)

	if res.Error != "" || res.SpecError != "" {
		t.Fatalf("run error: %q spec error: %q", res.Error, res.SpecError)
	}
	for i, st := range res.Steps {
		if st.Skipped {
			if st.Label != "Slow endpoint" {
				t.Errorf("step %d %q was skipped unexpectedly", i+1, st.Label)
			}
			continue
		}
		if !st.Passed || st.Error != "" {
			t.Errorf("step %d %q failed (status %d, error %q)", i+1, st.Label, st.StatusCode, st.Error)
			for _, a := range st.Assertions {
				if !a.Passed {
					t.Logf("    %s: %s", a.Type, a.Message)
				}
			}
		}
	}
	if !res.Passed {
		t.Error("the example scenario should pass as a whole")
	}
	if len(res.Steps) != 17 {
		t.Errorf("expected 17 steps (including the disabled one), got %d", len(res.Steps))
	}
}
