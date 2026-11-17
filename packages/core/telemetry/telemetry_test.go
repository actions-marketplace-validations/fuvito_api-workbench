package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_DisabledIsNoop(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	c := New(false, dir, "1.0.0")
	if c.enabled {
		t.Fatal("client should be disabled")
	}
	c.Record(Event{Command: "run", Result: "pass"})
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("disabled client must not create the data directory")
	}
}

func TestNew_EnabledPersistsInstallID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	c1 := New(true, dir, "1.2.3")
	if !c1.enabled || c1.version != "1.2.3" {
		t.Fatalf("unexpected client %+v", c1)
	}
	if len(c1.installID) != 32 {
		t.Fatalf("install id should be 32 hex chars, got %q", c1.installID)
	}
	c2 := New(true, dir, "1.2.3")
	if c2.installID != c1.installID {
		t.Fatalf("install id should be stable across clients: %q vs %q", c1.installID, c2.installID)
	}
}

func TestLoadOrCreateInstallID_ReplacesEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "install-id"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if id := loadOrCreateInstallID(dir); len(id) != 32 {
		t.Fatalf("expected a fresh id, got %q", id)
	}
}

func TestNewID_Unique(t *testing.T) {
	if a, b := newID(), newID(); a == b {
		t.Fatal("ids should differ")
	}
}
