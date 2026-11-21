package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutsideRel(t *testing.T) {
	sep := string(filepath.Separator)
	for rel, want := range map[string]bool{
		"..":                   true,
		".." + sep + "x":       true,
		".":                    false,
		"requests":             false,
		"..notes":              false, // a name that starts with two dots is inside
		"..a" + sep + "b":      false,
		"a" + sep + ".." + "b": false,
	} {
		if got := outsideRel(rel); got != want {
			t.Errorf("outsideRel(%q) = %v, want %v", rel, got, want)
		}
	}
}

func TestDeleteFolder_RefusesWithoutAWorkspaceAndTheWorkspaceItself(t *testing.T) {
	isolateHome(t)
	ws := NewWorkspaceService()
	victim := t.TempDir()
	if err := ws.DeleteFolder(victim); err == nil || !strings.Contains(err.Error(), "open a workspace") {
		t.Fatalf("with no workspace open: %v", err)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatal("the folder must still exist")
	}

	root := t.TempDir()
	ws.enterWorkspace(root)
	if err := ws.DeleteFolder(root); err == nil {
		t.Fatal("the workspace folder itself must not be deletable")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("the workspace must still exist")
	}
}

func TestDeleteFolder_KeepsAFolderThatHoldsAHiddenFolder(t *testing.T) {
	isolateHome(t)
	ws := NewWorkspaceService()
	root := t.TempDir()
	ws.enterWorkspace(root)

	withGit := filepath.Join(root, "requests", "api")
	if err := os.MkdirAll(filepath.Join(withGit, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withGit, ".git", "config"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ws.DeleteFolder(withGit); err == nil || !strings.Contains(err.Error(), ".git") {
		t.Fatalf("a hidden folder must stop the delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(withGit, ".git", "config")); err != nil {
		t.Fatal("its content must be untouched")
	}

	// Hidden *files* still do not make a folder count as non-empty.
	onlyDotFile := filepath.Join(root, "requests", "empty")
	if err := os.MkdirAll(onlyDotFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(onlyDotFile, ".DS_Store"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ws.DeleteFolder(onlyDotFile); err != nil {
		t.Fatalf("a folder with only a hidden file is empty: %v", err)
	}
}

func TestRenameAndDeleteAcceptNamesThatStartWithTwoDots(t *testing.T) {
	isolateHome(t)
	ws := NewWorkspaceService()
	root := t.TempDir()
	ws.enterWorkspace(root)
	dir := filepath.Join(root, "requests", "..notes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ws.DeleteFolder(dir); err != nil {
		t.Fatalf("a folder called ..notes is inside the workspace: %v", err)
	}
}
