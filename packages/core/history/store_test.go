package history_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/history"
)

func TestStoreRecordAndRecent(t *testing.T) {
	dir := t.TempDir()
	store, err := history.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	id1, err := store.Record("GET", "https://example.com/api/users", 200, "200 OK", 123, 456)
	if err != nil {
		t.Fatalf("Record 1: %v", err)
	}
	id2, err := store.Record("POST", "https://example.com/api/items", 201, "201 Created", 50, 128)
	if err != nil {
		t.Fatalf("Record 2: %v", err)
	}
	if id1 >= id2 {
		t.Errorf("expected id2 > id1, got %d and %d", id1, id2)
	}

	entries, err := store.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Most recent first
	if entries[0].Method != "POST" || entries[0].StatusCode != 201 {
		t.Errorf("first entry: got %+v", entries[0])
	}
	if entries[1].Method != "GET" || entries[1].DurationMs != 123 {
		t.Errorf("second entry: got %+v", entries[1])
	}
}

func TestStoreRecentLimit(t *testing.T) {
	dir := t.TempDir()
	store, err := history.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	for i := range 5 {
		if _, err := store.Record("GET", "https://example.com", 200, "200 OK", int64(i), 0); err != nil {
			t.Fatalf("Record %d: %v", i, err)
		}
	}

	entries, err := store.Recent(3)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3, got %d", len(entries))
	}
}

func TestStoreEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := history.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	entries, err := store.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty, got %d", len(entries))
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := history.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("directory not created: %v", err)
	}
}
