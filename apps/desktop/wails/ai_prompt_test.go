package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestAIPrompt_AssemblesInstructionsSpecAndExample(t *testing.T) {
	prompt := NewHelpService().GetAIPrompt()

	order := []string{
		"# Your role",
		"# Reference 1 — Format Specification",
		firstLine(formatSpecMarkdown),
		"# Reference 2 — Example Scenario",
		firstLine(exampleScenarioMarkdown),
	}
	pos := -1
	for _, part := range order {
		i := strings.Index(prompt[pos+1:], part)
		if i < 0 {
			t.Fatalf("prompt is missing %q (or it is out of order)", part)
		}
		pos += i + 1
	}
	if !strings.Contains(prompt, "## Bundle File") {
		t.Error("the bundle format must reach the assistant through the spec")
	}
	if strings.Contains(prompt, "Use **Copy AI Prompt**") {
		t.Error("the human-facing introduction must not be sent to the assistant")
	}
	if strings.Contains(prompt, "\r\n") {
		t.Error("the assembled prompt should use plain newlines")
	}
}

func TestBuildAIPrompt_WithoutRuleKeepsEverything(t *testing.T) {
	out := buildAIPrompt("no rule here", "SPEC", "EXAMPLE")
	for _, want := range []string{"no rule here", "SPEC", "EXAMPLE", "Reference 1", "Reference 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	crlf := buildAIPrompt("intro\r\n---\r\nbody", "S", "E")
	if strings.Contains(crlf, "intro") || !strings.Contains(crlf, "body") {
		t.Errorf("CRLF input should be handled: %q", crlf)
	}
}

// The prompt promises the assistant a working output shape, so its own sample bundle
// must import cleanly and its scenarios must pass against the demo server.
func TestAIPrompt_SampleBundleImportsAndRuns(t *testing.T) {
	isolateHome(t)

	md := strings.ReplaceAll(aiPromptMarkdown, "\r\n", "\n")
	blocks := regexp.MustCompile("(?s)```yaml\n(.*?)```").FindAllStringSubmatch(md, -1)
	if len(blocks) < 2 {
		t.Fatal("expected the shape block and the complete sample bundle")
	}
	sample := blocks[len(blocks)-1][1]

	resetDemoState()
	srv := httptest.NewServer(newDemoMux())
	defer srv.Close()
	sample = strings.Replace(sample, "http://localhost:8090", srv.URL, 1)
	// The sample ships a placeholder password; a user fills it in before running.
	sample = strings.Replace(sample, "CHANGE_ME", "secret", 1)

	ws := newWS(t, t.TempDir())
	svc := NewBundleService(ws)

	pre := svc.PreviewBundle(sample, "", "rename")
	if pre.ParseError != "" || pre.HasErrors {
		t.Fatalf("the prompt's sample bundle must be valid: %+v", pre)
	}
	for _, i := range pre.Issues {
		t.Errorf("the sample should be warning-free, got %s: %s", i.Severity, i.Message)
	}
	kinds := map[string]int{}
	for _, e := range pre.Entries {
		kinds[e.Kind]++
	}
	if kinds["environment"] != 1 || kinds["request"] != 1 || kinds["scenario"] != 2 {
		t.Fatalf("entries: %v", kinds)
	}

	if res := svc.ImportBundle(sample, "", "rename"); res.Error != "" || res.Created != 4 {
		t.Fatalf("import: %+v", res)
	}
	envPath := filepath.Join(ws.workspacePath, "environments", "local.yaml")
	if err := ws.SelectEnvironment(envPath); err != nil {
		t.Fatal(err)
	}

	ss := NewScenarioService(ws, NewSettingsService())
	for _, rel := range []string{"scenarios/user-lifecycle.yaml", "scenarios/login-negative.yaml"} {
		if _, err := os.Stat(filepath.Join(ws.workspacePath, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s was not imported: %v", rel, err)
		}
		res := ss.RunScenario(filepath.Join(ws.workspacePath, filepath.FromSlash(rel)), "", 0)
		if res.Error != "" || !res.Passed {
			t.Errorf("%s did not pass: %+v", rel, res)
			for _, st := range res.Steps {
				for _, a := range st.Assertions {
					if !a.Passed {
						t.Logf("  %s / %s: %s", st.Label, a.Type, a.Message)
					}
				}
			}
		}
	}
}

// Guards the guidance that prevents the most likely mistakes in generated bundles.
func TestAIPrompt_KeepsTheCriticalRules(t *testing.T) {
	for _, rule := range []string{
		"at most **five** numbered questions",
		"never after a failed assertion",
		"`timeout: 5s`",
		"Variables and functions are expanded inside assertions",
		"Never invent a function that is not listed",
		"Swagger 2.0",
		"validateSpec: false",
		"Never write real secrets",
		"**one fenced ```yaml code block**",
		"Self-check",
	} {
		if !strings.Contains(strings.ReplaceAll(aiPromptMarkdown, "\r\n", "\n"), rule) && !strings.Contains(strings.ToLower(aiPromptMarkdown), strings.ToLower(rule)) {
			t.Errorf("the prompt no longer says %q", rule)
		}
	}
}

func firstLine(doc string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(doc), "\n")
	return strings.TrimSpace(line)
}
