package variables

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

var fixedNow = time.Date(2026, 1, 31, 10, 20, 30, 123_000_000, time.UTC)

func testEngine(seed int64) *Engine {
	return NewEngine(seed).WithClock(func() time.Time { return fixedNow })
}

func run(t *testing.T, e *Engine, inner string) string {
	t.Helper()
	out, err := e.Call(inner)
	if err != nil {
		t.Fatalf("%s: %v", inner, err)
	}
	return out
}

func TestTokenizeAndParseCall(t *testing.T) {
	c, err := parseCall(`date +7d format="dd/MM/yyyy HH:mm" tz=Europe/Istanbul`)
	if err != nil || c.name != "date" || len(c.args) != 1 || c.args[0] != "+7d" ||
		c.opts["format"] != "dd/MM/yyyy HH:mm" || c.opts["tz"] != "Europe/Istanbul" {
		t.Errorf("parsed %+v (%v)", c, err)
	}
	c, _ = parseCall(`randomChoice "two words" 'x y' z`)
	if len(c.args) != 3 || c.args[0] != "two words" || c.args[1] != "x y" || c.args[2] != "z" {
		t.Errorf("quotes keep words together: %+v", c.args)
	}
	c, _ = parseCall(`randomChoice a=b c`)
	if len(c.args) != 2 || len(c.opts) != 0 {
		t.Errorf("a=b is an argument unless a is a known option: %+v", c)
	}
	if _, err := parseCall(`randomChoice "oops`); err == nil {
		t.Error("an unclosed quote is an error")
	}
	if _, err := parseCall(``); err == nil {
		t.Error("a bare $ is an error")
	}
}

