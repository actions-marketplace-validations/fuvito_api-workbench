package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const typedBodySpec = `openapi: "3.0.3"
info: {title: typed, version: "1"}
paths:
  /orders:
    post:
      operationId: createOrder
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [item, quantity, express, tags]
              properties:
                id: {type: integer, readOnly: true}
                item: {type: string}
                quantity: {type: integer}
                express: {type: boolean}
                tags: {type: array, items: {type: string}}
                note: {type: string}
      responses:
        "201": {description: created}
`

// A generated body can now carry real numbers, booleans and arrays: the server decodes what
// `conform` sends into typed fields, which was impossible while every value was a quoted string.
func TestConformSendsATypedBodyWhenTheValuesAreThere(t *testing.T) {
	var mu sync.Mutex
	var received []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			http.Error(w, "not JSON: "+string(raw), http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, body)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	spec := filepath.Join(t.TempDir(), "typed.yaml")
	if err := os.WriteFile(spec, []byte(typedBodySpec), 0o600); err != nil {
		t.Fatal(err)
	}

	// Without values nothing is sent, and the report names every missing one, once.
	out, err := exec(t, "conform", "--base-url", srv.URL, spec)
	if exitCode(err) != 1 || !strings.Contains(out, "no value for {{item}}, {{quantity}}, {{express}}, {{tags}}") {
		t.Fatalf("missing values must block the request: %v\n%s", err, out)
	}
	mu.Lock()
	if len(received) != 0 {
		t.Fatalf("nothing should have been sent: %v", received)
	}
	mu.Unlock()

	out, err = exec(t, "conform", "--base-url", srv.URL,
		"--var", "item=Desk", "--var", "quantity=2", "--var", "express=true", "--var", `tags=["a","b"]`, spec)
	if err != nil || !strings.Contains(out, "PASSED") {
		t.Fatalf("a typed body with its values must pass: %v\n%s", err, out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("the request should have been sent once: %v", received)
	}
	got := received[0]
	if got["item"] != "Desk" || got["quantity"] != float64(2) || got["express"] != true {
		t.Errorf("item, quantity and express must arrive as a string, a number and a boolean: %v", got)
	}
	if tags, ok := got["tags"].([]any); !ok || len(tags) != 2 {
		t.Errorf("tags must arrive as an array: %v", got["tags"])
	}
	for _, absent := range []string{"id", "note"} {
		if _, ok := got[absent]; ok {
			t.Errorf("%s is read-only or optional and must not be generated: %v", absent, got)
		}
	}
}
