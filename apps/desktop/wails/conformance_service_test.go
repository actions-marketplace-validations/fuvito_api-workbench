package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
)

func newTestConformanceService(t *testing.T) *ConformanceService {
	t.Helper()
	isolateHome(t)
	return NewConformanceService(NewWorkspaceService(), NewSettingsService())
}

func conformanceSpecFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "conformance-openapi.yaml")
	if err := os.WriteFile(path, conformanceOpenAPIYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConformanceService_ReadOnlyByDefaultAndKnownViolations(t *testing.T) {
	var writes atomic.Int32
	inner := newConformanceMux()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writes.Add(1)
		}
		inner.ServeHTTP(w, r)
	}))
	defer srv.Close()
	svc := newTestConformanceService(t)
	spec := conformanceSpecFile(t)

	info := svc.InspectConformSpec(spec)
	if info.Error != "" || info.Writes != 2 || info.Reads != 5 {
		t.Fatalf("inspect: %+v (want 5 reads, 2 writes)", info)
	}

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if writes.Load() != 0 {
		t.Fatalf("read-only run sent %d write(s)", writes.Load())
	}
	if out.Total != 7 || out.Covered != 5 || out.NotCovered != 2 || out.Passing != 2 || out.Failing != 3 {
		t.Fatalf("totals: %+v", out)
	}
	var failing []string
	for _, op := range out.Operations {
		if op.Status == "failed" {
			failing = append(failing, op.OperationID)
		}
	}
	want := append([]string(nil), conformanceKnownViolations...)
	sort.Strings(want)
	sort.Strings(failing)
	if strings.Join(failing, ",") != strings.Join(want, ",") {
		t.Fatalf("failing = %v, want the known violations %v", failing, want)
	}

	withWrites := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: srv.URL, AllowWrites: true})
	if writes.Load() == 0 || withWrites.NotCovered != 0 {
		t.Fatalf("with AllowWrites the write operations must run: writes=%d %+v", writes.Load(), withWrites)
	}
}

func TestConformanceService_RefusesUnconfirmedRemoteBaseURL(t *testing.T) {
	svc := newTestConformanceService(t)
	spec := conformanceSpecFile(t)

	out := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: "https://api.example.com"})
	if !strings.Contains(out.Error, "confirm the base URL") {
		t.Fatalf("remote URL must need confirmation, got %+v", out)
	}
	if out := svc.RunConformance(ConformRunInput{SpecPath: spec}); !strings.Contains(out.Error, "enter a base URL") {
		t.Fatalf("spec without servers and no base URL: %+v", out)
	}
}

func TestIsLocalBaseURL(t *testing.T) {
	for url, want := range map[string]bool{
		"http://localhost:8080": true, "http://127.0.0.1:1": true, "http://[::1]:9": true,
		"https://api.example.com": false, "http://localhost.evil.com": false, "not a url": false,
	} {
		if got := IsLocalBaseURL(url); got != want {
			t.Errorf("IsLocalBaseURL(%q) = %v, want %v", url, got, want)
		}
	}
}
