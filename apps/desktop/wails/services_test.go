package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// isolateHome points the user home at a temp dir so config and settings files
// never touch the developer's real ~/.clrsig.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestAtomicWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := atomicWrite(path, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(path, []byte("two")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "two" {
		t.Errorf("content %q", b)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file must not remain")
	}
	if err := atomicWrite(filepath.Join(t.TempDir(), "missing", "f"), nil); err == nil {
		t.Error("write into a missing directory should fail")
	}
}

func TestSettingsService_SaveClampsAndPersists(t *testing.T) {
	home := isolateHome(t)
	s := NewSettingsService()
	if got := s.GetSettings(); got != domain.DefaultSettings() {
		t.Fatalf("fresh settings should be defaults: %+v", got)
	}

	saved := 0
	s.onSave = func() { saved++ }

	in := domain.DefaultSettings()
	in.FontSize = 2
	in.Theme = "neon"
	if got := s.SaveSettings(in); got.FontSize != 10 || got.Theme != "dark" {
		t.Errorf("low font/unknown theme not normalised: %+v", got)
	}
	in.FontSize = 99
	in.Theme = "light"
	if got := s.SaveSettings(in); got.FontSize != 24 || got.Theme != "light" {
		t.Errorf("high font not clamped: %+v", got)
	}
	if saved != 2 {
		t.Errorf("onSave called %d times", saved)
	}
	if _, err := os.Stat(filepath.Join(home, ".clrsig", "settings.json")); err != nil {
		t.Fatalf("settings not persisted: %v", err)
	}

	reloaded := NewSettingsService().GetSettings()
	if reloaded.FontSize != 24 || reloaded.Theme != "light" {
		t.Errorf("settings did not survive restart: %+v", reloaded)
	}

	if got := s.ResetSettings(); got != domain.DefaultSettings() {
		t.Errorf("reset: %+v", got)
	}
	if NewSettingsService().GetSettings() != domain.DefaultSettings() {
		t.Error("reset should be persisted")
	}
}

func TestSettingsService_LoadFillsZeroValuesAndIgnoresCorruptFile(t *testing.T) {
	home := isolateHome(t)
	dir := filepath.Join(home, ".clrsig")
	os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, "settings.json")

	os.WriteFile(path, []byte(`{"proxyUrl":"http://proxy:3128"}`), 0o600)
	got := NewSettingsService().GetSettings()
	def := domain.DefaultSettings()
	if got.ProxyURL != "http://proxy:3128" || got.Theme != def.Theme || got.FontSize != def.FontSize || got.DefaultTimeoutMs != def.DefaultTimeoutMs {
		t.Errorf("zero values should be defaulted: %+v", got)
	}

	os.WriteFile(path, []byte(`{broken`), 0o600)
	if NewSettingsService().GetSettings() != def {
		t.Error("corrupt settings should fall back to defaults")
	}
}

