package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/history"
	"github.com/spf13/cobra"
)

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func readSettings(t *testing.T, home string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".clrsig", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTelemetryCommands(t *testing.T) {
	home := isolateHome(t)

	if isTelemetryEnabled() {
		t.Fatal("telemetry must be off without a settings file")
	}
	out, err := execFull(t, "telemetry", "status")
	if err != nil || !strings.Contains(out, "Telemetry: disabled") {
		t.Errorf("status: %q %v", out, err)
	}

	out, err = execFull(t, "telemetry", "enable")
	if err != nil || !strings.Contains(out, "Telemetry enabled.") || !strings.Contains(out, "No request URLs") {
		t.Errorf("enable: %q %v", out, err)
	}
	if !isTelemetryEnabled() || readSettings(t, home)["telemetryEnabled"] != true {
		t.Error("setting should be persisted as enabled")
	}
	if out, _ := execFull(t, "telemetry", "status"); !strings.Contains(out, "Telemetry: enabled") {
		t.Errorf("status after enable: %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".clrsig", "settings.json.tmp")); !os.IsNotExist(err) {
		t.Error("atomic write must not leave a temp file")
	}

	out, err = execFull(t, "telemetry", "disable")
	if err != nil || !strings.Contains(out, "Telemetry disabled.") || isTelemetryEnabled() {
		t.Errorf("disable: %q %v", out, err)
	}
}

func TestWriteTelemetrySetting_PreservesOtherSettings(t *testing.T) {
	home := isolateHome(t)
	dir := filepath.Join(home, ".clrsig")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"light","fontSize":18,"proxyUrl":"http://p:1"}`), 0o600)

	if err := writeTelemetrySetting(true); err != nil {
		t.Fatal(err)
	}
	s := readSettings(t, home)
	if s["theme"] != "light" || s["fontSize"] != float64(18) || s["proxyUrl"] != "http://p:1" || s["telemetryEnabled"] != true {
		t.Errorf("existing settings must survive: %v", s)
	}
}

func TestIsTelemetryEnabled_IgnoresCorruptSettings(t *testing.T) {
	home := isolateHome(t)
	dir := filepath.Join(home, ".clrsig")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{broken"), 0o600)
	if isTelemetryEnabled() {
		t.Error("corrupt settings mean telemetry stays off")
	}
	if newTelemetryClient() == nil {
		t.Error("client should always be constructed")
	}
}

func TestCoverageList_TextAndJSON(t *testing.T) {
	ws := t.TempDir()
	if out, err := execFull(t, "coverage", "list", "-w", ws); err != nil || !strings.Contains(out, "No coverage data") {
		t.Errorf("empty: %q %v", out, err)
	}
	now := time.Date(2026, 3, 4, 10, 30, 0, 0, time.UTC)
	coverage.Save(ws, coverage.StoredCoverage{SpecFile: "a.yaml", TotalOps: 4, TotalRuns: 3, PassedRuns: 2, FailedRuns: 1, Exercised: []string{"GET /a", "GET /b"}, LastRunAt: now})
	coverage.Save(ws, coverage.StoredCoverage{SpecFile: "b.yaml", TotalOps: 1, TotalRuns: 1, PassedRuns: 1, Exercised: []string{"GET /x"}})

	out, err := execFull(t, "coverage", "list", "-w", ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"a.yaml", "2/4 (50.0%)", "3 run(s) · 2 passed · 1 failed", "last run 2026-03-04", "b.yaml", "1/1 (100.0%)"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(strings.SplitN(out, "b.yaml", 2)[1], "last run") {
		t.Error("an entry without a run time must not print one")
	}

	out, err = execFull(t, "coverage", "list", "-w", ws, "-f", "json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		SpecFile  string   `json:"specFile"`
		Covered   int      `json:"covered"`
		Total     int      `json:"total"`
		LastRunAt string   `json:"lastRunAt"`
		Exercised []string `json:"exercised"`
	}
	if err := json.Unmarshal([]byte(out), &entries); err != nil || len(entries) != 2 {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if entries[0].SpecFile != "a.yaml" || entries[0].Covered != 2 || entries[0].LastRunAt == "" || len(entries[0].Exercised) != 2 {
		t.Errorf("first entry (most recent run first): %+v", entries[0])
	}
	if entries[1].LastRunAt != "" {
		t.Errorf("zero run time should be omitted: %+v", entries[1])
	}
}

func TestPrintCumulativeCoverage(t *testing.T) {
	isolateHome(t)
	run := func() string {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		printCumulativeCoverage(cmd, "hash-1")
		return buf.String()
	}

	if out := run(); out != "" {
		t.Errorf("no history → no output, got %q", out)
	}

	path, err := history.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	store, err := history.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ops := func(a, b bool) []history.ConformOp {
		return []history.ConformOp{
			{OperationID: "a", Method: "GET", Path: "/a", Covered: a},
			{OperationID: "b", Method: "GET", Path: "/b", Covered: b},
		}
	}
	if err := store.RecordConformRun("spec.yaml", "hash-1", ops(true, false)); err != nil {
		t.Fatal(err)
	}
	if out := run(); out != "" {
		t.Errorf("a single run is not cumulative: %q", out)
	}

	if err := store.RecordConformRun("spec.yaml", "hash-1", ops(true, false)); err != nil {
		t.Fatal(err)
	}
	out := run()
	if !strings.Contains(out, "Cumulative coverage (2 runs):  1/2 (50.0%)") || !strings.Contains(out, "Never covered: GET /b") {
		t.Errorf("unexpected output: %q", out)
	}

	if err := store.RecordConformRun("spec.yaml", "hash-1", ops(true, true)); err != nil {
		t.Fatal(err)
	}
	out = run()
	if !strings.Contains(out, "(3 runs):  2/2 (100.0%)") || strings.Contains(out, "Never covered") {
		t.Errorf("full coverage output: %q", out)
	}
}
