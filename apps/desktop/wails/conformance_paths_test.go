package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// The Check Conformance page names an operation as its exported reports do: by the spec's path.
// path stays the generated request's, as in the JSON report.
func TestConformanceService_ThePageGetsTheSpecPath(t *testing.T) {
	srv := localConformanceServer(t)
	svc := newTestConformanceService(t)
	var progress []string
	svc.emit = func(_ string, data ...interface{}) {
		progress = append(progress, data[0].(ConformProgressEvent).Operation)
	}
	in := ConformRunInput{SpecPath: conformanceSpecFile(t), BaseURL: srv.URL, EnvPath: conformanceEnvFile(t)}
	out := svc.RunConformance(in)
	if out.Error != "" {
		t.Fatal(out.Error)
	}

	withParameter := 0
	for _, op := range out.Operations {
		if strings.Contains(op.SpecPath, "{{") || op.SpecPath == "" {
			t.Errorf("%s %s: specPath %q", op.Method, op.Path, op.SpecPath)
		}
		if strings.Contains(op.Path, "{{") {
			withParameter++
		}
	}
	if withParameter == 0 {
		t.Fatal("no operation has a path parameter, so nothing is proven")
	}
	if len(progress) == 0 || strings.Contains(strings.Join(progress, "\n"), "{{") {
		t.Errorf("progress lines: %q", progress)
	}

	// An empty baseline makes every violation new; the list names them the same way.
	path := filepath.Join(testWorkspaceDir(t), "conform-baseline.yaml")
	if err := atomicWrite(path, []byte("schemaVersion: \"2\"\nviolations: []\n")); err != nil {
		t.Fatal(err)
	}
	in.BaselinePath = path
	compared := svc.RunConformance(in)
	if compared.Baseline == nil || len(compared.Baseline.New) == 0 {
		t.Fatalf("expected new violations: %+v", compared.Baseline)
	}
	for _, v := range compared.Baseline.New {
		if strings.Contains(v.Operation, "{{") {
			t.Errorf("baseline list shows the sent path: %q", v.Operation)
		}
	}
}

var conformanceBaselineDiffFixture = conformance.BaselineDiff{New: []conformance.Violation{{Operation: "GET /users/{{id}}", Check: "CS-RESP-005"}}}

func TestBaselineIOShowsTheSpecPath(t *testing.T) {
	got := baselineIO("b.yaml", &conformanceBaselineDiffFixture)
	if len(got.New) != 1 || got.New[0].Operation != "GET /users/{id}" {
		t.Errorf("%+v", got.New)
	}
}