func TestRandomFunctionsProduceValuesInRange(t *testing.T) {
	e := testEngine(42)
	guid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for i := 0; i < 200; i++ {
		if g := run(t, e, "$guid"); !guid.MatchString(g) {
			t.Fatalf("not a v4 uuid: %s", g)
		}
		if n, _ := strconv.Atoi(run(t, e, "$randomInt 5 9")); n < 5 || n > 9 {
			t.Fatalf("randomInt out of range: %d", n)
		}
		if n, _ := strconv.Atoi(run(t, e, "$randomInt")); n < 0 || n > 1000 {
			t.Fatalf("default randomInt out of range: %d", n)
		}
		if f, _ := strconv.ParseFloat(run(t, e, "$randomFloat 1.5 2.5 3"), 64); f < 1.5 || f > 2.5 {
			t.Fatalf("randomFloat out of range: %v", f)
		}
	}
	if got := run(t, e, "$randomFloat 10 10 1"); got != "10.0" {
		t.Errorf("decimals are honoured: %q", got)
	}
	if got := run(t, e, "$randomFloat decimals=0"); got != "0" && got != "1" {
		t.Errorf("0 decimals of 0..1: %q", got)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{12}$`).MatchString(run(t, e, "$randomString 12")) ||
		!regexp.MustCompile(`^[A-Za-z]{6}$`).MatchString(run(t, e, "$randomAlpha 6")) ||
		!regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(run(t, e, "$randomHex 16")) {
		t.Error("random text uses the documented character sets and lengths")
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		seen[run(t, e, `$randomChoice red green "dark blue"`)] = true
		if b := run(t, e, "$randomBool"); b != "true" && b != "false" {
			t.Fatalf("randomBool = %q", b)
		}
	}
	if len(seen) != 3 || !seen["dark blue"] {
		t.Errorf("randomChoice should pick every option, saw %v", seen)
	}
	if run(t, e, "$uuid") == run(t, e, "$uuid") {
		t.Error("two uuids should differ")
	}
}

func TestSameSeedReplaysAndDerivedEnginesAreIndependent(t *testing.T) {
	seq := func(e *Engine) string {
		var parts []string
		for i := 0; i < 5; i++ {
			parts = append(parts, run(t, e, "$guid"), run(t, e, "$randomInt 1 1000000"), run(t, e, "$randomString 10"))
		}
		return strings.Join(parts, "|")
	}
	if seq(testEngine(7)) != seq(testEngine(7)) {
		t.Error("the same seed must replay the same values")
	}
	if seq(testEngine(7)) == seq(testEngine(8)) {
		t.Error("different seeds should differ")
	}

	runEng := testEngine(99)
	a1, a2 := seq(runEng.Derive("scenarios/a.yaml")), seq(testEngine(99).Derive("scenarios/a.yaml"))
	if a1 != a2 {
		t.Error("a scenario's values depend only on the seed and its key")
	}
	// order independence: deriving b first must not change a
	order := testEngine(99)
	_ = seq(order.Derive("scenarios/b.yaml"))
	if seq(order.Derive("scenarios/a.yaml")) != a1 {
		t.Error("values must not depend on which scenario ran first")
	}
	if seq(testEngine(99).Derive("scenarios/a.yaml")) == seq(testEngine(99).Derive("scenarios/b.yaml")) {
		t.Error("different scenarios get different streams")
	}
}

func TestUsedRandomness(t *testing.T) {
	e := testEngine(1)
	run(t, e, "$date")
	run(t, e, "$timestamp")
	if e.UsedRandomness() {
		t.Error("time functions do not use the generator")
	}
	d := e.Derive("x")
	run(t, d, "$randomBool")
	if !e.UsedRandomness() || !d.UsedRandomness() {
		t.Error("drawing from a derived engine marks the whole run")
	}
	if e.Seed() != 1 {
		t.Errorf("seed = %d", e.Seed())
	}
}

func TestRandomSeedsAreDifferentAndNonNegative(t *testing.T) {
	a, b := RandomSeed(), RandomSeed()
	if a == b || a < 0 || b < 0 {
		t.Errorf("seeds = %d, %d", a, b)
	}
}

func TestTimeFunctions(t *testing.T) {
	e := testEngine(1)
	cases := map[string]string{
		"$timestamp":                            "1769854830",
		"$timestampMs":                          "1769854830123",
		"$isoTimestamp":                         "2026-01-31T10:20:30.123Z",
		"$date":                                 "2026-01-31",
		"$date +7d":                             "2026-02-07",
		"$date -1d":                             "2026-01-30",
		"$date +1w":                             "2026-02-07",
		"$date +1y":                             "2027-01-31",
		"$date -1y":                             "2025-01-31",
		"$date +1M":                             "2026-03-03", // 31 Jan + 1 month overflows February
		"$date format=dd/MM/yyyy":               "31/01/2026",
		"$date format=yy-M":                     "26-1",
		`$date format="MMM d, yyyy"`:            "Jan 31, 2026",
		`$date format="EEEE, d MMMM yyyy"`:      "Saturday, 31 January 2026",
		"$date format=iso":                      "2026-01-31T10:20:30Z",
		"$date format=unix":                     "1769854830",
		"$date format=unixms":                   "1769854830123",
		"$date format=rfc1123":                  "Sat, 31 Jan 2026 10:20:30 UTC",
		"$datetime":                             "2026-01-31T10:20:30Z",
		"$datetime +2h":                         "2026-01-31T12:20:30Z",
		"$datetime -30m":                        "2026-01-31T09:50:30Z",
		"$datetime +45s":                        "2026-01-31T10:21:15Z",
		`$datetime format="yyyy-MM-dd'T'HH:mm"`: "2026-01-31T10:20",
		"$datetime format=yyyy-MM-dd'T'HH:mm:ss.SSS": "",
		"$time":                           "10:20:30",
		"$time +1h format=HH:mm":          "11:20",
		"$datetime tz=Europe/Istanbul":    "2026-01-31T13:20:30+03:00",
		"$date +1d tz=Pacific/Kiritimati": "2026-02-02", // UTC+14: already 1 Feb locally
		"$time tz=utc":                    "10:20:30",
	}
	for in, want := range cases {
		if want == "" {
			continue // only checked for "no error" below
		}
		if got := run(t, e, in); got != want {
			t.Errorf("%s = %q, want %q", in, got, want)
		}
	}
	if _, err := e.Call(`$datetime format="yyyy-MM-dd'T'HH:mm:ss.SSS"`); err != nil {
		t.Errorf("a quoted literal T and milliseconds are valid: %v", err)
	}
	if got := run(t, e, `$datetime format="HH:mm:ss.SSS"`); got != "10:20:30.123" {
		t.Errorf("milliseconds: %q", got)
	}
}

func TestMonthOffsetsAroundYearEndsAndLeapDays(t *testing.T) {
	leap := time.Date(2024, 2, 29, 8, 0, 0, 0, time.UTC)
	e := NewEngine(1).WithClock(func() time.Time { return leap })
	if got := run(t, e, "$date +1y"); got != "2025-03-01" {
		t.Errorf("a leap day plus a year: %q", got)
	}
	if got := run(t, e, "$date -2M"); got != "2023-12-29" {
		t.Errorf("crossing a year end backwards: %q", got)
	}
}

func TestCallErrors(t *testing.T) {
	e := testEngine(1)
	cases := map[string]string{
		"$nope":                 "unknown function $nope",
		"$randomIntt 1 2":       "unknown function",
		"$randomInt 5":          "give both a minimum and a maximum",
		"$randomInt 9 5":        "larger than the maximum",
		"$randomInt a b":        "whole numbers",
		"$randomInt 1 2 3":      "wrong number of arguments",
		"$randomString":         "wrong number of arguments",
		"$randomString 0":       "from 1 to",
		"$randomString 99999":   "from 1 to",
		"$randomString abc":     "from 1 to",
		"$randomChoice":         "wrong number of arguments",
		"$randomFloat 1":        "give a minimum and a maximum",
		"$randomFloat 5 1":      "must not be larger",
		"$randomFloat 1 2 99":   "from 0 to 15",
		"$randomFloat x y":      "must be numbers",
		"$guid extra":           "wrong number of arguments",
		"$date 7d":              "is not an offset",
		"$date +7x":             "is not an offset",
		"$date +1d +2d":         "wrong number of arguments",
		"$date format=yyyy-qq":  "not a known token",
		`$date format="yyyy'T`:  "unclosed quote",
		"$date tz=Mars/Base":    "unknown time zone",
		"$randomInt format=iso": "does not take format=",
		"$timestamp tz=utc":     "does not take tz=",
		"$date +999999999d":     "too large",
	}
	for in, want := range cases {
		_, err := e.Call(in)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want an error containing %q, got %v", in, want, err)
		}
	}
	if _, err := e.Call("guid"); err == nil {
		t.Error("a call without the $ is not a function call")
	}
}

