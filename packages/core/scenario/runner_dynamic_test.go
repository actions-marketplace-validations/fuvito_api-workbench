package scenario_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// recorder is a server that remembers every request and echoes the query parameter "e".
type recorder struct {
	srv  *httptest.Server
	mu   sync.Mutex
	seen []string
}

func newRecorder(t *testing.T) *recorder {
	t.Helper()
	rec := &recorder{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.seen = append(rec.seen, r.Method+" "+r.URL.RequestURI()+" "+string(body))
		rec.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"echo":%q}`, r.URL.Query().Get("e"))
	}))
	t.Cleanup(rec.srv.Close)
	return rec
}

func (r *recorder) requests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.seen...)
}

func get(path string) *domain.Request {
	return &domain.Request{Method: domain.MethodGET, URL: "{{base}}" + path}
}

func runner() *scenario.Runner { return scenario.New(executor.New(executor.Options{})) }

func env(rec *recorder) *domain.Environment {
	return &domain.Environment{Variables: []domain.EnvironmentVariable{{Name: "base", Value: rec.srv.URL}}}
}

func orderScenario() *domain.Scenario {
	return &domain.Scenario{
		Title:     "Orders",
		Variables: []domain.EnvironmentVariable{{Name: "email", Value: "user-{{$randomString 10}}@example.com"}},
		Steps: []domain.ScenarioStep{
			{Label: "create", Request: &domain.Request{Method: domain.MethodPOST, URL: "{{base}}/create",
				Body: &domain.RequestBody{Type: domain.BodyRaw, Content: "email={{email}}"}}},
			{Label: "fetch", Request: get("/fetch?e={{email}}&n={{$randomInt 1 1000000}}"),
				Assertions: []domain.Assertion{{Type: domain.AssertionJSONPath, Path: "$.echo", Operator: domain.OperatorEq, Expected: "{{email}}"}}},
		},
	}
}

func emailIn(request string) string {
	request = strings.ReplaceAll(request, "%40", "@")
	i := strings.Index(request, "user-")
	if i < 0 {
		return ""
	}
	return request[i : i+len("user-")+10+len("@example.com")]
}

func TestScenarioVariableIsGeneratedOnceAndUsedEverywhere(t *testing.T) {
	rec := newRecorder(t)
	res := runner().WithEngine(variables.NewEngine(11), "orders.yaml").Run(context.Background(), orderScenario(), env(rec))
	if !res.Passed {
		t.Fatalf("the assertion {{email}} should match what the server echoed: %+v", res.Steps)
	}
	reqs := rec.requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %v", reqs)
	}
	a, b := emailIn(reqs[0]), emailIn(reqs[1])
	if a == "" || a != b {
		t.Errorf("the same scenario variable must be identical in every step: %q vs %q", a, b)
	}
	if len(res.Variables) != 1 || res.Variables[0].Name != "email" || res.Variables[0].Value != a {
		t.Errorf("the run records what was generated: %+v", res.Variables)
	}
	if res.Seed != 11 || !res.UsedRandomness {
		t.Errorf("seed/used = %d/%v", res.Seed, res.UsedRandomness)
	}
	if got := res.Steps[1].Assertions[0].Assertion.Expected; got != a {
		t.Errorf("the assertion shows the resolved value, got %v", got)
	}
}

func TestInlineFunctionsDifferEveryTimeAndAcrossScenarioRuns(t *testing.T) {
	rec := newRecorder(t)
	r := runner().WithEngine(variables.NewEngine(11), "orders.yaml")
	r.Run(context.Background(), orderScenario(), env(rec))
	r.Run(context.Background(), orderScenario(), env(rec))
	reqs := rec.requests()
	if emailIn(reqs[0]) != emailIn(reqs[2]) {
		t.Error("the same engine and key replay the same scenario variable")
	}

	other := newRecorder(t)
	runner().WithEngine(variables.NewEngine(11), "other.yaml").Run(context.Background(), orderScenario(), env(other))
	if emailIn(other.requests()[0]) == emailIn(reqs[0]) {
		t.Error("two scenarios with the same definition get their own values")
	}
}

func TestSameSeedReplaysTheWholeRun(t *testing.T) {
	run := func(seed int64) []string {
		rec := newRecorder(t)
		runner().WithEngine(variables.NewEngine(seed), "orders.yaml").Run(context.Background(), orderScenario(), env(rec))
		out := rec.requests()
		for i := range out {
			out[i] = strings.ReplaceAll(out[i], rec.srv.URL, "")
		}
		return out
	}
	a, b, c := run(5), run(5), run(6)
	if strings.Join(a, "|") != strings.Join(b, "|") {
		t.Errorf("the same seed must send the same requests:\n%v\n%v", a, b)
	}
	if strings.Join(a, "|") == strings.Join(c, "|") {
		t.Error("a different seed sends different values")
	}
}

func TestValuesDoNotDependOnWhichScenarioRunsFirst(t *testing.T) {
	first := func(order ...string) map[string]string {
		got := map[string]string{}
		eng := variables.NewEngine(21)
		for _, key := range order {
			rec := newRecorder(t)
			runner().WithEngine(eng, key).Run(context.Background(), orderScenario(), env(rec))
			got[key] = emailIn(rec.requests()[0])
		}
		return got
	}
	ab, ba := first("a.yaml", "b.yaml"), first("b.yaml", "a.yaml")
	if ab["a.yaml"] != ba["a.yaml"] || ab["b.yaml"] != ba["b.yaml"] {
		t.Errorf("order must not change a scenario's values: %v vs %v", ab, ba)
	}
}

func TestEnvironmentFunctionIsSharedWhenResolvedOncePerCommand(t *testing.T) {
	rec := newRecorder(t)
	rawEnv := &domain.Environment{Variables: []domain.EnvironmentVariable{
		{Name: "base", Value: rec.srv.URL},
		{Name: "runId", Value: "run-{{$randomString 8}}"},
	}}
	runEngine := variables.NewEngine(3)
	resolved, errs := variables.ResolveEnvironment(rawEnv, runEngine.Derive("environment"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	if rawEnv.Variables[1].Value != "run-{{$randomString 8}}" {
		t.Error("the original environment must stay untouched")
	}

	sc := func(title string) *domain.Scenario {
		return &domain.Scenario{Title: title, Steps: []domain.ScenarioStep{{Label: "go", Request: get("/x?e={{runId}}&id={{$guid}}")}}}
	}
	for _, key := range []string{"a.yaml", "b.yaml"} {
		runner().WithEngine(runEngine, key).Run(context.Background(), sc(key), resolved)
	}
	reqs := rec.requests()
	idOf := func(r string) string { return r[strings.Index(r, "run-") : strings.Index(r, "run-")+12] }
	if idOf(reqs[0]) != idOf(reqs[1]) {
		t.Errorf("an environment function is generated once for the whole command: %q vs %q", idOf(reqs[0]), idOf(reqs[1]))
	}
	if reqs[0] == reqs[1] {
		t.Error("inline calls still differ per request")
	}
}

func TestUnknownFunctionStopsTheScenarioBeforeAnythingIsSent(t *testing.T) {
	rec := newRecorder(t)
	sc := &domain.Scenario{
		Title:     "Broken",
		Variables: []domain.EnvironmentVariable{{Name: "x", Value: "{{$randomIntt 1 2}}"}},
		Steps:     []domain.ScenarioStep{{Label: "a", Request: get("/a")}, {Label: "b", Request: get("/b")}},
	}
	res := runner().Run(context.Background(), sc, env(rec))
	if res.Passed || !strings.Contains(res.SetupError, "unknown function $randomIntt") {
		t.Fatalf("passed=%v setup error=%q", res.Passed, res.SetupError)
	}
	if len(rec.requests()) != 0 {
		t.Error("nothing may be sent when the scenario cannot start")
	}
	if len(res.Steps) != 2 || !res.Steps[0].Skipped || !res.Steps[1].Skipped {
		t.Errorf("every step is reported as skipped: %+v", res.Steps)
	}
}

func TestBadFunctionInAStepFailsThatStepWithoutSending(t *testing.T) {
	rec := newRecorder(t)
	sc := &domain.Scenario{Title: "S", Steps: []domain.ScenarioStep{
		{Label: "bad", Request: get("/a?n={{$randomInt 9 1}}"), FailPolicy: domain.FailPolicyContinue},
		{Label: "good", Request: get("/b")},
	}}
	res := runner().Run(context.Background(), sc, env(rec))
	if res.Steps[0].Passed || !strings.Contains(res.Steps[0].Error, "larger than the maximum") {
		t.Errorf("step 1 = %+v", res.Steps[0])
	}
	if reqs := rec.requests(); len(reqs) != 1 || !strings.Contains(reqs[0], "/b") {
		t.Errorf("only the good step is sent: %v", reqs)
	}
}

func TestAssertionPathAndNameAreInterpolatedToo(t *testing.T) {
	rec := newRecorder(t)
	sc := &domain.Scenario{
		Title:     "S",
		Variables: []domain.EnvironmentVariable{{Name: "field", Value: "echo"}, {Name: "want", Value: "hello"}},
		Steps: []domain.ScenarioStep{{Label: "a", Request: get("/a?e={{want}}"),
			Assertions: []domain.Assertion{{Type: domain.AssertionJSONPath, Name: "echoes {{want}}", Path: "$.{{field}}", Operator: domain.OperatorEq, Expected: "{{want}}"}}}},
	}
	res := runner().Run(context.Background(), sc, env(rec))
	if !res.Passed || res.Steps[0].Assertions[0].Assertion.Name != "echoes hello" {
		t.Errorf("result = %+v", res.Steps[0])
	}
	// a literal number is left alone
	sc.Steps[0].Assertions = []domain.Assertion{{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: 200}}
	if !runner().Run(context.Background(), sc, env(rec)).Passed {
		t.Error("non-string expected values are unchanged")
	}
}

func TestSecretScenarioVariablesAreRecordedAsSecrets(t *testing.T) {
	rec := newRecorder(t)
	sc := &domain.Scenario{
		Title:     "S",
		Variables: []domain.EnvironmentVariable{{Name: "token", Value: "tok-{{$randomString 12}}", Secret: true}},
		Steps:     []domain.ScenarioStep{{Label: "a", Request: get("/a?e={{token}}")}},
	}
	res := runner().WithEngine(variables.NewEngine(1), "s").Run(context.Background(), sc, env(rec))
	if len(res.Secrets) != 1 || !strings.HasPrefix(res.Secrets[0], "tok-") || !res.Variables[0].Secret {
		t.Errorf("secrets = %v, variables = %+v", res.Secrets, res.Variables)
	}
}

func TestTimeOnlyScenarioNeedsNoSeed(t *testing.T) {
	rec := newRecorder(t)
	sc := &domain.Scenario{
		Title:     "S",
		Variables: []domain.EnvironmentVariable{{Name: "due", Value: "{{$date +7d}}"}},
		Steps:     []domain.ScenarioStep{{Label: "a", Request: get("/a?e={{due}}")}},
	}
	res := runner().Run(context.Background(), sc, env(rec))
	if res.UsedRandomness {
		t.Error("only the clock was used")
	}
	want := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	if !strings.Contains(rec.requests()[0], "e="+want) {
		t.Errorf("request = %v, want the date %s", rec.requests(), want)
	}
}
