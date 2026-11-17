package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedRequest(t *testing.T, ws *WorkspaceService, title string) string {
	t.Helper()
	p, err := ws.SaveRequest(RequestFileIO{SchemaVersion: "1", Title: title, Method: "post", URL: "http://x/" + title})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFolders_RenameDeleteAndGuards(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)

	folder, err := ws.CreateFolder("", "One")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.CreateFolder("", "One"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate folder: %v", err)
	}
	if _, err := ws.CreateFolder("", "///"); err == nil {
		t.Error("unusable name should be rejected")
	}

	renamed, err := ws.RenameFolder(folder, "Two")
	if err != nil || filepath.Base(renamed) != "two" {
		t.Fatalf("rename: %q %v", renamed, err)
	}
	if same, err := ws.RenameFolder(renamed, "Two"); err != nil || same != renamed {
		t.Errorf("renaming to the same name is a no-op: %q %v", same, err)
	}
	other, _ := ws.CreateFolder("", "Three")
	if _, err := ws.RenameFolder(other, "Two"); err == nil {
		t.Error("rename onto an existing folder must fail")
	}
	if _, err := ws.RenameFolder(other, "  "); err == nil {
		t.Error("blank name must fail")
	}
	if _, err := ws.RenameFolder(t.TempDir(), "x"); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("outside workspace: %v", err)
	}

	os.WriteFile(filepath.Join(renamed, "req.yaml"), []byte("x"), 0o644)
	if err := ws.DeleteFolder(renamed); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Errorf("non-empty folder must not be deleted: %v", err)
	}
	os.Remove(filepath.Join(renamed, "req.yaml"))
	os.WriteFile(filepath.Join(renamed, ".DS_Store"), []byte("x"), 0o644)
	if err := ws.DeleteFolder(renamed); err != nil {
		t.Errorf("hidden files should be ignored: %v", err)
	}
	if _, err := os.Stat(renamed); !os.IsNotExist(err) {
		t.Error("folder should be gone")
	}
	if err := ws.DeleteFolder(filepath.Join(dir, "requests", "ghost")); err == nil {
		t.Error("missing folder should error")
	}
	if err := ws.DeleteFolder(t.TempDir()); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("outside workspace: %v", err)
	}
}

func TestRenameRequest(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	a := seedRequest(t, ws, "alpha")
	b := seedRequest(t, ws, "beta")

	np, err := ws.RenameRequest(a, "Gamma Ray")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}
	loaded, err := ws.LoadRequest(np)
	if err != nil || loaded.Title != "Gamma Ray" || loaded.URL != "http://x/alpha" {
		t.Errorf("title updated, content preserved: %+v %v", loaded, err)
	}

	if _, err := ws.RenameRequest(b, "Gamma Ray"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("conflict: %v", err)
	}
	if _, err := ws.RenameRequest(b, ""); err == nil {
		t.Error("empty name")
	}
	if _, err := ws.RenameRequest(filepath.Join(dir, "requests", "none.yaml"), "x"); err == nil {
		t.Error("missing source")
	}
	if _, err := ws.RenameRequest(filepath.Join(t.TempDir(), "o.yaml"), "x"); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("outside workspace: %v", err)
	}
	bad := filepath.Join(dir, "requests", "bad.yaml")
	os.WriteFile(bad, []byte("method: ["), 0o644)
	if _, err := ws.RenameRequest(bad, "x"); err == nil {
		t.Error("unparseable source")
	}
}