func TestResolverScopes(t *testing.T) {
	env := &domain.Environment{Variables: []domain.EnvironmentVariable{
		{Name: "runId", Value: "run-{{$randomString 6}}"},
		{Name: "base", Value: "http://x"},
		{Name: "url", Value: "{{base}}/{{runId}}"}, // uses earlier environment variables
		{Name: "later", Value: "{{notYet}}"},       // refers to something defined afterwards
		{Name: "notYet", Value: "v"},
	}}
	r := NewResolverWith(env, testEngine(5))

	runID := r.Interpolate("{{runId}}")
	if !strings.HasPrefix(runID, "run-") || len(runID) != 10 {
		t.Fatalf("an environment value may use functions: %q", runID)
	}
	if r.Interpolate("{{runId}}") != runID {
		t.Error("an environment function is generated once, then stable")
	}
	if got := r.Interpolate("{{url}}"); got != "http://x/"+runID {
		t.Errorf("an environment value may use earlier variables: %q", got)
	}
	if got := r.Interpolate("{{later}}"); got != "{{notYet}}" {
		t.Errorf("a reference to a later variable is left as written: %q", got)
	}

	r.Define([]domain.EnvironmentVariable{
		{Name: "orderNo", Value: "{{$randomInt 1000 9999}}"},
		{Name: "otherNo", Value: "{{$randomInt 1000 9999}}"},
		{Name: "label", Value: "order-{{orderNo}}-{{runId}}"},
		{Name: "fromEnv", Value: "${CS_DYN_TEST_ENV}"},
	})
	order := r.Interpolate("{{orderNo}}")
	if order == "" || order == "{{orderNo}}" {
		t.Fatalf("scenario variable not defined: %q", order)
	}
	if got := r.Interpolate("{{orderNo}}/{{orderNo}}"); got != order+"/"+order {
		t.Errorf("a scenario variable is the same everywhere: %q", got)
	}
	if got := r.Interpolate("{{label}}"); got != "order-"+order+"-"+runID {
		t.Errorf("a scenario variable may use earlier scenario and environment variables: %q", got)
	}
	// inline calls generate every time
	distinct := map[string]bool{}
	for i := 0; i < 30; i++ {
		distinct[r.Interpolate("{{$randomInt 1 1000000}}")] = true
	}
	if len(distinct) < 25 {
		t.Errorf("inline calls should be independent, got %d distinct of 30", len(distinct))
	}
	if r.Interpolate("{{$guid}}-{{$guid}}") == r.Interpolate("{{$guid}}-{{$guid}}") {
		t.Error("each inline call differs")
	}

	// runtime values (extractions) win over scenario and environment variables
	r.Set("orderNo", "from-extraction")
	if r.Interpolate("{{orderNo}}") != "from-extraction" {
		t.Error("an extracted value takes precedence")
	}
}

