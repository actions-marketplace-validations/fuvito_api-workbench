package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The Action is a published interface: one stray character in action.yml breaks every workflow that
// uses it, and nothing else loads the file. (0.7.0 shipped with a lone carriage return in a shell
// script, which YAML reads as a line break, so the whole file was invalid.)
func TestActionFileIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "action.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for i, b := range raw {
		if b == '\r' && (i+1 >= len(raw) || raw[i+1] != '\n') {
			t.Fatalf("action.yml line %d: a lone carriage return; YAML reads it as a line break", bytes.Count(raw[:i], []byte("\n"))+1)
		}
	}

	var doc struct {
		Name   string                    `yaml:"name"`
		Inputs map[string]map[string]any `yaml:"inputs"`
		Runs   struct {
			Using string `yaml:"using"`
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("action.yml is not valid YAML: %v", err)
	}
	if doc.Name == "" || doc.Runs.Using != "composite" || len(doc.Inputs) == 0 || len(doc.Runs.Steps) == 0 {
		t.Fatalf("action.yml is missing its name, inputs or steps: %+v", doc)
	}

	// Every input the scripts read must be declared, so a renamed input cannot be silently ignored.
	for _, name := range []string{"command", "spec", "read-only", "negative", "baseline", "fail-on", "against", "var", "report"} {
		if _, ok := doc.Inputs[name]; !ok {
			t.Errorf("input %q is not declared", name)
		}
	}

	bash, err := osexec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available to check the scripts")
	}
	for _, step := range doc.Runs.Steps {
		if step.Run == "" {
			continue
		}
		cmd := osexec.Command(bash, "-n")
		cmd.Stdin = bytes.NewReader([]byte(step.Run))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("step %q is not valid bash: %v\n%s", step.Name, err, out)
		}
	}
}

// fuvito/api-workbench/clrsig-action is a second published path to the same Action. A copy that
// falls behind is dangerous: GitHub only warns about an input it does not know, so a workflow that
// sets read-only: 'true' on a stale copy sends every request, DELETE included.
func TestSubdirectoryActionIsTheSameAsTheRootAction(t *testing.T) {
	read := func(parts ...string) string {
		data, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
		if err != nil {
			t.Fatal(err)
		}
		return strings.ReplaceAll(string(data), "\r\n", "\n")
	}
	if read("action.yml") != read("clrsig-action", "action.yml") {
		t.Error("clrsig-action/action.yml differs from action.yml: copy the root file over it (cp action.yml clrsig-action/action.yml)")
	}
}

// installScript returns the shell script of the Action's install step.
func installScript(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Runs struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, step := range doc.Runs.Steps {
		if step.Name == "Install clrsig CLI" {
			return step.Run
		}
	}
	t.Fatal("action.yml has no install step")
	return ""
}

// Runs the real install script with curl replaced by a shell function, so no network is used.
func TestActionVerifiesTheDownload(t *testing.T) {
	bash, err := osexec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available to run the script")
	}
	script := installScript(t)
	if strings.Contains(script, ",,}") {
		t.Error("${VAR,,} needs bash 4; the system bash on macOS is 3.2")
	}
	const content = "pretend clrsig binary"
	good := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	cases := []struct {
		name, runnerOS, runnerArch string
		checksum                   string // what the release publishes; "" means the file is missing
		binary                     string
		wantError                  string
	}{
		{name: "linux, matching", runnerOS: "Linux", runnerArch: "X64", checksum: good, binary: "clrsig-linux-amd64"},
		{name: "macOS, matching, upper case and CRLF", runnerOS: "macOS", runnerArch: "ARM64", checksum: strings.ToUpper(good) + "\r", binary: "clrsig-darwin-arm64"},
		{name: "windows, matching", runnerOS: "Windows", runnerArch: "X64", checksum: good, binary: "clrsig-windows-amd64.exe"},
		{name: "mismatch", runnerOS: "Linux", runnerArch: "X64", checksum: strings.Repeat("0", 64), binary: "clrsig-linux-amd64", wantError: "Checksum mismatch"},
		{name: "no checksum file", runnerOS: "Linux", runnerArch: "X64", binary: "clrsig-linux-amd64", wantError: "cannot be verified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.ToSlash(t.TempDir())
			fake := `curl() {
  local out="" url=""
  while [ $# -gt 0 ]; do
    case "$1" in
      -o) out="$2"; shift 2 ;;
      -*) shift ;;
      *) url="$1"; shift ;;
    esac
  done
  echo "$url" >> "$RUNNER_TEMP/urls"
  case "$url" in
    *.sha256) [ -n "$FAKE_SUM" ] || return 22; printf '%s\n' "$FAKE_SUM" ;;
    *) printf '%s' "$FAKE_CONTENT" > "$out" ;;
  esac
}
`
			cmd := osexec.Command(bash)
			cmd.Stdin = strings.NewReader(fake + script)
			cmd.Env = append(os.Environ(), "RUNNER_OS="+tc.runnerOS, "RUNNER_ARCH="+tc.runnerArch, "RUNNER_TEMP="+dir,
				"GITHUB_PATH="+dir+"/github-path", "IN_VERSION=v1.2.3", "FAKE_SUM="+tc.checksum, "FAKE_CONTENT="+content)
			out, err := cmd.CombinedOutput()

			urls, _ := os.ReadFile(dir + "/urls")
			base := "https://github.com/fuvito/api-workbench/releases/download/v1.2.3/" + tc.binary
			if got, want := strings.Fields(string(urls)), []string{base, base + ".sha256"}; !slices.Equal(got, want) {
				t.Errorf("downloaded %v, want %v\n%s", got, want, out)
			}
			installed := "clrsig"
			if tc.runnerOS == "Windows" {
				installed = "clrsig.exe"
			}
			_, statErr := os.Stat(filepath.Join(dir, "clrsig-bin", installed))
			path, _ := os.ReadFile(dir + "/github-path")

			if tc.wantError == "" {
				if err != nil || statErr != nil || len(path) == 0 {
					t.Fatalf("a verified download must be installed and put on the PATH: %v %v %q\n%s", err, statErr, path, out)
				}
				return
			}
			if err == nil || !strings.Contains(string(out), tc.wantError) {
				t.Errorf("want a failure saying %q, got err=%v\n%s", tc.wantError, err, out)
			}
			if statErr == nil || len(path) != 0 {
				t.Errorf("an unverified binary must be removed and not put on the PATH (stat=%v, PATH=%q)", statErr, path)
			}
		})
	}
}

// The install step must not write into the workspace: a repository with its own clrsig/ folder
// (the sandbox has one) made curl fail with "client returned ERROR on write".
func TestActionInstallsOutsideTheWorkspace(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{[]byte(`DEST="clrsig"`), []byte(`DEST="clrsig.exe"`), []byte(`echo "$PWD" >> "$GITHUB_PATH"`)} {
		if bytes.Contains(data, bad) {
			t.Errorf("action.yml installs into the workspace: %s", bad)
		}
	}
	if !bytes.Contains(data, []byte(`$RUNNER_TEMP/clrsig-bin`)) {
		t.Error("the binary must go to $RUNNER_TEMP/clrsig-bin")
	}
}
