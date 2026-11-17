package coverage

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestKey(t *testing.T) {
	got := Key("https://API.example.com/v1/openapi.yaml")
	if got != "https_api_example_com_v1_openapi_yaml.json" {
		t.Fatalf("unexpected key %q", got)
	}
	long := Key(strings.Repeat("a", 200))
	if len(long) != 80+len(".json") {
		t.Fatalf("long key not truncated: %d", len(long))
	}
}

func TestSaveLoadReset(t *testing.T) {
	ws := t.TempDir()
	spec := "specs/api.yaml"

	if got := Load(ws, spec); got.SpecFile != spec || got.TotalRuns != 0 {
		t.Fatalf("expected empty stored coverage, got %+v", got)
	}

	in := StoredCoverage{SpecFile: spec, Exercised: []string{"GET /a"}, TotalOps: 2, TotalRuns: 1, PassedRuns: 1}
	if err := Save(ws, in); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(DirPath(ws), Key(spec)+".tmp")); !os.IsNotExist(err) {
		t.Fatal("temp file should not remain after atomic save")
	}
	if got := Load(ws, spec); !reflect.DeepEqual(got, in) {
		t.Fatalf("round trip mismatch: %+v vs %+v", got, in)
	}

	if err := Reset(ws, spec); err != nil {
		t.Fatal(err)
	}
	if got := Load(ws, spec); got.TotalRuns != 0 {
		t.Fatal("expected data removed after reset")
	}
	if err := Reset(ws, spec); err != nil {
		t.Fatalf("reset of missing file should succeed: %v", err)
	}
}

func TestSaveEmptyWorkspaceIsNoop(t *testing.T) {
	if err := Save("", StoredCoverage{SpecFile: "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	ws := t.TempDir()
	spec := "bad.yaml"
	if err := os.MkdirAll(DirPath(ws), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(DirPath(ws), Key(spec)), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := Load(ws, spec); got.SpecFile != spec || got.TotalRuns != 0 {
		t.Fatalf("corrupt file should yield empty coverage, got %+v", got)
	}
}

func TestMergeHelpers(t *testing.T) {
	if got := MergeStatusCodes([]int{404, 200}, []int{200, 500}); !reflect.DeepEqual(got, []int{200, 404, 500}) {
		t.Fatalf("got %v", got)
	}
	if got := MergeExercised([]string{"POST /b", "GET /a"}, []string{"GET /a", "DELETE /c"}); !reflect.DeepEqual(got, []string{"DELETE /c", "GET /a", "POST /b"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSummary(t *testing.T) {
	r := StoredCoverage{SpecFile: "s", Exercised: []string{"GET /a", "GET /b", "GET /c"}, TotalOps: 2}.Summary()
	if r.Covered != 2 || r.Total != 2 || r.Percent != 100 {
		t.Fatalf("covered should be capped at total: %+v", r)
	}
	empty := StoredCoverage{SpecFile: "s"}.Summary()
	if empty.Percent != 0 || empty.Exercised == nil || len(empty.Exercised) != 0 {
		t.Fatalf("unexpected empty summary: %+v", empty)
	}
}

func TestBuildReport(t *testing.T) {
	stored := StoredCoverage{SpecFile: "s", Exercised: []string{"GET /a"}, TotalRuns: 3, PassedRuns: 2, FailedRuns: 1}
	r := BuildReport(stored, []string{"GET /a", "POST /b", "GET /c"})
	if r.Covered != 1 || r.Total != 3 || !reflect.DeepEqual(r.Missed, []string{"POST /b", "GET /c"}) {
		t.Fatalf("unexpected report: %+v", r)
	}
	if r.Percent < 33.3 || r.Percent > 33.4 {
		t.Fatalf("percent %v", r.Percent)
	}
	if r.TotalRuns != 3 || r.FailedRuns != 1 {
		t.Fatalf("run counts not carried over: %+v", r)
	}
	none := BuildReport(StoredCoverage{SpecFile: "s"}, nil)
	if none.Percent != 0 || none.Exercised == nil {
		t.Fatalf("unexpected empty report: %+v", none)
	}
}

func TestListAll(t *testing.T) {
	ws := t.TempDir()
	if got := ListAll(ws); got != nil {
		t.Fatalf("expected nil for missing dir, got %v", got)
	}

	now := time.Now()
	for _, sc := range []StoredCoverage{
		{SpecFile: "old.yaml", TotalRuns: 1, TotalOps: 1, LastRunAt: now.Add(-time.Hour)},
		{SpecFile: "new.yaml", TotalRuns: 2, TotalOps: 1, LastRunAt: now},
		{SpecFile: "never.yaml", TotalRuns: 0},
	} {
		if err := Save(ws, sc); err != nil {
			t.Fatal(err)
		}
	}
	dir := DirPath(ws)
	if err := os.WriteFile(filepath.Join(dir, "garbage.json"), []byte("nope"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.json"), 0755); err != nil {
		t.Fatal(err)
	}

	got := ListAll(ws)
	if len(got) != 2 || got[0].SpecFile != "new.yaml" || got[1].SpecFile != "old.yaml" {
		t.Fatalf("unexpected list: %+v", got)
	}
}