func TestCopyAndDeleteRequest(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	src := seedRequest(t, ws, "alpha")
	dest := filepath.Join(dir, "requests", "copies")

	first, err := ws.CopyRequest(src, dest)
	if err != nil || filepath.Base(first) != "alpha.yaml" {
		t.Fatalf("first copy: %q %v", first, err)
	}
	second, _ := ws.CopyRequest(src, dest)
	third, _ := ws.CopyRequest(src, dest)
	fourth, _ := ws.CopyRequest(src, dest)
	if filepath.Base(second) != "alpha-copy.yaml" || filepath.Base(third) != "alpha-copy-3.yaml" || filepath.Base(fourth) != "alpha-copy-4.yaml" {
		t.Errorf("copy naming: %s %s %s", second, third, fourth)
	}
	if a, b := mustRead(t, src), mustRead(t, third); a != b {
		t.Error("copy should be byte-identical")
	}
	if _, err := ws.CopyRequest(filepath.Join(dir, "requests", "none.yaml"), dest); err == nil {
		t.Error("missing source")
	}
	if _, err := ws.CopyRequest(src, t.TempDir()); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("outside workspace: %v", err)
	}

	if err := ws.DeleteRequest(first); err != nil {
		t.Fatal(err)
	}
	if err := ws.DeleteRequest(first); err != nil {
		t.Errorf("deleting twice should be fine: %v", err)
	}
	if err := ws.DeleteRequest(""); err == nil {
		t.Error("empty path")
	}
	outside := filepath.Join(t.TempDir(), "keep.yaml")
	os.WriteFile(outside, []byte("x"), 0o644)
	if err := ws.DeleteRequest(outside); err == nil {
		t.Error("files outside the workspace must not be deletable")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("outside file must survive")
	}
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEnvironmentRawAndDelete(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	os.MkdirAll(filepath.Join(dir, "environments"), 0o755)
	path := filepath.Join(dir, "environments", "raw.yaml")

	const good = "schemaVersion: \"1\"\nname: Raw\nvariables:\n  - name: a\n    value: b\n"
	if msg := ws.ValidateEnvironmentYAML(good); msg != "" {
		t.Errorf("valid yaml reported: %s", msg)
	}
	if msg := ws.ValidateEnvironmentYAML("name: ["); msg == "" {
		t.Error("invalid yaml should report an error")
	}

	info, err := ws.SaveEnvironmentRaw(path, good)
	if err != nil || info.Name != "Raw" {
		t.Fatalf("save raw: %+v %v", info, err)
	}
	if got, err := ws.GetEnvironmentRaw(path); err != nil || got != good {
		t.Errorf("raw round trip: %q %v", got, err)
	}
	if _, err := ws.GetEnvironmentRaw(filepath.Join(dir, "x.yaml")); err == nil {
		t.Error("missing raw env")
	}
	if _, err := ws.SaveEnvironmentRaw(path, "name: ["); err == nil || !strings.Contains(err.Error(), "invalid environment") {
		t.Errorf("invalid content: %v", err)
	}
	if got := mustRead(t, path); got != good {
		t.Error("a rejected save must not clobber the file")
	}
	if _, err := ws.SaveEnvironmentRaw("", good); err == nil {
		t.Error("empty path with workspace")
	}
	if _, err := newWS(t, "").SaveEnvironmentRaw("", good); err == nil {
		t.Error("empty path without workspace")
	}

	if err := ws.SelectEnvironment(path); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.SaveEnvironmentRaw(path, strings.Replace(good, "Raw", "Edited", 1)); err != nil {
		t.Fatal(err)
	}
	if ws.ActiveEnvironment().Name != "Edited" {
		t.Error("active env should refresh after raw save")
	}
	if err := ws.DeleteEnvironment(path); err != nil {
		t.Fatal(err)
	}
	if ws.GetActiveEnvironment() != nil {
		t.Error("deleting the active environment clears the selection")
	}
	if err := ws.DeleteEnvironment(path); err != nil {
		t.Errorf("deleting twice: %v", err)
	}
}

