package application_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const spec = `
openapi: "3.0.3"
info: {title: t, version: "1"}
paths:
  /ok:
    get:
      operationId: getOk
      tags: [a]
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
  /bad:
    get:
      operationId: getBad
      tags: [b]
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [id], properties: {id: {type: integer}}}
`

func server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bad" {
			fmt.Fprint(w, `{"id":"x"}`)
			return
		}
		fmt.Fprint(w, `{"id":1}`)
	}))
}

func run(t *testing.T, in application.ConformInput) application.ConformOutput {
	t.Helper()
	s, err := openapi.ParseBytes([]byte(spec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	srv := server()
	t.Cleanup(srv.Close)
	in.Exec, in.Spec, in.BaseURL = executor.New(executor.Options{}), s, srv.URL
	return application.RunConformance(context.Background(), in)
}

func TestRunConformance_FailsOnViolation(t *testing.T) {
	out := run(t, application.ConformInput{Coverage: true})
	if !out.Failed || len(out.Violations) != 1 || out.Coverage == nil || out.Negative != nil || out.Diff != nil {
		t.Fatalf("unexpected output: failed=%v violations=%d cov=%v neg=%v diff=%v",
			out.Failed, len(out.Violations), out.Coverage != nil, out.Negative != nil, out.Diff != nil)
	}
}

func TestRunConformance_TagFilterAndNegative(t *testing.T) {
	out := run(t, application.ConformInput{Tags: []string{"a"}, Negative: true})
	if out.Failed {
		t.Fatalf("only the conforming operation should run: %+v", out.Violations)
	}
	if out.Negative == nil {
		t.Fatal("negative probes were requested")
	}
}

func TestRunConformance_BaselineAcceptsKnownViolations(t *testing.T) {
	first := run(t, application.ConformInput{})
	base := conformance.NewBaseline(first.Violations)

	out := run(t, application.ConformInput{Baseline: base})
	if out.Failed || out.Diff == nil || len(out.Diff.Known) != 1 || len(out.Diff.New) != 0 {
		t.Fatalf("baselined violation must not fail the run: failed=%v diff=%+v", out.Failed, out.Diff)
	}

	empty := conformance.NewBaseline(nil)
	if !run(t, application.ConformInput{Baseline: empty}).Failed {
		t.Fatal("a violation absent from the baseline must fail the run")
	}
}

func TestSplitList(t *testing.T) {
	got := application.SplitList(" a, b ,,c ")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("got %v", got)
	}
	if application.SplitList("") != nil {
		t.Fatal("empty input should give nil")
	}
}

func TestRunConformance_ReadOnlySendsNoWrites(t *testing.T) {
	var writes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writes++
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer srv.Close()
	s, err := openapi.ParseBytes([]byte(`
openapi: "3.0.3"
info: {title: t, version: "1"}
paths:
  /x:
    get:
      responses: {"200": {description: ok}}
    delete:
      responses: {"200": {description: ok}}
`), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	in := application.ConformInput{Exec: executor.New(executor.Options{}), Spec: s, BaseURL: srv.URL, Negative: true, ReadOnly: true}
	out := application.RunConformance(context.Background(), in)
	if writes != 0 {
		t.Fatalf("read-only run sent %d write request(s)", writes)
	}
	_, _, skipped := out.Result.Counts()
	if skipped != 1 {
		t.Fatalf("the DELETE operation should be skipped, skipped = %d", skipped)
	}

	in.ReadOnly = false
	application.RunConformance(context.Background(), in)
	if writes == 0 {
		t.Fatal("without ReadOnly the DELETE operation should be sent")
	}
}

func TestRunConformance_ReportsProgressForEveryOperation(t *testing.T) {
	var done []int
	var totals []int
	var seen []string
	out := run(t, application.ConformInput{
		ReadOnly: false,
		Tags:     []string{"a"}, // /bad is skipped but still counts as handled
		Progress: func(d, total int, op conformance.OperationResult) {
			done, totals = append(done, d), append(totals, total)
			seen = append(seen, op.Method+" "+op.Path)
		},
	})
	if out.Failed {
		t.Fatalf("unexpected failure: %+v", out.Violations)
	}
	if len(done) != 2 || done[0] != 1 || done[1] != 2 || totals[0] != 2 || totals[1] != 2 {
		t.Fatalf("progress = done %v of %v (%v)", done, totals, seen)
	}
}

// A check that is only a warning does not fail the run unless FailOn says so. Nothing in the
// default severities is a warning yet, so the test lowers one.
func TestRunConformance_WarningsDoNotFailUnlessAsked(t *testing.T) {
	t.Cleanup(checks.WithSeverity("CS-RESP-006", checks.Warning))

	out := run(t, application.ConformInput{})
	if out.Failed || out.Warnings != 1 || len(out.Violations) != 0 {
		t.Fatalf("default: failed=%v warnings=%d violations=%d, want a pass with one warning", out.Failed, out.Warnings, len(out.Violations))
	}
	if p, f, _ := out.Result.Counts(); p != 2 || f != 0 {
		t.Fatalf("counts passed=%d failed=%d, want 2 and 0", p, f)
	}

	out = run(t, application.ConformInput{FailOn: checks.Warning})
	if !out.Failed || out.Warnings != 0 || len(out.Violations) != 1 {
		t.Fatalf("fail on warning: failed=%v warnings=%d violations=%d, want a failure", out.Failed, out.Warnings, len(out.Violations))
	}
	if _, f, _ := out.Result.Counts(); f != 1 {
		t.Fatalf("failed count = %d, want 1", f)
	}
}

func TestRunConformance_BaselineFollowsFailOn(t *testing.T) {
	t.Cleanup(checks.WithSeverity("CS-RESP-006", checks.Warning))
	empty := conformance.NewBaseline(nil)

	if out := run(t, application.ConformInput{Baseline: empty}); out.Failed {
		t.Fatal("a warning must not count as a new violation when warnings do not fail the run")
	}
	if out := run(t, application.ConformInput{Baseline: empty, FailOn: checks.Warning}); !out.Failed {
		t.Fatal("with --fail-on warning the warning is a new violation")
	}
}

func TestRunConformance_OnlyOperations(t *testing.T) {
	all := run(t, application.ConformInput{})
	if p, f, s := all.Result.Counts(); p != 1 || f != 1 || s != 0 {
		t.Fatalf("baseline of the test: %d %d %d", p, f, s)
	}

	only := run(t, application.ConformInput{OnlyOperations: map[string]bool{"GET /ok": true}})
	if p, f, s := only.Result.Counts(); p != 1 || f != 0 || s != 1 || only.Failed {
		t.Errorf("only /ok is sent: passed=%d failed=%d skipped=%d failed=%v", p, f, s, only.Failed)
	}

	none := run(t, application.ConformInput{OnlyOperations: map[string]bool{}, Tags: []string{"a"}})
	if _, f, _ := none.Result.Counts(); f != 0 || none.Failed || none.NothingSelected != "" {
		t.Errorf("an empty set selects nothing and passes, even with a tag filter that would otherwise be an error: %+v", none)
	}
}
