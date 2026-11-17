package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const scenarioDoc = "schemaVersion: \"1\"\ntitle: S\nsteps:\n  - label: One\n    request: {method: GET, url: \"http://x\"}\n"

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func rels(root string, files []string) string {
	var out []string
	for _, f := range files {
		r, _ := filepath.Rel(root, f)
		out = append(out, filepath.ToSlash(r))
	}
	return strings.Join(out, ",")
}

func TestDiscover_FolderRecursiveAndSorted(t *testing.T) {
	root := tree(t, map[string]string{
		"b.yaml":          scenarioDoc,
		"a.yaml":          scenarioDoc,
		"sub/c.yml":       scenarioDoc,
		"sub/deep/d.json": `{"title":"D","steps":[]}`,
	})
	d, err := Discover([]string{root}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "a.yaml,b.yaml,sub/c.yml,sub/deep/d.json" {
		t.Errorf("files = %s", got)
	}
}

func TestDiscover_NoRecursive(t *testing.T) {
	root := tree(t, map[string]string{"a.yaml": scenarioDoc, "sub/b.yaml": scenarioDoc})
	d, err := Discover([]string{root}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "a.yaml" {
		t.Errorf("files = %s, want only the top level", got)
	}
}

func TestDiscover_SkipsNonScenariosAndReportsThem(t *testing.T) {
	root := tree(t, map[string]string{
		"s.yaml":       scenarioDoc,
		"env.yaml":     "schemaVersion: \"1\"\nname: ci\nvariables: []\n",
		"request.yaml": "schemaVersion: \"1\"\nmethod: GET\nurl: http://x\n",
		"openapi.json": `{"openapi":"3.0.3"}`,
		"notes.txt":    "steps are great",
		"README.md":    "# steps",
	})
	d, err := Discover([]string{root}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "s.yaml" {
		t.Errorf("files = %s", got)
	}
	if got := rels(root, d.Skipped); got != "env.yaml,openapi.json,request.yaml" {
		t.Errorf("skipped = %s", got)
	}
}

func TestDiscover_SkipsHiddenAndNodeModules(t *testing.T) {
	root := tree(t, map[string]string{
		"s.yaml":                scenarioDoc,
		".git/hooks/x.yaml":     scenarioDoc,
		".cache/y.yaml":         scenarioDoc,
		"node_modules/z/s.yaml": scenarioDoc,
	})
	d, err := Discover([]string{root}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "s.yaml" {
		t.Errorf("files = %s", got)
	}
}

func TestDiscover_BrokenScenarioIsKeptSoItCanFail(t *testing.T) {
	root := tree(t, map[string]string{
		"broken.yaml": "title: [unclosed\nsteps:\n  - label: x\n",
		"other.yaml":  "title: [unclosed\nno marker here\n",
	})
	d, err := Discover([]string{root}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "broken.yaml" {
		t.Errorf("a broken file that mentions steps must be kept: %s", got)
	}
	if got := rels(root, d.Skipped); got != "other.yaml" {
		t.Errorf("skipped = %s", got)
	}
}

func TestDiscover_ExplicitFilesAreTakenAsGiven(t *testing.T) {
	root := tree(t, map[string]string{"env.yaml": "name: ci\n", "s.yaml": scenarioDoc})
	d, err := Discover([]string{filepath.Join(root, "env.yaml"), filepath.Join(root, "s.yaml")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "env.yaml,s.yaml" {
		t.Errorf("an explicit file is never skipped: %s", got)
	}
	if len(d.Skipped) != 0 {
		t.Errorf("skipped = %v", d.Skipped)
	}
}

func TestDiscover_DeduplicatesAcrossArguments(t *testing.T) {
	root := tree(t, map[string]string{"a.yaml": scenarioDoc, "sub/b.yaml": scenarioDoc})
	d, err := Discover([]string{root, filepath.Join(root, "a.yaml"), filepath.Join(root, "sub")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(root, d.Files); got != "a.yaml,sub/b.yaml" {
		t.Errorf("files = %s", got)
	}
}

func TestDiscover_Errors(t *testing.T) {
	root := tree(t, map[string]string{"env.yaml": "name: ci\n"})
	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Discover([]string{filepath.Join(root, "missing")}, true); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("missing path: %v", err)
	}
	if _, err := Discover([]string{empty}, true); err == nil || !strings.Contains(err.Error(), "no scenarios found") {
		t.Errorf("empty folder: %v", err)
	}
	if _, err := Discover([]string{root}, true); err == nil || !strings.Contains(err.Error(), "no scenarios found") {
		t.Errorf("folder with only non-scenarios: %v", err)
	}
	// every problem is reported, not just the first
	_, err := Discover([]string{filepath.Join(root, "a"), filepath.Join(root, "b")}, true)
	if err == nil || strings.Count(err.Error(), "not found") != 2 {
		t.Errorf("want both problems, got %v", err)
	}
}
