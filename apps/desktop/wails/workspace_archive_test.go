package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func zipEntries(t *testing.T, path string) map[string]string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	out := map[string]string{}
	for _, f := range r.File {
		rc, _ := f.Open()
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := rc.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		rc.Close()
		out[f.Name] = sb.String()
	}
	return out
}

func buildZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "in.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		w, _ := zw.Create(n)
		w.Write([]byte(entries[n]))
	}
	zw.Close()
	f.Close()
	return path
}

func TestCreateWorkspaceZip(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, filepath.Join(ws, "requests", "a", "r.yaml"), "req")
	writeFile(t, filepath.Join(ws, "requests", ".hidden.yaml"), "hidden")
	writeFile(t, filepath.Join(ws, "scenarios", "s.yaml"), "scn")
	writeFile(t, filepath.Join(ws, "environments", "e.yaml"), "env")
	writeFile(t, filepath.Join(ws, ".clrsigcache", "c.json"), "cache")
	writeFile(t, filepath.Join(ws, ".clrsig-coverage", "k.json"), "cov")
	writeFile(t, filepath.Join(ws, "notes.txt"), "outside the exported dirs")

	dest := filepath.Join(t.TempDir(), "out.zip")
	if err := createWorkspaceZip(ws, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file must be renamed away")
	}
	got := zipEntries(t, dest)
	want := map[string]string{"requests/a/r.yaml": "req", "scenarios/s.yaml": "scn", "environments/e.yaml": "env"}
	if len(got) != len(want) {
		t.Fatalf("entries: %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q", k, got[k])
		}
	}

	bare := t.TempDir()
	empty := filepath.Join(t.TempDir(), "empty.zip")
	if err := createWorkspaceZip(bare, empty); err != nil || len(zipEntries(t, empty)) != 0 {
		t.Errorf("workspace without data dirs: %v", err)
	}

	bad := filepath.Join(t.TempDir(), "no", "such", "dir", "x.zip")
	if err := createWorkspaceZip(ws, bad); err == nil {
		t.Error("unwritable destination should fail")
	}
}

func TestApplyWorkspaceImport_Strategies(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	zipPath := buildZip(t, map[string]string{
		"requests/a.yaml": "new-a",
		"requests/b.yaml": "new-b",
	})

	setup := func() {
		os.RemoveAll(filepath.Join(dir, "requests"))
		writeFile(t, filepath.Join(dir, "requests", "a.yaml"), "old-a")
	}

	setup()
	r := ws.ApplyWorkspaceImport(zipPath, "skip")
	if r.Imported != 1 || r.Skipped != 1 || r.Error != "" || mustRead(t, filepath.Join(dir, "requests", "a.yaml")) != "old-a" {
		t.Errorf("skip: %+v", r)
	}

	setup()
	r = ws.ApplyWorkspaceImport(zipPath, "overwrite")
	if r.Imported != 2 || mustRead(t, filepath.Join(dir, "requests", "a.yaml")) != "new-a" {
		t.Errorf("overwrite: %+v", r)
	}

	setup()
	r = ws.ApplyWorkspaceImport(zipPath, "rename")
	if r.Imported != 2 || r.Renamed != 1 ||
		mustRead(t, filepath.Join(dir, "requests", "a.yaml")) != "old-a" ||
		mustRead(t, filepath.Join(dir, "requests", "a-2.yaml")) != "new-a" {
		t.Errorf("rename: %+v", r)
	}
}

func TestApplyWorkspaceImport_RejectsPathTraversal(t *testing.T) {
	isolateHome(t)
	parent := t.TempDir()
	dir := filepath.Join(parent, "ws")
	os.MkdirAll(dir, 0o755)
	ws := newWS(t, dir)

	zipPath := buildZip(t, map[string]string{
		"../escaped.txt":              "pwned",
		"requests/../../escaped2.txt": "pwned",
		"requests/ok.yaml":            "fine",
	})
	r := ws.ApplyWorkspaceImport(zipPath, "overwrite")
	if r.Error != "" || r.Imported != 1 {
		t.Errorf("only the safe entry should be extracted: %+v", r)
	}
	for _, p := range []string{filepath.Join(parent, "escaped.txt"), filepath.Join(parent, "escaped2.txt")} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("zip-slip: %s was written outside the workspace", p)
		}
	}
	if mustRead(t, filepath.Join(dir, "requests", "ok.yaml")) != "fine" {
		t.Error("safe entry should be extracted")
	}
}

func TestApplyWorkspaceImport_Errors(t *testing.T) {
	isolateHome(t)
	if r := newWS(t, "").ApplyWorkspaceImport("x.zip", "skip"); r.Error == "" {
		t.Error("no workspace")
	}
	ws := newWS(t, t.TempDir())
	if r := ws.ApplyWorkspaceImport(filepath.Join(t.TempDir(), "none.zip"), "skip"); !strings.Contains(r.Error, "invalid ZIP") {
		t.Errorf("missing zip: %+v", r)
	}
	notZip := writeFile(t, filepath.Join(t.TempDir(), "n.zip"), "this is not a zip")
	if r := ws.ApplyWorkspaceImport(notZip, "skip"); !strings.Contains(r.Error, "invalid ZIP") {
		t.Errorf("garbage zip: %+v", r)
	}
	if r := newWS(t, "").ScanWorkspaceImport(); r.Error == "" {
		t.Error("scan without workspace")
	}
	if newWS(t, "").ExportWorkspace() == "" {
		t.Error("export without workspace")
	}
}

func TestWorkspaceHasSecrets(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	if ws.WorkspaceHasSecrets() || newWS(t, "").WorkspaceHasSecrets() {
		t.Error("no environments → no secrets")
	}

	put := func(name, vars string) {
		writeFile(t, filepath.Join(dir, "environments", name), "schemaVersion: \"1\"\nname: n\nvariables:\n"+vars)
	}
	put("plain.yaml", "  - name: host\n    value: x\n")
	put("emptysecret.yml", "  - name: tok\n    value: \"\"\n    secret: true\n")
	writeFile(t, filepath.Join(dir, "environments", "broken.yaml"), "name: [")
	writeFile(t, filepath.Join(dir, "environments", "notes.txt"), "secret: true")
	os.Mkdir(filepath.Join(dir, "environments", "sub"), 0o755)
	if ws.WorkspaceHasSecrets() {
		t.Error("empty secrets and non-secret values do not count")
	}

	put("real.yaml", "  - name: tok\n    value: abc\n    secret: true\n")
	if !ws.WorkspaceHasSecrets() {
		t.Error("a non-empty secret should be detected")
	}
}

func TestCreateExampleWorkspace(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	if err := createExampleWorkspace(dir); err != nil {
		t.Fatal(err)
	}
	ws := newWS(t, dir)
	envs := ws.ListEnvironments()
	if len(envs) == 0 {
		t.Fatal("example workspace should ship an environment")
	}
	if tree := ws.ListRequests(); len(tree) == 0 {
		t.Fatal("example workspace should ship a request")
	}
	if err := ws.SelectEnvironment(envs[0].Path); err != nil {
		t.Errorf("example environment must be loadable: %v", err)
	}
	var firstRequest string
	var walk func([]FileNodeIO)
	walk = func(ns []FileNodeIO) {
		for _, n := range ns {
			if n.IsDir {
				walk(n.Children)
			} else if firstRequest == "" {
				firstRequest = n.Path
			}
		}
	}
	walk(ws.ListRequests())
	if _, err := ws.LoadRequest(firstRequest); err != nil {
		t.Errorf("example request must parse: %v", err)
	}
}
