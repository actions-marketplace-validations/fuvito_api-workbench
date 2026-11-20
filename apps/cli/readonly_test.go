package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const readOnlySpec = `openapi: "3.0.3"
info: {title: ro, version: "1"}
paths:
  /things:
    get:
      operationId: listThings
      responses: {"200": {description: ok}}
    post:
      operationId: createThing
      responses: {"201": {description: created}}
  /things/1:
    delete:
      operationId: deleteThing
      responses: {"204": {description: gone}}
`

func TestConformReadOnlySendsNoWrites(t *testing.T) {
	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writes.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	spec := filepath.Join(t.TempDir(), "ro.yaml")
	if err := os.WriteFile(spec, []byte(readOnlySpec), 0o600); err != nil {
		t.Fatal(err)
	}

	out, _ := exec(t, "conform", "--base-url", srv.URL, "--read-only", "--negative", spec)
	if writes.Load() != 0 {
		t.Fatalf("--read-only sent %d write request(s):\n%s", writes.Load(), out)
	}
	for _, want := range []string{"SKIP  POST    /things", "SKIP  DELETE  /things/1", "2 skipped"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in the output:\n%s", want, out)
		}
	}

	// Without the flag the command keeps its documented behaviour and sends writes.
	exec(t, "conform", "--base-url", srv.URL, spec)
	if writes.Load() == 0 {
		t.Fatal("conform without --read-only must still send write requests")
	}
}

// The command the builder produces for a read-only choice carries the flag, and running it sends
// no writes: this is what the Check Conformance page's "Get CLI command" button relies on.
func TestBuilderReadOnlyConformCommand(t *testing.T) {
	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writes.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"specs/ro.yaml": readOnlySpec})

	out, err := buildAndRun(t, root, "conform", map[string]string{
		"spec":      filepath.Join(root, "specs", "ro.yaml"), // absolute, as the page passes it
		"base-url":  srv.URL,
		"read-only": "true",
		"negative":  "true",
	})
	if err != nil {
		t.Fatalf("the built command failed: %v\n%s", err, out)
	}
	if writes.Load() != 0 {
		t.Fatalf("the built command sent %d write request(s):\n%s", writes.Load(), out)
	}
	if !strings.Contains(out, "2 skipped") {
		t.Fatalf("expected the two write operations to be skipped:\n%s", out)
	}
}
