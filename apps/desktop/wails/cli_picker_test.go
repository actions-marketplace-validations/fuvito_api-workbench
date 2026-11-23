package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The pickers open in the workspace when the field is empty, otherwise where the field points.
func TestPickerStart(t *testing.T) {
	ws := t.TempDir()
	runFrom := filepath.Join(ws, "run")
	for _, d := range []string{filepath.Join(ws, "specs"), runFrom, filepath.Join(runFrom, "sub")} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(ws, "specs", "api.yaml")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, value, runFrom, want string }{
		{"empty field opens the workspace", "", runFrom, ws},
		{"blank field opens the workspace", "   ", "", ws},
		{"web address opens the workspace", "https://example.com/openapi.yaml", runFrom, ws},
		{"absolute file opens its folder", file, runFrom, filepath.Join(ws, "specs")},
		{"absolute folder opens that folder", filepath.Join(ws, "specs"), runFrom, filepath.Join(ws, "specs")},
		{"relative path is read from the run-from folder", "sub", runFrom, filepath.Join(runFrom, "sub")},
		{"relative path without a run-from folder uses the workspace", "specs/api.yaml", "", filepath.Join(ws, "specs")},
		{"a file that does not exist yet opens its nearest existing folder", "specs/new/out.html", "", filepath.Join(ws, "specs")},
		{"nothing exists on the way: the workspace", filepath.Join(ws, "..", "no-such-root-xyz", "a", "b.yaml"), runFrom, filepath.Clean(filepath.Join(ws, ".."))},
	}
	for _, c := range cases {
		if got := pickerStart(c.value, ws, c.runFrom); got != c.want {
			t.Errorf("%s: pickerStart(%q) = %q, want %q", c.name, c.value, got, c.want)
		}
	}
	if got := pickerStart("", "", ""); got != "" {
		t.Errorf("no workspace and no value: %q", got)
	}
	if got := pickerStart("a.yaml", "", ""); got != "" {
		t.Errorf("a relative value with no base: %q", got)
	}
}
