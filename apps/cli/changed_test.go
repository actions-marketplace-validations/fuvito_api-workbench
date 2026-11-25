package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// changedSpec writes a spec with /a and /b (and extra paths); bProps adds required property names
// to /b's response.
func changedSpec(extra string, bProps string) string {
	op := func(id, schema string) string {
		return "      operationId: " + id + "\n      responses:\n        \"200\":\n          description: ok\n          content:\n            application/json:\n              schema: " + schema + "\n"
	}
	return `openapi: "3.0.3"
info: {title: changed, version: "1"}
paths:
  /a:
    get:
` + op("a", "{type: object, required: [id], properties: {id: {type: integer}}}") + `  /b:
    get:
` + op("b", "{type: object, required: [id"+bProps+"], properties: {id: {type: integer}, email: {type: string}}}") + extra
}

const changedNewOp = `  /d:
    get:
      operationId: d
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
`

// hitServer answers every GET with an object and records which paths were requested. A path in
// broken answers with a body that breaks its spec.
func hitServer(t *testing.T, broken ...string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits = append(hits, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		for _, b := range broken {
			if r.URL.Path == b {
				_, _ = w.Write([]byte(`{"name":"no id"}`))
				return
			}
		}
		_, _ = w.Write([]byte(`{"id":1,"email":"a@b.c"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), hits...)
	}
}

// changedRepo makes a git repository with the spec committed and makes it the current directory.
func changedRepo(t *testing.T, spec string) (dir, specPath string) {
	t.Helper()
	if _, err := osexec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir = t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := osexec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	specPath = filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-q", "-m", "base")
	t.Chdir(dir)
	return dir, specPath
}

func TestConformChangedAgainst_OnlyChangedOperationsAreSent(t *testing.T) {
	srv, hits := hitServer(t)
	_, specPath := changedRepo(t, changedSpec("", ""))
	// /b gains a required property, and /d is new
	if err := os.WriteFile(specPath, []byte(changedSpec(changedNewOp, ",email")), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--changed-against", "HEAD")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := strings.Join(hits(), ","); got != "/b,/d" {
		t.Errorf("only the new and the changed operation are requested: %q", got)
	}
	for _, want := range []string{"Changed against HEAD: 2 operations (1 added, 1 modified) checked, 1 unchanged and not checked", "SKIP  GET     /a", "PASS  GET     /b", "PASS  GET     /d"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestConformChangedAgainst_JSONReportCarriesTheScope(t *testing.T) {
	srv, _ := hitServer(t)
	_, specPath := changedRepo(t, changedSpec("", ""))
	if err := os.WriteFile(specPath, []byte(changedSpec(changedNewOp, "")), 0o644); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(t.TempDir(), "r.json")
	if _, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--changed-against", "HEAD", "--report", "json="+report); err != nil {
		t.Fatal(err)
	}
	var rep struct {
		Scope struct {
			ChangedAgainst string   `json:"changedAgainst"`
			Checked        []string `json:"checked"`
			Added          int      `json:"added"`
			Unchanged      int      `json:"unchanged"`
		} `json:"scope"`
	}
	if err := json.Unmarshal([]byte(readFile(t, report)), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Scope.ChangedAgainst != "HEAD" || len(rep.Scope.Checked) != 1 || rep.Scope.Checked[0] != "GET /d" || rep.Scope.Added != 1 || rep.Scope.Unchanged != 2 {
		t.Errorf("scope: %+v", rep.Scope)
	}
}

func TestConformChangedAgainst_NothingChangedPassesWithNothingChecked(t *testing.T) {
	srv, hits := hitServer(t)
	changedRepo(t, changedSpec("", ""))
	out, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--changed-against", "HEAD")
	if err != nil {
		t.Fatalf("a pull request that does not touch the API passes: %v\n%s", err, out)
	}
	for _, want := range []string{"no operation changed, so nothing was checked", "NOTHING CHECKED", "no operation changed against HEAD"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if len(hits()) != 0 {
		t.Errorf("nothing is sent: %v", hits())
	}
}

func TestConformChangedAgainst_ABaselineEntryOfAnUncheckedOperationIsNotFixed(t *testing.T) {
	srv, _ := hitServer(t, "/a") // /a already breaks its spec
	_, specPath := changedRepo(t, changedSpec("", ""))
	baseline := filepath.Join(t.TempDir(), "baseline.yaml")
	if _, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--write-baseline", baseline); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte(changedSpec(changedNewOp, "")), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--baseline", baseline, "--changed-against", "HEAD")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "Baseline: 0 known, 0 new, 0 fixed") || strings.Contains(out, "FIXED") {
		t.Errorf("/a was not checked, so its baseline entry is neither known nor fixed:\n%s", out)
	}

	// the same holds for a run limited by --paths
	out, err = exec(t, "conform", "openapi.yaml", "--base-url", srv.URL, "--baseline", baseline, "--paths", "/b")
	if err != nil || strings.Contains(out, "FIXED") {
		t.Errorf("--paths: %v\n%s", err, out)
	}
}

func TestConformChangedAgainst_Errors(t *testing.T) {
	srv, _ := hitServer(t)
	changedRepo(t, changedSpec("", ""))
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"unknown ref":                 {[]string{"conform", "openapi.yaml", "--base-url", srv.URL, "--changed-against", "no-such-ref"}, "was not found"},
		"a URL spec":                  {[]string{"conform", srv.URL + "/openapi.yaml", "--changed-against", "HEAD"}, "needs a local spec file"},
		"a ref that looks like flag":  {[]string{"conform", "openapi.yaml", "--base-url", srv.URL, "--changed-against=--output=x"}, "must not start with"},
		"write-baseline with a scope": {[]string{"conform", "openapi.yaml", "--base-url", srv.URL, "--changed-against", "HEAD", "--write-baseline", "b.yaml"}, "record the baseline with a full run"},
		"write-baseline with tags":    {[]string{"conform", "openapi.yaml", "--base-url", srv.URL, "--tags", "x", "--write-baseline", "b.yaml"}, "record the baseline with a full run"},
	} {
		_, err := exec(t, tc.args...)
		if err == nil || exitCodeOf(err) == 1 || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want an error containing %q", name, err, tc.want)
		}
	}
}
