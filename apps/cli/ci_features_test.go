package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const diffBase = `openapi: "3.0.3"
info: {title: t, version: "1"}
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name: {type: string}
                nickname: {type: string}
      responses:
        "201": {description: created}
`

// removing a request property is classified WARN (potentially breaking), not ERR
func diffRevisionWarn() string {
	return strings.Replace(diffBase, "                nickname: {type: string}\n", "", 1)
}

func diffFiles(t *testing.T) (base, rev string) {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"base.yaml": diffBase, "rev.yaml": diffRevisionWarn()})
	return filepath.Join(dir, "base.yaml"), filepath.Join(dir, "rev.yaml")
}

// ---- diff --fail-on ----

func TestDiff_FailOnThresholds(t *testing.T) {
	base, rev := diffFiles(t)

	out, err := exec(t, "diff", base, rev)
	if err != nil || !strings.Contains(out, "[WARN]") {
		t.Fatalf("a warning does not fail by default: %v\n%s", err, out)
	}
	out, err = exec(t, "diff", "--fail-on", "warning", base, rev)
	if exitCode(err) != 1 || !strings.Contains(out, `1 at or above "warning"`) {
		t.Errorf("--fail-on warning fails on a WARN: %v\n%s", err, out)
	}
	if _, err = exec(t, "diff", "--fail-on", "info", base, rev); exitCode(err) != 1 {
		t.Errorf("--fail-on info fails on any change: %v", err)
	}
	if _, err = exec(t, "diff", "--fail-on", "info", base, base); err != nil {
		t.Errorf("no changes never fail: %v", err)
	}
}