func TestResponseCache(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	req := seedRequest(t, ws, "cached")

	if got, err := ws.LoadResponseCache(req); got != nil || err != nil {
		t.Fatalf("no cache yet: %v %v", got, err)
	}
	if err := ws.SaveResponseCache(req, ResponseCacheIO{StatusCode: 201, Body: "hi", DurationMs: 5}); err != nil {
		t.Fatal(err)
	}
	got, err := ws.LoadResponseCache(req)
	if err != nil || got == nil || got.StatusCode != 201 || got.Body != "hi" || got.CachedAt == "" {
		t.Fatalf("cache round trip: %+v %v", got, err)
	}
	if err := ws.ClearResponseCache(req); err != nil {
		t.Fatal(err)
	}
	if got, _ := ws.LoadResponseCache(req); got != nil {
		t.Error("cache should be cleared")
	}
	if err := ws.ClearResponseCache(req); err != nil {
		t.Errorf("clearing twice: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "r.yaml")
	if err := ws.SaveResponseCache(outside, ResponseCacheIO{}); err == nil {
		t.Error("cache for a file outside the workspace must be refused")
	}
	none := newWS(t, "")
	if err := none.SaveResponseCache(req, ResponseCacheIO{}); err == nil {
		t.Error("no workspace")
	}
	if got, err := none.LoadResponseCache(req); got != nil || err != nil {
		t.Error("load without workspace is silently empty")
	}
	if err := none.ClearResponseCache(req); err != nil {
		t.Error("clear without workspace is silent")
	}

	cp, _ := ws.cachePath(req)
	os.WriteFile(cp, []byte("{broken"), 0o644)
	if _, err := ws.LoadResponseCache(req); err == nil {
		t.Error("corrupt cache should surface an error")
	}
}

func TestRecentWorkspacesAndOpenByPath(t *testing.T) {
	isolateHome(t)
	ws := NewWorkspaceService()
	if got := ws.GetRecentWorkspaces(); got == nil || len(got) != 0 {
		t.Errorf("empty list should be non-nil: %v", got)
	}

	var dirs []string
	for i := 0; i < 12; i++ {
		d := t.TempDir()
		dirs = append(dirs, d)
		if got, err := ws.OpenWorkspaceByPath(d); err != nil || got != d {
			t.Fatal(got, err)
		}
	}
	recent := ws.GetRecentWorkspaces()
	if len(recent) != 10 || recent[0] != dirs[11] {
		t.Errorf("recent should be capped at 10, newest first: %v", recent)
	}
	ws.OpenWorkspaceByPath(dirs[5])
	if r := ws.GetRecentWorkspaces(); r[0] != dirs[5] || len(r) != 10 {
		t.Errorf("reopening should move to front without duplicating: %v", r)
	}

	if NewWorkspaceService().GetRecentWorkspaces()[0] != dirs[5] {
		t.Error("recents should persist")
	}

	f := filepath.Join(t.TempDir(), "file")
	os.WriteFile(f, nil, 0o644)
	for name, p := range map[string]string{"empty": "", "missing": filepath.Join(t.TempDir(), "no"), "file": f} {
		if _, err := ws.OpenWorkspaceByPath(p); err == nil {
			t.Errorf("%s path should fail", name)
		}
	}
}

func TestFileTreeAndPeekMethod(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	seedRequest(t, ws, "alpha")
	reqs := filepath.Join(dir, "requests")
	os.MkdirAll(filepath.Join(reqs, "sub"), 0o755)
	os.MkdirAll(filepath.Join(reqs, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(reqs, "sub", "j.json"), []byte(`{"method":"delete"}`), 0o644)
	os.WriteFile(filepath.Join(reqs, "readme.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(reqs, "broken.json"), []byte(`{`), 0o644)

	tree := ws.ListRequests()
	byName := map[string]FileNodeIO{}
	for _, n := range tree {
		byName[n.Name] = n
	}
	if _, ok := byName["readme.txt"]; ok {
		t.Error("non-data files are excluded")
	}
	if _, ok := byName[".hidden"]; ok {
		t.Error("hidden folders are excluded")
	}
	if byName["alpha.yaml"].Method != "POST" {
		t.Errorf("method peek: %+v", byName["alpha.yaml"])
	}
	if sub := byName["sub"]; !sub.IsDir || len(sub.Children) != 1 || sub.Children[0].Method != "DELETE" {
		t.Errorf("nested json: %+v", sub)
	}
	if byName["broken.json"].Method != "" {
		t.Error("unparseable file has no method")
	}
	if peekMethod(filepath.Join(reqs, "missing.yaml")) != "" {
		t.Error("missing file has no method")
	}
}
