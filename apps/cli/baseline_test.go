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

const baselineSpec = `openapi: "3.0.3"
info: {title: base, version: "1"}
paths:
  /ok:
    get:
      operationId: ok
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /legacy:
    get:
      operationId: legacy
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /fresh:
    get:
      operationId: fresh
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
`

// baselineAPI answers /ok correctly; /legacy is broken until fixLegacy is set, /fresh is correct
// until breakFresh is set.
func baselineAPI(t *testing.T) (url string, fixLegacy, breakFresh *atomic.Bool) {
	t.Helper()
	fixLegacy, breakFresh = &atomic.Bool{}, &atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		broken := (r.URL.Path == "/legacy" && !fixLegacy.Load()) || (r.URL.Path == "/fresh" && breakFresh.Load())
		if broken {
			w.Write([]byte(`{"name":"no id"}`))
			return
		}
		w.Write([]byte(`{"id":1}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, fixLegacy, breakFresh
}

func TestConformBaseline_RatchetsQualityUp(t *testing.T) {
	url, fixLegacy, breakFresh := baselineAPI(t)
	spec := writeTempFile(t, "api.yaml", baselineSpec)
	file := filepath.Join(t.TempDir(), "baseline", "conform-baseline.yaml")

	// 1. Day one: /legacy is already broken. Without a baseline the check fails.
	if _, err := exec(t, "conform", "--base-url", url, spec); exitCode(err) != 1 {
		t.Fatalf("an existing violation fails an ordinary run: %v", err)
	}

	// 2. Record it: exit 0 and a versioned file.
	out, err := exec(t, "conform", "--base-url", url, "--write-baseline", file, spec)
	if err != nil || !strings.Contains(out, "Baseline written: 1 violation recorded in") {
		t.Fatalf("write-baseline must succeed: %v\n%s", err, out)
	}
	data, _ := os.ReadFile(file)
	if !strings.Contains(string(data), `schemaVersion: "1"`) || !strings.Contains(string(data), "GET /legacy") {
		t.Fatalf("baseline file:\n%s", data)
	}

	// 3. With the baseline the known violation is accepted.
	out, err = exec(t, "conform", "--base-url", url, "--baseline", file, spec)
	if err != nil || !strings.Contains(out, "Baseline: 1 known, 0 new, 0 fixed") {
		t.Fatalf("known violations are accepted: %v\n%s", err, out)
	}

	// 4. A new violation fails the run, and is named.
	breakFresh.Store(true)
	out, err = exec(t, "conform", "--base-url", url, "--baseline", file, spec)
	if exitCode(err) != 1 || !strings.Contains(out, "1 new") || !strings.Contains(out, "NEW    GET /fresh") {
		t.Fatalf("a new violation must fail: %v\n%s", err, out)
	}
	breakFresh.Store(false)

	// 5. Fixing the old one is reported, and still passes.
	fixLegacy.Store(true)
	out, err = exec(t, "conform", "--base-url", url, "--baseline", file, spec)
	if err != nil || !strings.Contains(out, "0 known, 0 new, 1 fixed") || !strings.Contains(out, "FIXED  GET /legacy") {
		t.Fatalf("fixed violations are reported: %v\n%s", err, out)
	}

	// 6. Re-recording ratchets: the empty baseline now fails if /legacy breaks again.
	if _, err := exec(t, "conform", "--base-url", url, "--write-baseline", file, spec); err != nil {
		t.Fatal(err)
	}
	fixLegacy.Store(false)
	if _, err := exec(t, "conform", "--base-url", url, "--baseline", file, spec); exitCode(err) != 1 {
		t.Errorf("after the ratchet a regression fails: %v", err)
	}
}

func TestConformBaseline_Errors(t *testing.T) {
	url, _, _ := baselineAPI(t)
	spec := writeTempFile(t, "api.yaml", baselineSpec)

	_, err := exec(t, "conform", "--base-url", url, "--baseline", filepath.Join(t.TempDir(), "missing.yaml"), spec)
	if err == nil || exitCode(err) == 1 || !strings.Contains(err.Error(), "--write-baseline") {
		t.Errorf("a missing baseline is a configuration error with a hint: %v", err)
	}
	both := filepath.Join(t.TempDir(), "b.yaml")
	if _, err := exec(t, "conform", "--base-url", url, "--baseline", both, "--write-baseline", both, spec); err == nil || !strings.Contains(err.Error(), "not both") {
		t.Errorf("both flags together: %v", err)
	}
	future := writeTempFile(t, "future.yaml", "schemaVersion: \"9\"\nviolations: []\n")
	if _, err := exec(t, "conform", "--base-url", url, "--baseline", future, spec); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Errorf("a newer baseline version is refused: %v", err)
	}
}

func TestConformBaseline_JSONFileAndMachineReportsStayPure(t *testing.T) {
	url, _, _ := baselineAPI(t)
	spec := writeTempFile(t, "api.yaml", baselineSpec)
	file := filepath.Join(t.TempDir(), "b.json")
	if _, err := exec(t, "conform", "--base-url", url, "--write-baseline", file, spec); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file)
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "{") || !strings.Contains(string(data), `"schemaVersion": "1"`) {
		t.Errorf("a .json baseline is JSON:\n%s", data)
	}
	// the baseline summary goes to stderr, so --format json on stdout is still pure JSON
	var stdout, stderr strings.Builder
	root := newTestRoot()
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"conform", "--base-url", url, "--baseline", file, "--format", "json", spec})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout.String()), "{") || strings.Contains(stdout.String(), "Baseline:") {
		t.Errorf("stdout must stay machine-readable:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Baseline: 1 known") {
		t.Errorf("stderr:\n%s", stderr.String())
	}
}
