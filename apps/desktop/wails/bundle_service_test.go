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
	doc := "schemaVersion: \"1\"\nspecFile: https://guess.example.com/openapi.json\nscenarios:\n  - title: S\n    steps:\n      - {label: a, request: {method: GET, url: \"http://x\"}}\n"

	pre := svc.PreviewBundle(doc, "C:/specs/api.yaml", "rename")
	if pre.HasErrors || pre.SpecFile != "https://guess.example.com/openapi.json" {
		t.Fatalf("%+v", pre)
	}
	if len(pre.Issues) != 0 {
		t.Errorf("a valid bundle has no findings: %+v", pre.Issues)
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

func TestBundleService_ExportThenImportIntoAnotherWorkspace(t *testing.T) {
	isolateHome(t)
	src := newWS(t, t.TempDir())
	writeFile(t, filepath.Join(src.workspacePath, "environments", "prod.yaml"),
		"schemaVersion: \"1\"\nname: prod\nvariables:\n  - {name: baseUrl, value: \"https://api.example.com\"}\n  - {name: apiKey, value: sk_live_123, secret: true}\n")
	writeFile(t, filepath.Join(src.workspacePath, "requests", "ping.yaml"), "schemaVersion: \"1\"\nmethod: GET\nurl: \"{{baseUrl}}/ping\"\n")
	writeFile(t, filepath.Join(src.workspacePath, "scenarios", "flow.yaml"),
		"schemaVersion: \"1\"\ntitle: Flow\nsteps:\n  - {label: Ping, requestRef: ../requests/ping.yaml, assertions: [{type: status, operator: eq, expected: 200}]}\n")

	out := filepath.Join(t.TempDir(), "export.yaml")
	res := NewBundleService(src).exportBundleTo(out)
	if res.Error != "" || res.Path != out || res.Environments != 1 || res.Requests != 1 || res.Scenarios != 1 || res.RedactedSecrets != 1 {
		t.Fatalf("export: %+v", res)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk_live_123") {
		t.Fatalf("a secret leaked into the exported file:\n%s", data)
	}
	if _, err := os.Stat(out + ".tmp"); !os.IsNotExist(err) {
		t.Error("no temp file should remain")
	}

	dst := newWS(t, t.TempDir())
	dstSvc := NewBundleService(dst)
	pre := dstSvc.PreviewBundle(string(data), "", "rename")
	if pre.ParseError != "" || pre.HasErrors || len(pre.Entries) != 3 {
		t.Fatalf("the exported file must import cleanly: %+v", pre)
	}
	if imp := dstSvc.ImportBundle(string(data), "", "rename"); imp.Error != "" || imp.Created != 3 {
		t.Fatalf("import: %+v", imp)
	}
	scn, _ := os.ReadFile(filepath.Join(dst.workspacePath, "scenarios", "flow.yaml"))
	if !strings.Contains(string(scn), "requestRef: ../requests/ping.yaml") {
		t.Errorf("the imported scenario should reference the imported request:\n%s", scn)
	}
}

func TestBundleService_ExportProblems(t *testing.T) {
	isolateHome(t)
	if r := NewBundleService(newWS(t, "")).exportBundleTo(filepath.Join(t.TempDir(), "x.yaml")); !strings.Contains(r.Error, "no workspace") {
		t.Errorf("no workspace: %+v", r)
	}
	if r := NewBundleService(newWS(t, "")).ExportBundle(); !strings.Contains(r.Error, "no workspace") {
		t.Errorf("no workspace (dialog entry point): %+v", r)
	}

	empty := NewBundleService(newWS(t, t.TempDir()))
	r := empty.exportBundleTo(filepath.Join(t.TempDir(), "x.yaml"))
	if r.Error == "" || r.Path != "" {
		t.Errorf("an empty workspace has nothing to export: %+v", r)
	}

	ws := newWS(t, t.TempDir())
	writeFile(t, filepath.Join(ws.workspacePath, "requests", "ok.yaml"), "method: GET\nurl: http://x\n")
	writeFile(t, filepath.Join(ws.workspacePath, "requests", "bad.yaml"), "method: [")
	svc := NewBundleService(ws)
	good := svc.exportBundleTo(filepath.Join(t.TempDir(), "x.yaml"))
	if good.Error != "" || len(good.Skipped) != 1 || !strings.Contains(good.Skipped[0], "bad.yaml") {
		t.Errorf("unparseable files are reported, not fatal: %+v", good)
	}
	if r := svc.exportBundleTo(filepath.Join(t.TempDir(), "no", "such", "dir", "x.yaml")); r.Error == "" {
		t.Errorf("an unwritable destination: %+v", r)
	}
}
