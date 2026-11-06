package main

import (
	"os"
	"path/filepath"
	"testing"
)

func newWS(t *testing.T, workspacePath string) *WorkspaceService {
	t.Helper()
	ws := &WorkspaceService{}
	ws.workspacePath = workspacePath
	return ws
}

func TestMoveRequest_Basic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "req.yaml")
	dest := filepath.Join(dir, "sub")
	if err := os.WriteFile(src, []byte("method: GET\nurl: http://example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}

	ws := newWS(t, dir)
	newPath, err := ws.MoveRequest(src, dest)
	if err != nil {
		t.Fatalf("MoveRequest: %v", err)
	}
	want := filepath.Join(dest, "req.yaml")
	if newPath != want {
		t.Errorf("returned path: want %q, got %q", want, newPath)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("file not found at destination: %v", err)
	}
	if _, err := os.Stat(src); err == nil {
		t.Error("source file should have been removed")
	}
}

func TestMoveRequest_SameFolderNoOp(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "req.yaml")
	if err := os.WriteFile(src, []byte("method: GET\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ws := newWS(t, dir)
	newPath, err := ws.MoveRequest(src, dir)
	if err != nil {
		t.Fatalf("MoveRequest: %v", err)
	}
	if newPath != src {
		t.Errorf("want %q, got %q", src, newPath)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("file should still exist: %v", err)
	}
}

func TestMoveRequest_ConflictError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "req.yaml")
	dest := filepath.Join(dir, "sub")
	conflict := filepath.Join(dest, "req.yaml")

	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("method: GET\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conflict, []byte("method: POST\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ws := newWS(t, dir)
	_, err := ws.MoveRequest(src, dest)
	if err == nil {
		t.Error("expected conflict error, got nil")
	}
	// Source file should not have been removed.
	if _, statErr := os.Stat(src); statErr != nil {
		t.Errorf("source file should still exist: %v", statErr)
	}
}

func TestMoveRequest_OutOfWorkspaceError(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	src := filepath.Join(dir, "req.yaml")
	if err := os.WriteFile(src, []byte("method: GET\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ws := newWS(t, dir)
	_, err := ws.MoveRequest(src, outside)
	if err == nil {
		t.Error("expected out-of-workspace error, got nil")
	}
}
