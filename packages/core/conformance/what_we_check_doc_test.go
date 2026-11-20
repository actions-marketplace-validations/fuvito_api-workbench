package conformance_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// apps/desktop/wails/help/what-we-check.md (shown in the app's Help panel as What We Check) is the list of what Clear Signal catches and does not catch. These checks
// keep it honest: it may only cite tests that exist, its IDs are unique, and every CS- ID used in
// the documentation or in a test comment is defined there.

var (
	reCitedTest = regexp.MustCompile("`(Test[A-Za-z0-9_]+)`")
	reDefinedID = regexp.MustCompile(`(?m)^\| (CS-[A-Z]+-\d{3}) \|`)
	reAnyID     = regexp.MustCompile(`CS-[A-Z]+-\d{3}`)
	reTestFunc  = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "apps", "desktop", "wails", "help", "what-we-check.md")); err != nil {
		t.Fatalf("apps/desktop/wails/help/what-we-check.md not found from %s: %v", root, err)
	}
	return root
}

func walkTests(t *testing.T, root string, visit func(path, text string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "wailsjs", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			data, err := os.ReadFile(path)
			if err == nil {
				visit(path, string(data))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCatalogCitesRealTests(t *testing.T) {
	root := repoRoot(t)
	doc, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "wails", "help", "what-we-check.md"))
	if err != nil {
		t.Fatal(err)
	}
	defined := map[string]bool{}
	walkTests(t, root, func(_, text string) {
		for _, m := range reTestFunc.FindAllStringSubmatch(text, -1) {
			defined[m[1]] = true
		}
	})
	cited := map[string]bool{}
	for _, m := range reCitedTest.FindAllStringSubmatch(string(doc), -1) {
		cited[m[1]] = true
	}
	if len(cited) < 20 {
		t.Fatalf("the catalog should cite its evidence tests, found only %d", len(cited))
	}
	var missing []string
	for name := range cited {
		if !defined[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("apps/desktop/wails/help/what-we-check.md cites tests that do not exist (renamed or deleted?): %v", missing)
	}
}

func TestCatalogIDsAreUniqueAndDefined(t *testing.T) {
	root := repoRoot(t)
	doc, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "wails", "help", "what-we-check.md"))
	if err != nil {
		t.Fatal(err)
	}
	defined := map[string]bool{}
	for _, m := range reDefinedID.FindAllStringSubmatch(string(doc), -1) {
		if defined[m[1]] {
			t.Errorf("%s is defined twice in apps/desktop/wails/help/what-we-check.md", m[1])
		}
		defined[m[1]] = true
	}
	if len(defined) < 40 {
		t.Fatalf("expected the catalog to define its checks, found %d IDs", len(defined))
	}

	check := func(where, text string) {
		for _, id := range reAnyID.FindAllString(text, -1) {
			if !defined[id] {
				t.Errorf("%s mentions %s, which apps/desktop/wails/help/what-we-check.md does not define", where, id)
			}
		}
	}
	check("apps/desktop/wails/help/what-we-check.md", string(doc))
	walkTests(t, root, func(path, text string) { check(path, text) })
	for _, name := range []string{"README.md", filepath.Join("docs", "backlog.md")} {
		if data, err := os.ReadFile(filepath.Join(root, name)); err == nil {
			check(name, string(data))
		}
	}
}
