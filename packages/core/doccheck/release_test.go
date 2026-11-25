package doccheck

import (
	"bytes"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func repoFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"..", "..", ".."}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

var reBrewTap = regexp.MustCompile(`brew tap fuvito/api-workbench[^\n` + "`" + `)']*`)

// `brew tap user/name` alone clones github.com/user/homebrew-name. There is no such repository: the
// formula lives in this one, so the tap needs the address. Up to 0.13.1 the documents left it out,
// and the Homebrew install could not work.
func TestBrewTapGivesTheRepositoryAddress(t *testing.T) {
	const want = "brew tap fuvito/api-workbench https://github.com/fuvito/api-workbench"
	found := 0
	check := func(rel, text string) {
		for i, line := range strings.Split(text, "\n") {
			for _, m := range reBrewTap.FindAllString(line, -1) {
				found++
				if !strings.HasPrefix(strings.TrimSpace(m), want) {
					t.Errorf("%s:%d says %q; write %q", rel, i+1, strings.TrimSpace(m), want)
				}
			}
		}
	}
	currentDocs(t, check)
	check(".github/workflows/smoke.yml", repoFile(t, ".github", "workflows", "smoke.yml"))
	if found < 3 {
		t.Fatalf("only %d brew tap lines were found, so little was checked", found)
	}
}

type workflowFile struct {
	On   map[string]any `yaml:"on"`
	Jobs map[string]struct {
		Needs any               `yaml:"needs"`
		Uses  string            `yaml:"uses"`
		With  map[string]string `yaml:"with"`
		Steps []struct {
			Name string `yaml:"name"`
			Uses string `yaml:"uses"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func workflow(t *testing.T, name string) workflowFile {
	t.Helper()
	var w workflowFile
	if err := yaml.Unmarshal([]byte(repoFile(t, ".github", "workflows", name)), &w); err != nil {
		t.Fatalf("%s is not valid YAML: %v", name, err)
	}
	return w
}

// The smoke checks run only at a release, so a mistake in them would show when it is too late to
// notice calmly: here their shape and the syntax of their scripts are checked on every change.
func TestTheReleaseRunsTheSmokeChecksAfterPublishing(t *testing.T) {
	release := workflow(t, "release.yml")
	job, ok := release.Jobs["smoke"]
	if !ok || job.Uses != "./.github/workflows/smoke.yml" || job.With["tag"] != "${{ github.ref_name }}" {
		t.Fatalf("release.yml must call smoke.yml with the tag: %+v", job)
	}
	var needs []string
	if list, ok := job.Needs.([]any); ok {
		for _, n := range list {
			needs = append(needs, n.(string))
		}
	}
	// the release must exist (publish) and the formula must name it (update-tap)
	for _, n := range []string{"publish", "update-tap"} {
		if !slices.Contains(needs, n) {
			t.Errorf("smoke must wait for %s, needs %v", n, needs)
		}
	}

	smoke := workflow(t, "smoke.yml")
	for _, trigger := range []string{"workflow_call", "workflow_dispatch"} {
		if _, ok := smoke.On[trigger]; !ok {
			t.Errorf("smoke.yml has no %s trigger", trigger)
		}
	}
	for _, name := range []string{"docker", "action", "homebrew"} {
		if _, ok := smoke.Jobs[name]; !ok {
			t.Errorf("smoke.yml has no %s job", name)
		}
	}
	bash, err := osexec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available to check the scripts")
	}
	for id, j := range smoke.Jobs {
		for _, step := range j.Steps {
			// A value from outside must reach a script through env:, never be pasted into it.
			if strings.Contains(step.Run, "${{") {
				t.Errorf("smoke.yml %s, step %q pastes an expression into its script", id, step.Name)
			}
			if step.Run == "" {
				continue
			}
			cmd := osexec.Command(bash, "-n")
			cmd.Stdin = bytes.NewReader([]byte(step.Run))
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("smoke.yml %s, step %q is not valid bash: %v\n%s", id, step.Name, err, out)
			}
		}
	}
}

// Wails is installed in CI and in the release at the version the app is built against, never at
// latest: the bindings check compares generated files, so another version would report drift.
func TestWorkflowsInstallTheWailsVersionOfTheApp(t *testing.T) {
	m := regexp.MustCompile(`github\.com/wailsapp/wails/v2 (v\d+\.\d+\.\d+)`).FindStringSubmatch(repoFile(t, "apps", "desktop", "wails", "go.mod"))
	if m == nil {
		t.Fatal("the Wails version was not found in go.mod")
	}
	found := 0
	for _, name := range []string{"ci.yml", "release.yml"} {
		for _, use := range regexp.MustCompile(`wails/v2/cmd/wails@(\S+)`).FindAllStringSubmatch(repoFile(t, ".github", "workflows", name), -1) {
			found++
			if use[1] != m[1] {
				t.Errorf("%s installs wails@%s, go.mod uses %s", name, use[1], m[1])
			}
		}
	}
	if found < 4 {
		t.Errorf("found %d wails installs; ci.yml and release.yml were expected to have 4", found)
	}
}

// CI regenerates the desktop bindings and runs both comparisons.
func TestCIChecksTheBindings(t *testing.T) {
	ci := repoFile(t, ".github", "workflows", "ci.yml")
	for _, want := range []string{
		"wails generate module",
		"git diff --exit-code -- apps/desktop/frontend/wailsjs",
		"git ls-files --others --exclude-standard apps/desktop/frontend/wailsjs",
		"python3 -m unittest discover -s scripts",
		"python3 scripts/check_bindings.py",
	} {
		if !strings.Contains(ci, want) {
			t.Errorf("ci.yml does not run %q", want)
		}
	}
}
