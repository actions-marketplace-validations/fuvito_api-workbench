package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const svcBundle = `schemaVersion: "1"
title: Svc
environments:
  - name: Local
    variables: [{name: baseUrl, value: "http://x"}]
requests:
  - {file: ping, method: GET, url: "{{baseUrl}}/ping"}
scenarios:
  - title: Ping flow
    steps:
      - {label: Ping, requestRef: ping, assertions: [{type: status, operator: eq, expected: 200}]}
`

func TestBundleService_PreviewThenImport(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	svc := NewBundleService(ws)

	pre := svc.PreviewBundle(svcBundle, "", "rename")
	if pre.ParseError != "" || pre.HasErrors || pre.Title != "Svc" || len(pre.Entries) != 3 {
		t.Fatalf("preview: %+v", pre)
	}
	for _, e := range pre.Entries {
		if e.Action != "create" || e.Path != e.RequestedPath {
			t.Errorf("entry: %+v", e)
		}
	}
	if _, err := os.Stat(filepath.Join(ws.workspacePath, "scenarios")); !os.IsNotExist(err) {
		t.Fatal("a preview must not write anything")
	}

	res := svc.ImportBundle(svcBundle, "", "rename")
	if res.Error != "" || res.Created != 3 || len(res.Written) != 3 {
		t.Fatalf("import: %+v", res)
	}
	scn, err := os.ReadFile(filepath.Join(ws.workspacePath, "scenarios", "ping-flow.yaml"))
	if err != nil || !strings.Contains(string(scn), "requestRef: ../requests/ping.yaml") {
		t.Errorf("scenario: %v\n%s", err, scn)
	}

	again := svc.PreviewBundle(svcBundle, "", "rename")
	for _, e := range again.Entries {
		if e.Action != "rename" || !strings.Contains(e.Path, "-2.") {
			t.Errorf("a second import should rename: %+v", e)
		}
	}
	if sk := svc.ImportBundle(svcBundle, "", "skip"); sk.Skipped != 3 || sk.Created != 0 {
		t.Errorf("skip: %+v", sk)
	}
}

func TestBundleService_ReportsProblems(t *testing.T) {
	isolateHome(t)
	svc := NewBundleService(newWS(t, t.TempDir()))

	if p := svc.PreviewBundle("a: [", "", "rename"); p.ParseError == "" || !p.HasErrors {
		t.Errorf("parse error: %+v", p)
	}
	if p := svc.PreviewBundle("", "", "rename"); p.ParseError == "" {
		t.Errorf("empty: %+v", p)
	}
	bad := "schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - {label: a, requestRef: nope}\n"
	p := svc.PreviewBundle(bad, "", "rename")
	if !p.HasErrors || len(p.Issues) == 0 || p.Issues[0].Severity != "error" || len(p.Entries) != 0 {
		t.Errorf("validation: %+v", p)
	}
	if r := svc.ImportBundle(bad, "", "rename"); r.Error == "" {
		t.Error("an invalid bundle must not import")
	}
	if p := svc.PreviewBundle(svcBundle, "", "merge"); p.ParseError == "" {
		t.Error("unknown strategy")
	}
	if r := svc.ImportBundle(svcBundle, "", "merge"); r.Error == "" {
		t.Error("unknown strategy on import")
	}
}

func TestBundleService_SpecFileOverrideAndWarnings(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	svc := NewBundleService(ws)
	doc := "schemaVersion: \"1\"\nspecFile: https://guess.example.com/openapi.json\nscenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GET, url: \"http://x\"}, timeoutMs: 100}\n"

	pre := svc.PreviewBundle(doc, "C:/specs/api.yaml", "rename")
	if pre.HasErrors || pre.SpecFile != "https://guess.example.com/openapi.json" {
		t.Fatalf("%+v", pre)
	}
	if len(pre.Issues) != 1 || pre.Issues[0].Severity != "warning" {
		t.Errorf("expected the not-applied warning: %+v", pre.Issues)
	}
	if r := svc.ImportBundle(doc, "C:/specs/api.yaml", "rename"); r.Error != "" {
		t.Fatal(r.Error)
	}
	data, _ := os.ReadFile(filepath.Join(ws.workspacePath, "scenarios", "s.yaml"))
	if !strings.Contains(string(data), "specFile: C:/specs/api.yaml") {
		t.Errorf("override should win:\n%s", data)
	}
}

func TestBundleService_NoWorkspace(t *testing.T) {
	isolateHome(t)
	svc := NewBundleService(newWS(t, ""))
	if p := svc.PreviewBundle(svcBundle, "", "rename"); !strings.Contains(p.ParseError, "no workspace") {
		t.Errorf("%+v", p)
	}
	if r := svc.ImportBundle(svcBundle, "", "rename"); !strings.Contains(r.Error, "no workspace") {
		t.Errorf("%+v", r)
	}
}

func TestReadBundleFile(t *testing.T) {
	dir := t.TempDir()
	ok := writeFile(t, filepath.Join(dir, "b.yaml"), svcBundle)
	if f := readBundleFile(ok); f.Error != "" || f.Text != svcBundle || f.Path != ok {
		t.Errorf("%+v", f)
	}
	if f := readBundleFile(filepath.Join(dir, "missing.yaml")); f.Error == "" {
		t.Error("missing file")
	}
	big := filepath.Join(dir, "big.yaml")
	os.WriteFile(big, make([]byte, maxBundleBytes+1), 0o644)
	if f := readBundleFile(big); !strings.Contains(f.Error, "too large") || f.Text != "" {
		t.Errorf("%+v", f)
	}
}

// The bundle example in the format spec must stay a valid bundle.
func TestFormatSpec_BundleExampleIsValid(t *testing.T) {
	spec := strings.ReplaceAll(formatSpecMarkdown, "\r\n", "\n")
	start := strings.Index(spec, "## Bundle File")
	if start < 0 {
		t.Fatal("format spec has no Bundle File section")
	}
	section := spec[start:]
	open := strings.Index(section, "```yaml\n")
	end := strings.Index(section[open+8:], "```")
	if open < 0 || end < 0 {
		t.Fatal("no yaml example in the Bundle File section")
	}
	example := section[open+8 : open+8+end]

	isolateHome(t)
	svc := NewBundleService(newWS(t, t.TempDir()))
	p := svc.PreviewBundle(example, "", "rename")
	if p.ParseError != "" || p.HasErrors {
		t.Fatalf("the spec's bundle example is not a valid bundle: %+v", p)
	}
	kinds := map[string]int{}
	for _, e := range p.Entries {
		kinds[e.Kind]++
	}
	if kinds["environment"] != 1 || kinds["request"] != 1 || kinds["scenario"] != 1 {
		t.Errorf("the example should show one of each kind: %v", kinds)
	}
}
