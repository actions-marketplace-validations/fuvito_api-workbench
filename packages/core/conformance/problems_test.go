package conformance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// Several problems in one OpenAPI 3.1 response. The library finds them while walking Go maps, so
// anything that depends on their order shows up only over many runs.
const manyRuns = 60

// the response the testdata baselines were recorded against with the released 0.13.0, 0.13.1, 0.13.2
const manyBody = `{"status":"off","count":"many","user":{"age":"x","id":"y"},"extra":1,"more":2}`

func manySpec(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/problems-31.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func runAnswer(t *testing.T, spec, body string, runs int, check func(*conformance.ConformResult)) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	parsed, err := openapi.ParseBytes([]byte(spec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	runner := conformance.New(executor.New(executor.Options{}))
	for range runs {
		check(runner.Run(context.Background(), parsed, srv.URL, nil, conformance.Options{}))
		if t.Failed() {
			return
		}
	}
}

func record(t *testing.T, spec, body string) *conformance.Baseline {
	t.Helper()
	var b *conformance.Baseline
	runAnswer(t, spec, body, 1, func(r *conformance.ConformResult) {
		// through the file, as a team would have it
		data, err := conformance.NewBaseline(conformance.Violations(r, nil)).Marshal("b.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if b, err = conformance.ParseBaseline(data, "b.yaml"); err != nil {
			t.Fatal(err)
		}
	})
	return b
}

func messages(r *conformance.ConformResult) string {
	var out []string
	for _, op := range r.Operations {
		for _, v := range op.Violations {
			out = append(out, v.Error())
		}
	}
	return strings.Join(out, "\n--\n")
}

func summaries(violations []conformance.Violation) string {
	var out []string
	for _, v := range violations {
		line, _, _ := strings.Cut(v.Detail, "\n")
		out = append(out, line)
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}

// answering is a 3.1 spec whose GET /thing declares schema for its 200 answer.
func answering(schema string) string {
	return `{"openapi":"3.1.0","info":{"title":"a","version":"1"},"paths":{"/thing":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":` + schema + `}}}}}}}}`
}

const (
	integer = `{"type":"integer"}`
	allOf   = `{"allOf":[{"type":"object","properties":{"a":` + integer + `,"b":` + integer + `}},{"type":"object","properties":{"c":` + integer + `}}]}`
	anyOf   = `{"type":"object","properties":{"v":{"anyOf":[` + integer + `,{"type":"object","properties":{"m":` + integer + `,"n":` + integer + `,"o":` + integer + `}}]}}}`
	aList   = `{"type":"object","properties":{"l":{"type":"array","items":{"type":"object","properties":{"p":` + integer + `,"q":` + integer + `}}}}}`
)

var composed = map[string]struct{ spec, body string }{
	"several fields": {"", manyBody},
	"allOf":          {answering(allOf), `{"a":"x","b":"y","c":"z"}`},
	"anyOf with several bad fields in a branch": {answering(anyOf), `{"v":{"m":"x","n":"y","o":"z"}}`},
	"a list": {answering(aList), `{"l":[{"p":"a","q":"b"},{"p":"c","q":"d"}]}`},
}

func TestSeveralProblemsAreReportedTheSameWayEveryRun(t *testing.T) {
	for name, c := range composed {
		t.Run(name, func(t *testing.T) {
			if c.spec == "" {
				c.spec = manySpec(t)
			}
			first := ""
			runAnswer(t, c.spec, c.body, manyRuns, func(r *conformance.ConformResult) {
				if first == "" {
					first = messages(r)
				} else if got := messages(r); got != first {
					t.Errorf("another run, another report:\n%s\n=== first\n%s", got, first)
				}
			})
		})
	}
}

func TestEveryProblemIsAViolationOfItsOwn(t *testing.T) {
	runAnswer(t, manySpec(t), manyBody, 1, func(r *conformance.ConformResult) {
		got := summaries(conformance.Violations(r, nil))
		want := strings.Join([]string{
			`at '': additional properties 'extra', 'more' not allowed`,
			`at '': missing properties 'name', 'owner'`,
			`error at "/count": at '/count': got string, want integer`,
			`error at "/status": at '/status': value must be 'on'`,
			`error at "/user/user/age": at '/user/age': got string, want integer`,
			`error at "/user/user/id": at '/user/id': got string, want integer`,
		}, "\n")
		if got != want {
			t.Errorf("violations:\n%s\nwant:\n%s", got, want)
		}
	})
}

func TestAnUnchangedAnswerAlwaysMatchesItsBaseline(t *testing.T) {
	for name, c := range composed {
		t.Run(name, func(t *testing.T) {
			if c.spec == "" {
				c.spec = manySpec(t)
			}
			b := record(t, c.spec, c.body)
			runAnswer(t, c.spec, c.body, manyRuns, func(r *conformance.ConformResult) {
				if d := conformance.Compare(conformance.Violations(r, nil), b); len(d.New) != 0 || len(d.Fixed) != 0 || len(d.Known) == 0 {
					t.Errorf("known %d, new %d, fixed %d\n%s", len(d.Known), len(d.New), len(d.Fixed), d.Text())
				}
			})
		})
	}
}

// The failure of 0.13.2: a problem added to a response that already had a recorded one counted as
// known, and the build passed.
func TestAProblemAddedToARecordedResponseIsAlwaysNew(t *testing.T) {
	spec := answering(`{"type":"object","required":["name"],"additionalProperties":false,"properties":{"name":{"type":"string"},"status":{"type":"string","enum":["on"]},"count":` + integer + `,"user":{"type":"object","properties":{"age":` + integer + `}}}}`)
	recorded := `{"name":"n","status":"off"}`
	for name, c := range map[string]struct {
		body, added string
	}{
		"another field":             {`{"name":"n","status":"off","count":"many"}`, `at '/count'`},
		"the same constraint twice": {`{"name":"n","status":"off","user":{"age":"x"}}`, `at '/user/age'`},
		"a field goes missing":      {`{"status":"off"}`, `missing propert`},
		"an extra property":         {`{"name":"n","status":"off","extra":1}`, `'extra' not allowed`},
	} {
		t.Run(name, func(t *testing.T) {
			b := record(t, spec, recorded)
			runAnswer(t, spec, c.body, manyRuns, func(r *conformance.ConformResult) {
				d := conformance.Compare(conformance.Violations(r, nil), b)
				if len(d.New) != 1 || !strings.Contains(d.New[0].Detail, c.added) || len(d.Known) != 1 || len(d.Fixed) != 0 {
					t.Errorf("known %d, new %d, fixed %d\n%s", len(d.Known), len(d.New), len(d.Fixed), d.Text())
				}
			})
		})
	}
}

// The library writes the path of a nested error differently when it has neighbours. Fixing one of
// them must not make the other look new.
func TestFixingOneOfTwoNestedProblemsLeavesTheOtherKnown(t *testing.T) {
	spec := manySpec(t)
	base := `{"name":"n","owner":"o","user":{"age":"x","id":"y"}}`
	for _, recorded := range []string{base, `{"name":"n","owner":"o","user":{"age":"x","id":1}}`} {
		b := record(t, spec, recorded)
		runAnswer(t, spec, `{"name":"n","owner":"o","user":{"age":"x","id":1}}`, 20, func(r *conformance.ConformResult) {
			for _, v := range conformance.Violations(r, nil) {
				if !strings.Contains(v.Detail, "/user/age") {
					continue
				}
				if d := conformance.Compare([]conformance.Violation{v}, b); len(d.New) != 0 {
					t.Errorf("recorded from %s:\n%s", recorded, d.Text())
				}
			}
		})
	}
}

// The errors inside an anyOf explain one problem together: it is known while it says nothing that
// was not recorded.
func TestAGroupedProblemIsNewWhenItSaysSomethingNew(t *testing.T) {
	spec := answering(anyOf)
	two := record(t, spec, `{"v":{"m":"x","n":"y"}}`)
	for body, wantNew := range map[string]int{
		`{"v":{"m":"x","n":"y"}}`:         0,
		`{"v":{"m":"x","n":2}}`:           0,
		`{"v":{"m":"x","n":"y","o":"z"}}`: 1,
	} {
		runAnswer(t, spec, body, 20, func(r *conformance.ConformResult) {
			if d := conformance.Compare(conformance.Violations(r, nil), two); len(d.New) != wantNew || len(d.Fixed) != 0 {
				t.Errorf("%s: new %d (want %d), fixed %d\n%s", body, len(d.New), wantNew, len(d.Fixed), d.Text())
			}
		})
	}
}

func released(t *testing.T, version string) *conformance.Baseline {
	t.Helper()
	name := "testdata/baseline-31-by-" + version + ".yaml"
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	b, err := conformance.ParseBaseline(data, name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// 0.13.0 and 0.13.1 stored the whole message: every problem in it stays known.
func TestBaselinesWrittenBy0130And0131StillMatchSeveralProblems(t *testing.T) {
	for _, version := range []string{"0.13.0", "0.13.1"} {
		t.Run(version, func(t *testing.T) {
			b := released(t, version)
			runAnswer(t, manySpec(t), manyBody, manyRuns, func(r *conformance.ConformResult) {
				if d := conformance.Compare(conformance.Violations(r, nil), b); len(d.Known) != 6 || len(d.New) != 0 || len(d.Fixed) != 0 {
					t.Errorf("known %d, new %d, fixed %d\n%s", len(d.Known), len(d.New), len(d.Fixed), d.Text())
				}
			})
		})
	}
}

// 0.13.2 stored only the start of the message: here two of the six problems. The other four were
// never recorded, so they are new until the baseline is recorded again.
func TestABaselineWrittenBy0132MatchesOnlyWhatItRecorded(t *testing.T) {
	b := released(t, "0.13.2")
	runAnswer(t, manySpec(t), manyBody, manyRuns, func(r *conformance.ConformResult) {
		d := conformance.Compare(conformance.Violations(r, nil), b)
		want := `at '': missing properties 'name', 'owner'` + "\n" + `error at "/status": at '/status': value must be 'on'`
		if got := summaries(d.Known); got != want || len(d.New) != 4 || len(d.Fixed) != 0 {
			t.Errorf("known:\n%s\nnew %d, fixed %d", got, len(d.New), len(d.Fixed))
		}
	})
}

// A version 1 baseline (0.8.9 and earlier) is compared by its wording, problem by problem too.
func TestAVersion1BaselineWithSeveralProblemsInOneEntry(t *testing.T) {
	v1 := `schemaVersion: "1"
violations:
  - operation: GET /thing
    kind: response
    detail: "validation failed due to: error at \"/b\": at '/b': got string, want integer\nSchema:\n  {}\n\nValue:\n  \"y\"\n | error at \"/a\": at '/a': got string, want integer\nSchema:\n  {}\n\nValue:\n  \"x\"\n"
`
	b, err := conformance.ParseBaseline([]byte(v1), "old.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec := answering(allOf)
	for body, wantNew := range map[string]int{`{"a":"x","b":"y"}`: 0, `{"a":"x"}`: 0, `{"a":"x","b":"y","c":"z"}`: 1} {
		runAnswer(t, spec, body, 20, func(r *conformance.ConformResult) {
			if d := conformance.Compare(conformance.Violations(r, nil), b); len(d.New) != wantNew || len(d.Fixed) != 0 {
				t.Errorf("%s: new %d (want %d), fixed %d\n%s", body, len(d.New), wantNew, len(d.Fixed), d.Text())
			}
		})
	}
}

// Pins CS-GAP-019: some OpenAPI 3.1 sentences hold the failing value. When that value differs on
// every run, so does the violation, and the baseline reports it as new (it fails; nothing is hidden).
func TestBaseline_AChangingValueInsideTheSentenceIsNewEachTime_KnownGap(t *testing.T) {
	spec := answering(`{"type":"object","properties":{"ref":{"type":"string","pattern":"^[0-9]+$"}}}`)
	b := record(t, spec, `{"ref":"ab-one"}`)
	runAnswer(t, spec, `{"ref":"ab-two"}`, 1, func(r *conformance.ConformResult) {
		if d := conformance.Compare(conformance.Violations(r, nil), b); len(d.New) != 1 || len(d.Fixed) != 1 {
			t.Errorf("the value no longer changes the identity: update CS-GAP-019\n%s", d.Text())
		}
	})
}
