package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const redactionSecret = "s3cr3t-value-1234"

// echoServer answers every request with the secret in its body, the way a misbehaving or
// verbose API might.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"echo":%q,"access_token":"jwt-abcdef-0001","userId":"42"}`, redactionSecret)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func redactionFiles(t *testing.T, srv *httptest.Server) (scenarioPath, envPath string) {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"env.yaml": "schemaVersion: \"1\"\nname: ci\nvariables:\n  - name: baseUrl\n    value: " + srv.URL + "\n  - name: apiKey\n    value: \"${CS_REDACT_SECRET}\"\n    secret: true\n",
		"s.yaml": `schemaVersion: "1"
title: Redaction
steps:
  - label: Call
    request:
      method: GET
      url: "{{baseUrl}}/x"
    extractions:
      - {name: access_token, source: jsonpath, path: $.access_token}
      - {name: userId, source: jsonpath, path: $.userId}
    assertions:
      - {type: jsonpath, path: $.echo, operator: eq, expected: something-else}
`,
	})
	return root + "/s.yaml", root + "/env.yaml"
}

func TestRun_ReportsNeverContainSecrets(t *testing.T) {
	srv := echoServer(t)
	t.Setenv("CS_REDACT_SECRET", redactionSecret)
	sc, env := redactionFiles(t, srv)

	for _, format := range []string{"text", "json", "junit", "html"} {
		out, err := exec(t, "run", "--env", env, "--format", format, sc)
		if exitCode(err) != 1 {
			t.Fatalf("%s: the assertion fails on purpose (exit 1), got %v\n%s", format, err, out)
		}
		if strings.Contains(out, redactionSecret) {
			t.Errorf("%s report leaked the secret environment value:\n%s", format, out)
		}
		if strings.Contains(out, "jwt-abcdef-0001") {
			t.Errorf("%s report leaked an extracted token:\n%s", format, out)
		}
	}

	out, _ := exec(t, "run", "--env", env, sc)
	for _, want := range []string{`access_token = "***"`, `userId = "42"`, "***"} {
		if !strings.Contains(out, want) {
			t.Errorf("text report should contain %q:\n%s", want, out)
		}
	}
}

func TestRun_ShowExtractedRevealsTokensButNotSecretEnvironmentValues(t *testing.T) {
	srv := echoServer(t)
	t.Setenv("CS_REDACT_SECRET", redactionSecret)
	sc, env := redactionFiles(t, srv)

	out, _ := exec(t, "run", "--env", env, "--show-extracted", sc)
	if !strings.Contains(out, `access_token = "jwt-abcdef-0001"`) {
		t.Errorf("--show-extracted should show the extracted token:\n%s", out)
	}
	if strings.Contains(out, redactionSecret) {
		t.Errorf("a secret environment value is never shown:\n%s", out)
	}
}

func TestRun_FolderReportsAreRedactedToo(t *testing.T) {
	srv := echoServer(t)
	t.Setenv("CS_REDACT_SECRET", redactionSecret)
	sc, env := redactionFiles(t, srv)
	dir := sc[:strings.LastIndex(sc, "/")]
	out, _ := exec(t, "run", "--env", env, "--format", "json", dir)
	if strings.Contains(out, redactionSecret) || strings.Contains(out, "jwt-abcdef-0001") {
		t.Errorf("combined report leaked a secret:\n%s", out)
	}
}