func TestScenarioVariablesAreIndependentPerEngine(t *testing.T) {
	vars := []domain.EnvironmentVariable{{Name: "email", Value: "u-{{$randomString 8}}@x.com"}}
	runEng := testEngine(3)
	a := NewResolverWith(nil, runEng.Derive("a.yaml"))
	b := NewResolverWith(nil, runEng.Derive("b.yaml"))
	a.Define(vars)
	b.Define(vars)
	if a.Interpolate("{{email}}") == b.Interpolate("{{email}}") {
		t.Error("the same definition gives each scenario its own value")
	}
	a2 := NewResolverWith(nil, testEngine(3).Derive("a.yaml"))
	a2.Define(vars)
	if a2.Interpolate("{{email}}") != a.Interpolate("{{email}}") {
		t.Error("with the same seed a scenario's value replays")
	}
}

func TestResolverCollectsErrorsAndLeavesTheTextAlone(t *testing.T) {
	r := NewResolverWith(nil, testEngine(1))
	got := r.Interpolate("a {{$nope}} b {{$randomInt 1 2}} c {{$randomInt x}}")
	if !strings.HasPrefix(got, "a {{$nope}} b ") || !strings.HasSuffix(got, " c {{$randomInt x}}") {
		t.Errorf("failed calls stay as written: %q", got)
	}
	errs := r.TakeErrors()
	if len(errs) != 2 {
		t.Fatalf("want 2 errors, got %v", errs)
	}
	if len(r.TakeErrors()) != 0 {
		t.Error("TakeErrors clears the list")
	}

	bad := NewResolverWith(&domain.Environment{Variables: []domain.EnvironmentVariable{{Name: "x", Value: "{{$bogus}}"}}}, testEngine(1))
	if len(bad.TakeErrors()) != 1 {
		t.Error("an error in an environment value is reported too")
	}
}

func TestCheck(t *testing.T) {
	if errs := Check(`{{baseUrl}} {{ $guid }} {{$randomInt 1 9}} {{$date +7d format=yyyy-MM-dd tz=utc}} {{plain}}`); len(errs) != 0 {
		t.Errorf("valid text has no errors: %v", errs)
	}
	errs := Check(`{{$randomIntt 1 2}} and {{$date +7x}} and {{$randomInt 5}}`)
	if len(errs) != 3 {
		t.Fatalf("want 3 errors, got %v", errs)
	}
	for _, e := range errs {
		if !strings.Contains(e.Error(), "{{$") {
			t.Errorf("an error should quote the call: %v", e)
		}
	}
	if len(Check("no braces at all")) != 0 {
		t.Error("text without calls has no errors")
	}
}

func TestFunctionNamesAreListedAndSorted(t *testing.T) {
	names := FunctionNames()
	if len(names) < 15 || names[0] > names[1] {
		t.Errorf("names = %v", names)
	}
}
