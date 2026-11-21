package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/reporting"
)

// The desktop page and `clrsig conform` must give the same answer for the same spec, environment
// and base URL. This runs the real CLI (built from apps/cli) and compares it, operation by
// operation, with ConformanceService; it also pins the CLI-level claim in the user guide that
// the Conformance Server fails exactly its three documented operations.
func TestConformanceParity_ServiceMatchesTheCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the CLI")
	}
	// The server keeps state (the DELETE operation removes product 1), so each run gets a fresh one.
	serviceSrv, cliSrv := httptest.NewServer(newConformanceMux()), httptest.NewServer(newConformanceMux())
	defer serviceSrv.Close()
	defer cliSrv.Close()
	svc := newTestConformanceService(t)
	spec, env := conformanceSpecFile(t), conformanceEnvFile(t)

	exe := filepath.Join(t.TempDir(), "clrsig.exe")
	build := exec.Command("go", "build", "-o", exe, ".")
	build.Dir = filepath.Join("..", "..", "cli")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the CLI: %v\n%s", err, out)
	}

	// The service sends writes only when asked; the CLI always does, so compare with writes on.
	service := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: serviceSrv.URL, EnvPath: env, AllowWrites: true})
	if service.Error != "" {
		t.Fatal(service.Error)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "conform", spec, "--env", env, "--base-url", cliSrv.URL, "--format", "json")
	stdout, runErr := cmd.Output()
	exit := 0
	if ee, ok := runErr.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if runErr != nil {
		t.Fatalf("running the CLI: %v", runErr)
	}
	var cli reporting.JSONConformReport
	if err := json.Unmarshal(stdout, &cli); err != nil {
		t.Fatalf("CLI output is not JSON: %v\n%s", err, stdout)
	}

	outcome := func(passed bool) string {
		if passed {
			return "passed"
		}
		return "failed"
	}
	want := map[string]string{}
	for _, op := range cli.Operations {
		want[op.Method+" "+op.Path] = fmt.Sprintf("%s %d", outcome(op.Passed), op.StatusCode)
	}
	got := map[string]string{}
	for _, op := range service.Operations {
		got[op.Method+" "+op.Path] = fmt.Sprintf("%s %d", op.Status, op.StatusCode)
	}
	if len(got) != len(want) || len(want) != 10 {
		t.Fatalf("operations differ: service %v, CLI %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: service %q, CLI %q", k, got[k], v)
		}
	}

	// The CLI-level claim: exactly the documented violations, and a failing exit code.
	var failing []string
	for _, op := range cli.Operations {
		if !op.Passed {
			failing = append(failing, op.OperationID)
		}
	}
	known := append([]string(nil), conformanceKnownViolations...)
	sort.Strings(known)
	sort.Strings(failing)
	if strings.Join(failing, ",") != strings.Join(known, ",") || exit != 1 {
		t.Fatalf("CLI failing = %v (exit %d), want %v and exit 1", failing, exit, known)
	}
	if service.Covered != 10 || service.Failing != 6 || service.Passing != 4 {
		t.Fatalf("service totals: %+v", service)
	}
}

// Cancel stops a run that is waiting on a slow server, and the service can run again afterwards.
func TestConformanceService_CancelStopsARunInProgress(t *testing.T) {
	started := make(chan struct{}, 16)
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer slow.Close()
	defer close(release)

	svc := newTestConformanceService(t)
	spec, env := conformanceSpecFile(t), conformanceEnvFile(t)

	done := make(chan ConformRunOutputIO, 1)
	go func() {
		done <- svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: slow.URL, EnvPath: env, TimeoutMs: 20000})
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the run never reached the server")
	}
	svc.CancelConformance()

	select {
	case out := <-done:
		if out.Covered != 0 || out.Passing != 0 {
			t.Fatalf("a cancelled run must not count operations as covered: %+v", out)
		}
		for _, op := range out.Operations {
			if op.Status == "passed" || op.Status == "failed" {
				t.Errorf("%s %s finished as %s after cancel", op.Method, op.Path, op.Status)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the run (it was waiting up to the 20 s request timeout)")
	}

	// A later run is not stuck on the cancelled one.
	good := httptest.NewServer(newConformanceMux())
	defer good.Close()
	again := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: good.URL, EnvPath: env})
	if again.Error != "" || again.Covered == 0 {
		t.Fatalf("run after cancel: %+v", again)
	}
}

// A second run while one is active is refused, so it cannot take over the first run's cancel handle.
func TestConformanceService_RefusesASecondRunWhileOneIsActive(t *testing.T) {
	started := make(chan struct{}, 16)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer slow.Close()

	svc := newTestConformanceService(t)
	spec, env := conformanceSpecFile(t), conformanceEnvFile(t)
	done := make(chan ConformRunOutputIO, 1)
	go func() {
		done <- svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: slow.URL, EnvPath: env, TimeoutMs: 20000})
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the run never reached the server")
	}

	second := svc.RunConformance(ConformRunInput{SpecPath: spec, BaseURL: slow.URL, EnvPath: env})
	if !strings.Contains(second.Error, "already in progress") {
		t.Fatalf("second run error = %q", second.Error)
	}
	svc.CancelConformance()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the first run")
	}
}
