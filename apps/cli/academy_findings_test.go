package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const filterSpec = `openapi: "3.0.3"
info: {title: filt, version: "1"}
paths:
  /things:
    get:
      operationId: listThings
      tags: [things]
      responses: {"200": {description: ok}}
`

// A filter that selects nothing must not look like a green run (it proves nothing), and the
// message must say how --paths matches.
func TestConform_FilterThatMatchesNothingIsAnError(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits++ }))
	defer srv.Close()
	spec := writeTempFile(t, "filt.yaml", filterSpec)

	for _, args := range [][]string{{"--paths", "/nothing-here"}, {"--tags", "nope"}} {
		out, err := exec(t, append([]string{"conform", "--base-url", srv.URL, spec}, args...)...)
		if exitCode(err) != 2 && !strings.Contains(err.Error(), "no operation matches") {
			t.Fatalf("%v must fail with a clear message, got %v\n%s", args, err, out)
		}
		if !strings.Contains(out, "NOTHING CHECKED") || strings.Contains(out, "PASSED") {
			t.Fatalf("%v: the report must not say PASSED:\n%s", args, out)
		}
		if !strings.Contains(err.Error(), "no operation matches "+args[0]) || !strings.Contains(err.Error(), "MSYS_NO_PATHCONV") {
			t.Fatalf("the message must name the filter and the Git Bash cause: %v", err)
		}
	}
	if hits != 0 {
		t.Fatalf("nothing should be sent, got %d request(s)", hits)
	}

	// A filter that does match still works.
	if out, err := exec(t, "conform", "--base-url", srv.URL, "--paths", "/things", spec); err != nil || !strings.Contains(out, "PASSED") {
		t.Fatalf("a matching filter must run: %v\n%s", err, out)
	}
}

func TestConform_SpecWithNoOperationsDoesNotSayPassed(t *testing.T) {
	spec := writeTempFile(t, "empty.yaml", "openapi: \"3.0.3\"\ninfo: {title: e, version: \"1\"}\npaths: {}\n")
	out, err := exec(t, "conform", "--base-url", "http://127.0.0.1:1", spec)
	if err != nil || !strings.Contains(out, "NOTHING CHECKED") || !strings.Contains(out, "declares no operations") || strings.Contains(out, "PASSED") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestConform_BaselineHeadlineAgreesWithTheExitCode(t *testing.T) {
	url, _, _ := baselineAPI(t)
	spec := writeTempFile(t, "api.yaml", baselineSpec)
	file := t.TempDir() + "/base.yaml"
	if _, err := exec(t, "conform", "--base-url", url, "--write-baseline", file, spec); err != nil {
		t.Fatal(err)
	}
	out, err := exec(t, "conform", "--base-url", url, "--baseline", file, spec)
	if err != nil || !strings.Contains(out, "PASSED with 1 known violation") || strings.Contains(out, "FAILED  3") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestConform_UndeclaredNotFoundPointsAtTheRoute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no such route"}`))
	}))
	defer srv.Close()
	spec := writeTempFile(t, "filt.yaml", filterSpec)
	out, _ := exec(t, "conform", "--base-url", srv.URL, "--explain", spec)
	if !strings.Contains(out, "route exists on the server") || !strings.Contains(out, "no such route") {
		t.Fatalf("a 404 must point at the route and show the body:\n%s", out)
	}
}

// A server that cuts the connection is not "not running": each operation keeps its own error and
// the hint names the broken response, not the server process.
func TestConform_DroppedConnectionKeepsItsOwnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("short"))
	}))
	defer srv.Close()
	spec := writeTempFile(t, "filt.yaml", filterSpec)
	out, err := exec(t, "conform", "--base-url", srv.URL, "--explain", spec)
	if exitCode(err) != 1 {
		t.Fatalf("exit: %v\n%s", err, out)
	}
	if strings.Contains(out, "is the server running?") || !strings.Contains(out, "error:") || !strings.Contains(out, "Content-Length") {
		t.Fatalf("the error and a matching hint must be shown:\n%s", out)
	}
}