func TestDiff_FailOnIsReflectedInReports(t *testing.T) {
	base, rev := diffFiles(t)

	out, _ := exec(t, "diff", "--fail-on", "warning", "--format", "json", base, rev)
	var doc struct {
		FailOn string `json:"failOn"`
		Failed bool   `json:"failed"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || doc.FailOn != "warning" || !doc.Failed {
		t.Errorf("json = %+v (%v)\n%s", doc, err, out)
	}

	out, _ = exec(t, "diff", "--fail-on", "warning", "--format", "junit", base, rev)
	var ju struct {
		Failures int `xml:"failures,attr"`
	}
	if err := xml.Unmarshal([]byte(out[strings.Index(out, "<?xml"):]), &ju); err != nil || ju.Failures != 1 {
		t.Errorf("junit failures = %d (%v)", ju.Failures, err)
	}
	out, _ = exec(t, "diff", "--format", "junit", base, rev)
	_ = xml.Unmarshal([]byte(out[strings.Index(out, "<?xml"):]), &ju)
	if ju.Failures != 0 {
		t.Errorf("at the default threshold the WARN is not a failure, got %d", ju.Failures)
	}
}

func TestDiff_FailOnValidation(t *testing.T) {
	base, rev := diffFiles(t)
	if _, err := exec(t, "diff", "--fail-on", "error", base, rev); err == nil || !strings.Contains(err.Error(), "--fail-on must be one of") {
		t.Errorf("unknown level: %v", err)
	}
	if _, err := exec(t, "diff", "--breaking-only", "--fail-on", "warning", base, rev); err == nil || !strings.Contains(err.Error(), "--breaking-only") {
		t.Errorf("--breaking-only conflicts with a stricter --fail-on: %v", err)
	}
}

// ---- markdown ----

func TestMarkdownFormat_AllCommands(t *testing.T) {
	srv, _ := countingServer(t)
	root := suite(t, srv.URL)

	out, err := exec(t, "run", "--format", "markdown", filepath.Join(root, "a-first.yaml"))
	if err != nil || !strings.HasPrefix(out, "# First") || !strings.Contains(out, "✅ **PASSED**") {
		t.Errorf("run markdown: %v\n%s", err, out)
	}
	out, err = exec(t, "run", "--format", "markdown", root)
	if err != nil || !strings.Contains(out, "2/2 scenarios passed") {
		t.Errorf("folder markdown: %v\n%s", err, out)
	}

	spec := writeTempFile(t, "api.yaml", minimalSpec)
	out, err = exec(t, "conform", "--base-url", srv.URL, "--format", "markdown", spec)
	if err != nil || !strings.Contains(out, "# test-api 1.0.0") {
		t.Errorf("conform markdown: %v\n%s", err, out)
	}

	base, rev := diffFiles(t)
	out, _ = exec(t, "diff", "--format", "markdown", base, rev)
	if !strings.Contains(out, "# API Spec Diff") || !strings.Contains(out, "removed the request property") {
		t.Errorf("diff markdown:\n%s", out)
	}

	path := writeTempFile(t, "b.yaml", strings.Replace(bundleFor(srv.URL), "expected: 404", "expected: 200", 1))
	out, err = execBundle(t, "bundle", "run", "--format", "markdown", path)
	if err != nil || !strings.Contains(out, "# CLI suite") {
		t.Errorf("bundle markdown: %v\n%s", err, out)
	}
}

func TestUnknownFormatIsRejectedBeforeRunning(t *testing.T) {
	srv, n := countingServer(t)
	sc := writeTempFile(t, "s.yaml", scenarioExpecting("S", srv.URL+"/ping", 200))
	_, err := exec(t, "run", "--format", "pdf", sc)
	if err == nil || !strings.Contains(err.Error(), "unknown --format") {
		t.Errorf("want an unknown-format error, got %v", err)
	}
	if n.Load() != 0 {
		t.Errorf("nothing may run when the format is wrong, got %d requests", n.Load())
	}
}

// ---- --report ----

func TestReport_ExtraFilesFromOneRun(t *testing.T) {
	srv, n := countingServer(t)
	root := suite(t, srv.URL)
	reports := filepath.Join(root, "reports")

	out, err := exec(t, "run",
		"--report", "junit="+filepath.Join(reports, "all.xml"),
		"--report", "html="+filepath.Join(reports, "all.html"),
		"--report", "markdown="+filepath.Join(reports, "all.md"),
		root)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "Run PASSED") {
		t.Errorf("the screen still gets the --format text report:\n%s", out)
	}
	if got := n.Load(); got != 2 {
		t.Errorf("one run, not one per format: %d requests for 2 scenarios", got)
	}
	for file, marker := range map[string]string{"all.xml": "<testsuite ", "all.html": "<!DOCTYPE html>", "all.md": "# reports"} {
		_ = marker
		if _, statErr := os.Stat(filepath.Join(reports, file)); statErr != nil {
			t.Errorf("%s was not written: %v", file, statErr)
		}
	}
	if !strings.Contains(readFile(t, filepath.Join(reports, "all.xml")), "<testsuite ") ||
		!strings.Contains(readFile(t, filepath.Join(reports, "all.html")), "<!DOCTYPE html>") ||
		!strings.Contains(readFile(t, filepath.Join(reports, "all.md")), "✅ **PASSED**") {
		t.Error("each extra file holds its own format")
	}
}

func TestReport_CombinesWithOutput(t *testing.T) {
	srv, _ := countingServer(t)
	sc := writeTempFile(t, "s.yaml", scenarioExpecting("S", srv.URL+"/ping", 200))
	dir := t.TempDir()
	primary := filepath.Join(dir, "primary.json")
	_, err := exec(t, "run", "--format", "json", "--output", primary, "--report", "junit="+filepath.Join(dir, "extra.xml"), sc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, primary), `"passed"`) || !strings.Contains(readFile(t, filepath.Join(dir, "extra.xml")), "<testsuite") {
		t.Error("--output holds the --format report and --report the extra one")
	}
}

func TestReport_ValidationHappensBeforeAnyRequest(t *testing.T) {
	srv, n := countingServer(t)
	sc := writeTempFile(t, "s.yaml", scenarioExpecting("S", srv.URL+"/ping", 200))
	dir := t.TempDir()
	for _, tc := range []struct{ arg, want string }{
		{"junit", "must look like format=file"},
		{"=x.xml", "must look like format=file"},
		{"junit=", "must look like format=file"},
		{"pdf=x.pdf", "unknown format"},
		{"junit=" + filepath.Join(dir, "same.xml"), ""}, // first use is fine
	} {
		if tc.want == "" {
			continue
		}
		_, err := exec(t, "run", "--report", tc.arg, sc)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("--report %q: want %q, got %v", tc.arg, tc.want, err)
		}
	}
	same := filepath.Join(dir, "same.xml")
	_, err := exec(t, "run", "--output", same, "--report", "junit="+same, sc)
	if err == nil || !strings.Contains(err.Error(), "already used") {
		t.Errorf("the same file twice is an error: %v", err)
	}
	if n.Load() != 0 {
		t.Errorf("a bad --report must stop the run before any request, got %d", n.Load())
	}
}

func TestReport_ConformDiffAndBundle(t *testing.T) {
	srv, _ := countingServer(t)
	dir := t.TempDir()

	spec := writeTempFile(t, "api.yaml", minimalSpec)
	if _, err := exec(t, "conform", "--base-url", srv.URL, "--report", "junit="+filepath.Join(dir, "c.xml"), "--report", "markdown="+filepath.Join(dir, "c.md"), spec); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, filepath.Join(dir, "c.xml")), "<testsuite") || !strings.Contains(readFile(t, filepath.Join(dir, "c.md")), "# test-api") {
		t.Error("conform extra reports")
	}

	base, rev := diffFiles(t)
	if _, err := exec(t, "diff", "--report", "html="+filepath.Join(dir, "d.html"), base, rev); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, filepath.Join(dir, "d.html")), "API Spec Diff") {
		t.Error("diff extra report")
	}

	path := writeTempFile(t, "b.yaml", strings.Replace(bundleFor(srv.URL), "expected: 404", "expected: 200", 1))
	if _, err := execBundle(t, "bundle", "run", "--report", "junit="+filepath.Join(dir, "b.xml"), path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, filepath.Join(dir, "b.xml")), "<testsuite") {
		t.Error("bundle extra report")
	}
}

// ---- diff --against ----

func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := osexec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := osexec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	writeFiles(t, dir, map[string]string{"api/openapi.yaml": diffBase})
	git("add", ".")
	git("commit", "-q", "-m", "base")
	return dir
}

func TestDiff_AgainstAGitRef(t *testing.T) {
	dir := gitRepo(t)
	spec := filepath.Join(dir, "api", "openapi.yaml")

	out, err := exec(t, "diff", "--against", "HEAD", spec)
	if err != nil || !strings.Contains(out, "No changes.") {
		t.Fatalf("an unchanged file has no differences: %v\n%s", err, out)
	}

	if err := os.WriteFile(spec, []byte(diffRevisionWarn()), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = exec(t, "diff", "--against", "HEAD", spec)
	if err != nil || !strings.Contains(out, "removed the request property `nickname`") {
		t.Errorf("the working copy is compared with HEAD: %v\n%s", err, out)
	}
	if _, err = exec(t, "diff", "--against", "HEAD", "--fail-on", "warning", spec); exitCode(err) != 1 {
		t.Errorf("--against works with --fail-on: %v", err)
	}
}

func TestDiff_AgainstWorksFromAnotherDirectory(t *testing.T) {
	dir := gitRepo(t)
	spec := filepath.Join(dir, "api", "openapi.yaml")
	if err := os.WriteFile(spec, []byte(diffRevisionWarn()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	out, err := exec(t, "diff", "--against", "HEAD", spec)
	if err != nil || !strings.Contains(out, "nickname") {
		t.Errorf("%v\n%s", err, out)
	}
}

func TestDiff_AgainstErrors(t *testing.T) {
	dir := gitRepo(t)
	spec := filepath.Join(dir, "api", "openapi.yaml")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown ref", []string{"--against", "no-such-branch", spec}, "was not found"},
		{"two arguments", []string{"--against", "HEAD", spec, spec}, "accepts 1 arg"},
		{"a url", []string{"--against", "HEAD", "https://example.com/openapi.yaml"}, "needs a local spec file"},
	}
	for _, c := range cases {
		_, err := exec(t, append([]string{"diff"}, c.args...)...)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want %q, got %v", c.name, c.want, err)
		}
	}

	// a file that is not in the repository at that ref
	newSpec := filepath.Join(dir, "api", "new.yaml")
	if err := os.WriteFile(newSpec, []byte(diffBase), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := exec(t, "diff", "--against", "HEAD", newSpec); err == nil || !strings.Contains(err.Error(), "does not exist at git ref") {
		t.Errorf("a new file: %v", err)
	}

	// not in a git repository at all
	loose := writeTempFile(t, "loose.yaml", diffBase)
	if _, err := exec(t, "diff", "--against", "HEAD", loose); err == nil || !strings.Contains(err.Error(), "git repository") {
		t.Errorf("outside a repository: %v", err)
	}
}

func TestDiff_TwoArgumentsAreStillRequiredWithoutAgainst(t *testing.T) {
	base, _ := diffFiles(t)
	if _, err := exec(t, "diff", base); err == nil || !strings.Contains(err.Error(), "accepts 2 arg") {
		t.Errorf("one argument without --against: %v", err)
	}
}
