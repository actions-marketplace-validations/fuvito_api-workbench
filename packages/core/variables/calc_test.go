package variables

import (
	"regexp"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func calcWith(t *testing.T, expr string, vars map[string]string) (string, error) {
	t.Helper()
	env := &domain.Environment{}
	for k, v := range vars {
		env.Variables = append(env.Variables, domain.EnvironmentVariable{Name: k, Value: v})
	}
	r := NewResolverWith(env, NewEngine(1))
	out := r.Interpolate("{{$calc " + expr + "}}")
	if errs := r.TakeErrors(); len(errs) > 0 {
		return "", errs[0]
	}
	return out, nil
}

func TestCalcResults(t *testing.T) {
	vars := map[string]string{"price": "19.99", "quantity": "3", "n": "7"}
	for expr, want := range map[string]string{
		"2 + 3":                   "5",
		"2 + 3 * 4":               "14",
		"(2 + 3) * 4":             "20",
		"10 / 4":                  "2.5",
		"10 / 5":                  "2",
		"10 % 4":                  "2",
		"-n + 10":                 "3",
		"0.1 * 3":                 "0.3",
		"price * quantity":        "59.97",
		"round(price * quantity)": "60",
		"round(10 / 3, 2)":        "3.33",
		"floor(2.9) + ceil(2.1)":  "5",
		"abs(-4.5)":               "4.5",
		"min(3, n, 5)":            "3",
		"max(3, n, 5)":            "7",
		"1.5 + 1.5":               "3",
		"9223372036854775807 + 1": "9223372036854776000",
	} {
		got, err := calcWith(t, expr, vars)
		if err != nil || got != want {
			t.Errorf("%s = %q (%v), want %q", expr, got, err, want)
		}
	}
}

func TestCalcErrors(t *testing.T) {
	for expr, want := range map[string]string{
		"1 / 0":        "division by zero",
		"5 % 0":        "division by zero",
		"missing + 1":  "unknown variable",
		"label + 1":    "not a number",
		"(1 + 2":       "missing )",
		"1 +":          "expression ends",
		"2 $ 3":        "unexpected",
		"sqrt(4)":      "unknown function",
		"round(1,2,3)": "arguments",
		"":             "write an expression",
	} {
		_, err := calcWith(t, expr, map[string]string{"label": "abc"})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want it to contain %q", expr, err, want)
		}
	}
}

func TestCalcChecksSyntaxWithoutValues(t *testing.T) {
	if errs := Check("{{$calc price * quantity}}"); len(errs) != 0 {
		t.Errorf("names are not looked up when validating: %v", errs)
	}
	if errs := Check("{{$calc 1 / 0}}"); len(errs) != 1 {
		t.Errorf("a literal division by zero is found at validation: %v", errs)
	}
	if errs := Check("{{$calc (1 + }}"); len(errs) != 1 {
		t.Errorf("bad syntax is found at validation: %v", errs)
	}
}

func TestCalcUsesExtractedAndScenarioValues(t *testing.T) {
	r := NewResolverWith(nil, NewEngine(1))
	r.Define([]domain.EnvironmentVariable{{Name: "a", Value: "4"}, {Name: "b", Value: "{{$calc a * 2}}"}})
	r.Set("c", "10")
	if got := r.Interpolate("{{b}} {{$calc b + c}}"); got != "8 18" || len(r.TakeErrors()) != 0 {
		t.Errorf("got %q", got)
	}
}

func TestFakeData(t *testing.T) {
	r := NewResolverWith(nil, NewEngine(5))
	for _, c := range []struct{ call, re string }{
		{"$randomEmail", `^[a-z]+\.[a-z]+\d{4}@example\.com$`},
		{"$randomFirstName", `^[A-Z][a-z]+$`},
		{"$randomLastName", `^[A-Z][a-z]+$`},
		{"$randomFullName", `^[A-Z][a-z]+ [A-Z][a-z]+$`},
		{"$randomPhone", `^\+1-555-01\d\d$`},
		{"$randomWord", `^[a-z]+$`},
	} {
		got := r.Interpolate("{{" + c.call + "}}")
		if !regexpMatch(c.re, got) {
			t.Errorf("%s = %q, want match %s", c.call, got, c.re)
		}
	}
	a := NewResolverWith(nil, NewEngine(9)).Interpolate("{{$randomEmail}}")
	b := NewResolverWith(nil, NewEngine(9)).Interpolate("{{$randomEmail}}")
	if a != b {
		t.Error("fake data follows the seed")
	}
	if errs := Check("{{$randomEmail 1}}"); len(errs) != 1 {
		t.Error("fake data functions take no arguments")
	}
}

func regexpMatch(re, s string) bool { return regexp.MustCompile(re).MatchString(s) }
