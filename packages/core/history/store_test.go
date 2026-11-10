package history_test

import (
	"os"
	"path/filepath"
	"strings"
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

	headers := []history.Header{{Name: "Accept", Value: "application/json"}, {Name: "Authorization", Value: "Bearer secret"}}
	id1, err := store.Record("GET", "https://example.com/api/users", 200, "200 OK", 123, 456, headers, "none", "")
	if err != nil {
		t.Fatalf("Record 1: %v", err)
	}
	id2, err := store.Record("POST", "https://example.com/api/items", 201, "201 Created", 50, 128, nil, "json", `{"name":"test"}`)
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
	if entries[0].BodyType != "json" || entries[0].BodyContent != `{"name":"test"}` {
		t.Errorf("first entry body: got type=%q content=%q", entries[0].BodyType, entries[0].BodyContent)
	}
	if entries[1].Method != "GET" || entries[1].DurationMs != 123 {
		t.Errorf("second entry: got %+v", entries[1])
	}
	// Authorization header should be redacted
	if len(entries[1].Headers) != 2 {
		t.Fatalf("second entry headers: expected 2, got %d", len(entries[1].Headers))
	}
	for _, h := range entries[1].Headers {
		if strings.EqualFold(h.Name, "Authorization") && h.Value != "[redacted]" {
			t.Errorf("Authorization header not redacted: %q", h.Value)
		}
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
		if _, err := store.Record("GET", "https://example.com", 200, "200 OK", int64(i), 0, nil, "none", ""); err != nil {
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

func TestConformCumulativeCoverage(t *testing.T) {
	dir := t.TempDir()
	store, err := history.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ops1 := []history.ConformOp{
		{OperationID: "listUsers", Method: "GET", Path: "/users", Covered: true},
		{OperationID: "createUser", Method: "POST", Path: "/users", Covered: true},
		{OperationID: "deleteUser", Method: "DELETE", Path: "/users/{id}", Covered: false},
	}
	ops2 := []history.ConformOp{
		{OperationID: "listUsers", Method: "GET", Path: "/users", Covered: true},
		{OperationID: "createUser", Method: "POST", Path: "/users", Covered: false},
		{OperationID: "deleteUser", Method: "DELETE", Path: "/users/{id}", Covered: true},
	}

	const hash = "abc123"
	if err := store.RecordConformRun("openapi.yaml", hash, ops1); err != nil {
		t.Fatalf("RecordConformRun 1: %v", err)
	}

	// After one run, CumulativeCoverage should report 1 run
	rep, err := store.CumulativeCoverage(hash)
	if err != nil {
		t.Fatalf("CumulativeCoverage: %v", err)
	}
	if rep == nil || rep.RunCount != 1 {
		t.Fatalf("expected 1 run, got %+v", rep)
	}
	if rep.TotalOperations != 3 || len(rep.EverCoveredOps) != 2 || len(rep.NeverCoveredOps) != 1 {
		t.Errorf("after run 1: total=%d covered=%d missed=%d", rep.TotalOperations, len(rep.EverCoveredOps), len(rep.NeverCoveredOps))
	}

	if err := store.RecordConformRun("openapi.yaml", hash, ops2); err != nil {
		t.Fatalf("RecordConformRun 2: %v", err)
	}

	// After two runs, all three operations should be covered cumulatively
	rep, err = store.CumulativeCoverage(hash)
	if err != nil {
		t.Fatalf("CumulativeCoverage after run 2: %v", err)
	}
	if rep.RunCount != 2 {
		t.Errorf("expected 2 runs, got %d", rep.RunCount)
	}
	if len(rep.EverCoveredOps) != 3 || len(rep.NeverCoveredOps) != 0 {
		t.Errorf("expected all covered: %+v / %+v", rep.EverCoveredOps, rep.NeverCoveredOps)
	}
	if rep.Percent != 100.0 {
		t.Errorf("expected 100%%, got %.1f", rep.Percent)
	}
}

func TestConformCumulativeNoRuns(t *testing.T) {
	dir := t.TempDir()
	store, err := history.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	rep, err := store.CumulativeCoverage("unknown-hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep != nil {
		t.Errorf("expected nil for unknown spec, got %+v", rep)
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
