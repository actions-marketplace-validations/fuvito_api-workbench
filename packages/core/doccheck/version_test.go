package doccheck

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The version the app reports (the newest release notes entry in AboutDialog.tsx) and the version in
// the user guide's header are the same, so a release cannot ship with one of them forgotten.
func TestVersionsAgree(t *testing.T) {
	root := filepath.Join("..", "..", "..", "apps", "desktop")
	about, err := os.ReadFile(filepath.Join(root, "frontend", "src", "components", "AboutDialog.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	guide, err := os.ReadFile(filepath.Join(root, "wails", "help", "user-guide.md"))
	if err != nil {
		t.Fatal(err)
	}
	app := regexp.MustCompile(`(?m)^\s*version: '(\d+\.\d+\.\d+)'`).FindSubmatch(about)
	doc := regexp.MustCompile(`(?m)^\*\*Version:\*\* (\d+\.\d+\.\d+)`).FindSubmatch(guide)
	if app == nil || doc == nil {
		t.Fatalf("the version was not found: AboutDialog %v, user guide %v", app != nil, doc != nil)
	}
	if string(app[1]) != string(doc[1]) {
		t.Errorf("AboutDialog.tsx says %s and the user guide says %s: bump both for a release", app[1], doc[1])
	}
}
