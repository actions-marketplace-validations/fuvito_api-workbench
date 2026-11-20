package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// A folder run can be stopped: the request in flight is abandoned and not reported, the ones not
// started yet never run, and the results so far come back at once.
func TestRunFolder_CancelStopsTheRun(t *testing.T) {
	isolateHome(t)
	started := make(chan string, 8)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- r.URL.Path
		if r.URL.Path == "/two" {
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}
	}))
	defer srv.Close()
	defer close(release)

	dir := t.TempDir()
	for i, name := range []string{"one", "two", "three"} {
		body := "schemaVersion: \"1\"\ntitle: " + name + "\nmethod: GET\nurl: " + srv.URL + "/" + name + "\ntimeout: 20s\n"
		if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(i)+"-"+name+".yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewScenarioService(NewWorkspaceService(), NewSettingsService())

	done := make(chan []FolderRunItemIO, 1)
	go func() { done <- svc.RunFolder(dir) }()

	for path := range started { // wait until the second request is in flight
		if path == "/two" {
			break
		}
	}
	svc.CancelFolderRun()

	select {
	case results := <-done:
		if len(results) != 1 || results[0].Name != "one" || !results[0].Passed {
			t.Fatalf("only the finished request is a result: %+v", results)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the run (it was waiting on the slow request)")
	}
	select {
	case p := <-started:
		t.Fatalf("%s started after the run was cancelled", p)
	default:
	}

	// Nothing to cancel afterwards, and a new run works.
	svc.CancelFolderRun()
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv2.Close()
	dir2 := t.TempDir()
	body := "schemaVersion: \"1\"\ntitle: a\nmethod: GET\nurl: " + srv2.URL + "/a\n"
	if err := os.WriteFile(filepath.Join(dir2, "a.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if res := svc.RunFolder(dir2); len(res) != 1 || !res[0].Passed {
		t.Fatalf("a run after a cancelled one: %+v", res)
	}
}