func TestWorkspaceConfig_StartsOnLandingPage(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "environments"), 0o755)

	ws := NewWorkspaceService()
	if _, err := ws.OpenWorkspaceByPath(dir); err != nil {
		t.Fatal(err)
	}
	info, err := ws.SaveEnvironment(EnvironmentIO{Name: "Local Dev", Variables: []EnvironmentVariableIO{{Name: "host", Value: "x"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.SelectEnvironment(info.Path); err != nil {
		t.Fatal(err)
	}

	restarted := NewWorkspaceService()
	if restarted.GetWorkspacePath() != "" || restarted.GetActiveEnvironment() != nil || restarted.ActiveEnvironment() != nil {
		t.Errorf("the app must start with no workspace, got %q", restarted.GetWorkspacePath())
	}
	recent := restarted.GetRecentWorkspaces()
	if len(recent) != 1 || recent[0] != dir {
		t.Errorf("the last workspace should stay in the recent list: %v", recent)
	}
}

func TestWorkspaceConfig_TempWorkspacesLeaveNoResidue(t *testing.T) {
	isolateHome(t)
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv("TMPDIR", tmp)
	stale := filepath.Join(tmp, demoWorkspacePrefix+"123")
	stale2 := filepath.Join(tmp, conformanceWorkspacePrefix+"456")
	for _, d := range []string{stale, stale2} {
		os.MkdirAll(d, 0o755)
	}
	old := time.Now().Add(-2 * tempLegacyStaleAfter) // left by a copy that is gone
	for _, d := range []string{stale, stale2} {
		os.Chtimes(d, old, old)
	}
	saveConfig(workspaceConfig{RecentWorkspaces: []string{stale, filepath.Join(tmp, "gone"), stale2}})

	ws := NewWorkspaceService()
	if got := ws.GetRecentWorkspaces(); len(got) != 0 {
		t.Errorf("temp and missing workspaces must not be listed: %v", got)
	}
	for _, d := range []string{stale, stale2} {
		if _, err := os.Stat(d); err == nil {
			t.Errorf("stale temp workspace %s was not removed", d)
		}
	}

	ws.addRecentWorkspace(filepath.Join(tmp, demoWorkspacePrefix+"789"))
	if len(ws.GetRecentWorkspaces()) != 0 {
		t.Error("a temp workspace must never be recorded as recent")
	}
}

func TestEnvironments_CRUD(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)

	if ws.ListEnvironments() != nil {
		t.Error("no environments dir yet")
	}
	if _, err := newWS(t, "").SaveEnvironment(EnvironmentIO{Name: "x"}); err == nil {
		t.Error("saving without a workspace or path should fail")
	}

	a, err := ws.SaveEnvironment(EnvironmentIO{Name: "Staging / EU", Variables: []EnvironmentVariableIO{
		{Name: "token", Value: "s3cret", Secret: true}, {Name: "host", Value: "h"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(a.Path) != filepath.Join(dir, "environments") || strings.ContainsAny(filepath.Base(a.Path), "/\\ ") {
		t.Errorf("unexpected env path %q", a.Path)
	}
	unnamed, err := ws.SaveEnvironment(EnvironmentIO{})
	if err != nil || filepath.Base(unnamed.Path) != "environment.yaml" {
		t.Fatalf("unnamed env: %+v %v", unnamed, err)
	}
	noExt, err := ws.SaveEnvironment(EnvironmentIO{Name: "n", Path: filepath.Join(dir, "environments", "bare")})
	if err != nil || !strings.HasSuffix(noExt.Path, "bare.yaml") {
		t.Fatalf("missing extension should default to yaml: %+v %v", noExt, err)
	}
	os.WriteFile(filepath.Join(dir, "environments", "notes.txt"), []byte("x"), 0o644)
	os.Mkdir(filepath.Join(dir, "environments", "sub"), 0o755)

	if got := ws.ListEnvironments(); len(got) != 3 {
		t.Errorf("expected 3 environments (txt and dirs ignored), got %+v", got)
	}

	full, err := ws.GetEnvironment(a.Path)
	if err != nil || full.Name != "Staging / EU" || len(full.Variables) != 2 || !full.Variables[0].Secret {
		t.Fatalf("round trip: %+v %v", full, err)
	}
	if _, err := ws.GetEnvironment(filepath.Join(dir, "nope.yaml")); err == nil {
		t.Error("missing env should error")
	}

	if err := ws.SelectEnvironment(a.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.SaveEnvironment(EnvironmentIO{Path: a.Path, Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	if ws.ActiveEnvironment().Name != "Renamed" {
		t.Error("saving the active environment should refresh it in memory")
	}
	if err := ws.SelectEnvironment(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("selecting a missing env should fail")
	}
	if err := ws.SelectEnvironment(""); err != nil || ws.GetActiveEnvironment() != nil {
		t.Error("empty path should clear the selection")
	}
}

func TestRequests_SaveLoadRenameConflict(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)

	if _, err := newWS(t, "").SaveRequest(RequestFileIO{Title: "x", Method: "GET", URL: "http://a"}); err == nil {
		t.Error("save with no workspace and no target should fail")
	}

	path, err := ws.SaveRequest(RequestFileIO{SchemaVersion: "1", Title: "Get Users", Method: "GET", URL: "http://a/users"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != filepath.Join(dir, "requests") {
		t.Errorf("default location: %q", path)
	}

	loaded, err := ws.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Method != "GET" || loaded.URL != "http://a/users" || loaded.FilePath != path {
		t.Errorf("loaded: %+v", loaded)
	}

	other, err := ws.SaveRequest(RequestFileIO{Title: "Other", Method: "GET", URL: "http://a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.SaveRequest(RequestFileIO{Title: "Get Users", Method: "GET", URL: "http://a", FilePath: other}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("name conflict should be rejected: %v", err)
	}

	loaded.Title = "Fetch Users"
	renamed, err := ws.SaveRequest(*loaded)
	if err != nil {
		t.Fatal(err)
	}
	if renamed == path {
		t.Fatal("path should follow the new title")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("old file should be removed after rename")
	}

	sub := filepath.Join(dir, "requests", "nested")
	nested, err := ws.SaveRequest(RequestFileIO{Title: "", Method: "POST", URL: "http://a", TargetDir: sub})
	if err != nil || filepath.Base(nested) != "request.yaml" || filepath.Dir(nested) != sub {
		t.Errorf("untitled request in TargetDir: %q %v", nested, err)
	}

	if _, err := ws.LoadRequest(filepath.Join(dir, "none.yaml")); err == nil {
		t.Error("missing file should error")
	}
	bad := filepath.Join(dir, "bad.yaml")
	os.WriteFile(bad, []byte("method: ["), 0o644)
	if _, err := ws.LoadRequest(bad); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("bad file: %v", err)
	}

	tree := ws.ListRequests()
	if len(tree) == 0 {
		t.Fatal("expected request tree")
	}
	if newWS(t, "").ListRequests() != nil || newWS(t, t.TempDir()).ListRequests() != nil {
		t.Error("no workspace or no requests dir should list nothing")
	}
}

func TestCreateFolder_ConfinedToWorkspace(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)

	p, err := ws.CreateFolder("", "Auth Flows")
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(p); err != nil || !st.IsDir() || filepath.Dir(p) != filepath.Join(dir, "requests") {
		t.Errorf("folder not created under requests: %q %v", p, err)
	}
	if _, err := ws.CreateFolder(t.TempDir(), "escape"); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("folders outside the workspace must be refused: %v", err)
	}
	if _, err := newWS(t, "").CreateFolder("", "x"); err == nil {
		t.Error("no workspace should fail")
	}
}

func TestOpenWorkspaceDirect_SelectsLocalEnvironment(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, "")
	if _, err := ws.SaveEnvironment(EnvironmentIO{Name: "local", Path: filepath.Join(dir, "environments", "local.yaml")}); err == nil {
		t.Fatal("expected failure: environments dir does not exist yet")
	}
	os.MkdirAll(filepath.Join(dir, "environments"), 0o755)
	if _, err := ws.SaveEnvironment(EnvironmentIO{Name: "local", Path: filepath.Join(dir, "environments", "local.yaml")}); err != nil {
		t.Fatal(err)
	}
	ws.openWorkspaceDirect(dir)
	if ws.GetWorkspacePath() != dir || ws.GetActiveEnvironment() == nil || ws.GetActiveEnvironment().Name != "local" {
		t.Errorf("workspace=%q env=%+v", ws.GetWorkspacePath(), ws.GetActiveEnvironment())
	}
}

func TestWorkspaceEnvironment_RememberedPerWorkspace(t *testing.T) {
	isolateHome(t)
	a, b := t.TempDir(), t.TempDir()
	for _, d := range []string{a, b} {
		os.MkdirAll(filepath.Join(d, "environments"), 0o755)
	}
	ws := NewWorkspaceService()
	ws.OpenWorkspaceByPath(a)
	envA, err := ws.SaveEnvironment(EnvironmentIO{Name: "Staging", Variables: []EnvironmentVariableIO{{Name: "h", Value: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.SelectEnvironment(envA.Path); err != nil {
		t.Fatal(err)
	}

	ws.OpenWorkspaceByPath(b)
	if ws.GetActiveEnvironment() != nil {
		t.Error("another workspace must not inherit the previous environment")
	}

	restarted := NewWorkspaceService()
	restarted.OpenWorkspaceByPath(a)
	if env := restarted.GetActiveEnvironment(); env == nil || env.Name != "Staging" || restarted.ActiveEnvironment() == nil {
		t.Errorf("workspace A should come back with its environment: %+v", env)
	}

	restarted.SelectEnvironment("")
	again := NewWorkspaceService()
	again.OpenWorkspaceByPath(a)
	if again.GetActiveEnvironment() != nil {
		t.Error("choosing no environment should be remembered too")
	}
}
