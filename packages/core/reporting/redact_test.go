package reporting_test

import (
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestIsSecretName(t *testing.T) {
	secret := []string{"token", "accessToken", "access_token", "ACCESS_TOKEN", "apiKey", "api-key", "API_KEY", "password",
		"userPassword", "passwd", "pwd_passphrase", "secret", "clientSecret", "authorization", "auth", "authToken",
		"sessionId", "session", "cookie", "jwt", "bearer", "credentials", "privateKey", "key"}
	for _, n := range secret {
		if !reporting.IsSecretName(n) {
			t.Errorf("%q should be treated as secret", n)
		}
	}
	plain := []string{"userId", "id", "orderId", "author", "authorName", "monkey", "keyboard", "email", "name", "count", "next", ""}
	for _, n := range plain {
		if reporting.IsSecretName(n) {
			t.Errorf("%q should not be treated as secret", n)
		}
	}
}

// secretResult has a secret env value in an error, an assertion message and actual value,
// plus extracted variables with secret and ordinary names.
func secretResult() *scenario.Result {
	return &scenario.Result{
		Scenario: makeScenario("Auth"),
		Steps: []scenario.StepResult{{
			Label: "Login", Passed: false,
			Error: "request to https://api/?key=hunter2-secret failed",
			Assertions: []domain.AssertionResult{{
				Assertion: domain.Assertion{Type: domain.AssertionJSONPath, Path: "$.token", Operator: domain.OperatorEq, Expected: "x"},
				Passed:    false,
				Actual:    "hunter2-secret",
				Message:   "got hunter2-secret, want x",
			}},
			Extracted: map[string]string{"token": "eyJhbGciOi", "userId": "42", "note": "contains hunter2-secret inside"},
		}},
		Duration: time.Millisecond,
		Secrets:  []string{"hunter2-secret", "ab"},
	}
}

func TestRedaction_Text(t *testing.T) {
	out := reporting.Text(secretResult())
	if strings.Contains(out, "hunter2-secret") || strings.Contains(out, "eyJhbGciOi") {
		t.Errorf("a secret leaked into the text report:\n%s", out)
	}
	for _, want := range []string{`token = "***"`, `userId = "42"`, `note = "contains *** inside"`, "got ***, want x", "key=***"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRedaction_JSON(t *testing.T) {
	b, err := reporting.JSON(secretResult())
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	if strings.Contains(out, "hunter2-secret") || strings.Contains(out, "eyJhbGciOi") {
		t.Errorf("a secret leaked into the JSON report:\n%s", out)
	}
	for _, want := range []string{`"token": "***"`, `"userId": "42"`, `"actual": "***"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRedaction_JUnitHTMLAndMulti(t *testing.T) {
	r := secretResult()
	junit, err := reporting.JUnit(r)
	if err != nil {
		t.Fatal(err)
	}
	html, err := reporting.HTML(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	multi := reporting.TextMulti("x", []*scenario.Result{r}, nil, time.Millisecond)
	jm, err := reporting.JUnitMulti("x", []*scenario.Result{r})
	if err != nil {
		t.Fatal(err)
	}
	hm, err := reporting.HTMLMulti("x", []*scenario.Result{r}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]string{"junit": string(junit), "html": string(html), "text multi": multi, "junit multi": string(jm), "html multi": string(hm)} {
		if strings.Contains(out, "hunter2-secret") {
			t.Errorf("%s report leaked the secret:\n%s", name, out)
		}
		if !strings.Contains(out, "***") {
			t.Errorf("%s report should show where the secret was removed", name)
		}
	}
}

func TestRedaction_ShortSecretsAreNotReplacedInText(t *testing.T) {
	r := secretResult()
	r.Steps[0].Error = "cab and 42 and abc"
	out := reporting.Text(r)
	if !strings.Contains(out, "cab and 42 and abc") {
		t.Errorf("a 2-character secret must not rewrite ordinary text:\n%s", out)
	}
}

func TestRedaction_RevealExtractedShowsNamesButNeverSecrets(t *testing.T) {
	r := secretResult()
	r.RevealExtracted = true
	out := reporting.Text(r)
	if !strings.Contains(out, `token = "eyJhbGciOi"`) {
		t.Errorf("--show-extracted shows secret-looking names:\n%s", out)
	}
	if strings.Contains(out, "hunter2-secret") {
		t.Errorf("secret environment values are never revealed:\n%s", out)
	}
}

func TestRedaction_LongestSecretWins(t *testing.T) {
	r := secretResult()
	r.Secrets = []string{"abcd", "abcd-efgh"}
	r.Steps[0].Error = "value abcd-efgh here"
	out := reporting.Text(r)
	if !strings.Contains(out, "value *** here") {
		t.Errorf("a secret containing another must be replaced whole:\n%s", out)
	}
}

func TestRedaction_ExtractedAreListedInStableOrder(t *testing.T) {
	r := &scenario.Result{
		Scenario: makeScenario("order"),
		Steps: []scenario.StepResult{{Label: "s", Passed: true,
			Extracted: map[string]string{"zeta": "1", "alpha": "2", "mid": "3"}}},
		Passed: true,
	}
	out := reporting.Text(r)
	if !(strings.Index(out, "alpha") < strings.Index(out, "mid") && strings.Index(out, "mid") < strings.Index(out, "zeta")) {
		t.Errorf("extracted variables should be sorted by name:\n%s", out)
	}
}
