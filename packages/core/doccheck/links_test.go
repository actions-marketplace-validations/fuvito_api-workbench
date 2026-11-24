// Package doccheck holds tests that keep the repository's documents in order.
package doccheck

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var reLink = regexp.MustCompile(`\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// Every relative link in the Markdown documents points at a file or folder that exists, so a moved
// or renamed document cannot leave dead links behind.
func TestMarkdownLinksPointAtFilesThatExist(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var checked int
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build", "wailsjs", "test-results", "playwright-report":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := stripCode(string(data))
		for _, m := range reLink.FindAllStringSubmatch(text, -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "<") {
				continue
			}
			if i := strings.IndexAny(target, "#?"); i >= 0 {
				target = target[:i]
			}
			if target == "" {
				continue
			}
			if dec, err := url.PathUnescape(target); err == nil {
				target = dec
			}
			checked++
			full := filepath.Join(filepath.Dir(path), filepath.FromSlash(target))
			if _, err := os.Stat(full); err != nil {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s links to %q, which does not exist", filepath.ToSlash(rel), m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 50 {
		t.Fatalf("only %d links were checked; the walk is not finding the documents", checked)
	}
}

// stripCode removes fenced and inline code, where text such as ](x) is an example, not a link.
func stripCode(s string) string {
	var out []string
	in := false
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			in = !in
			continue
		}
		if !in {
			out = append(out, regexp.MustCompile("`[^`]*`").ReplaceAllString(line, ""))
		}
	}
	return strings.Join(out, "\n")
}
