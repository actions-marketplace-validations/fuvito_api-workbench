package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const setupSpec = `openapi: "3.0.3"
info: {title: secured, version: "1"}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
paths:
  /secure:
    get:
      operationId: secure
      security: [{bearerAuth: []}]
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: object, required: [ok], properties: {ok: {type: boolean}}}
        "401": {description: no}
`

const setupLogin = `schemaVersion: "1"
title: Login
steps:
  - label: Log in
    request:
      method: POST
      url: "{{baseUrl}}/login"
    extractions:
      - name: token
        source: jsonpath
        path: $.token
      - name: tenantId
        source: jsonpath
        path: $.tenant
    assertions:
      - type: status
        operator: eq
        expected: 200
`

const goodToken = "tkn-s3cr3t-12345"

func securedServer(t *testing.T, loginStatus int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var secureHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login":
			w.WriteHeader(loginStatus)
			_, _ = w.Write([]byte(`{"token":"` + goodToken + `","tenant":"acme"}`))
		case "/secure":
			secureHits.Add(1)
			if r.Header.Get("Authorization") != "Bearer "+goodToken {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &secureHits
}

func TestConformSetup_ALoginFeedsTheSecuredOperations(t *testing.T) {
	srv, _ := securedServer(t, 200)
	root := project(t, map[string]string{"api.yaml": setupSpec, "login.yaml": setupLogin})

	// without the setup the secured operation cannot be sent: nothing provides {{token}}
	out, err := exec(t, "conform", "api.yaml", "--base-url", srv.URL)
	if exitCodeOf(err) != 1 || !strings.Contains(out, "not sent") || !strings.Contains(out, "token") {
		t.Fatalf("without --setup: %v\n%s", err, out)
	}

	out, err = exec(t, "conform", "api.yaml", "--base-url", srv.URL, "--setup", "login.yaml")
	if err != nil {
		t.Fatalf("with --setup: %v\n%s", err, out)
	}
	for _, want := range []string{"Setup scenario login.yaml passed; values available to the check: tenantId, token", "PASS  GET     /secure"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, goodToken) {
		t.Errorf("the token must never be printed:\n%s", out)
	}

	// every report format masks it as well
	for _, f := range []string{"json", "html", "markdown", "junit"} {
		rep := filepath.Join(root, "rep."+f)
		if out, err := exec(t, "conform", "api.yaml", "--base-url", srv.URL, "--setup", "login.yaml", "--format", f, "--output", rep); err != nil {
			t.Fatalf("%s: %v\n%s", f, err, out)
		}
		if b, _ := os.ReadFile(rep); strings.Contains(string(b), goodToken) {
			t.Errorf("%s report leaks the extracted token", f)
		}
	}
}

func TestConformSetup_AFailedSetupStopsBeforeTheCheck(t *testing.T) {
	srv, secureHits := securedServer(t, 401)
	project(t, map[string]string{"api.yaml": setupSpec, "login.yaml": setupLogin})
	out, err := exec(t, "conform", "api.yaml", "--base-url", srv.URL, "--setup", "login.yaml")
	if err == nil || exitCodeOf(err) == 1 {
		t.Fatalf("a failed setup is a configuration error (exit 2), not a conformance failure: %v\n%s", err, out)
	}
	if !strings.Contains(err.Error(), "did not pass, so the check was not started") || !strings.Contains(out, "FAILED") {
		t.Errorf("it says what happened and shows the setup's report:\n%v\n%s", err, out)
	}
	if strings.Contains(out, goodToken) {
		t.Errorf("the failed setup report must not print the token:\n%s", out)
	}
	if secureHits.Load() != 0 {
		t.Error("no operation is sent when the setup failed")
	}
}

func TestConformSetup_AVarWinsOverAnExtractedValue(t *testing.T) {
	srv, _ := securedServer(t, 200)
	project(t, map[string]string{"api.yaml": setupSpec, "login.yaml": setupLogin})
	out, err := exec(t, "conform", "api.yaml", "--base-url", srv.URL, "--setup", "login.yaml", "--var", "token=typed-by-hand")
	if err != nil || !strings.Contains(out, "/secure  401") {
		t.Fatalf("--var token=... must win over the extracted token, so the server refuses it: %v\n%s", err, out)
	}
	if strings.Contains(out, "values available to the check: tenantId, token") {
		t.Errorf("the extracted token was not used:\n%s", out)
	}
}

func TestConformSetup_Errors(t *testing.T) {
	srv, _ := securedServer(t, 200)
	project(t, map[string]string{"api.yaml": setupSpec, "bad.yaml": "not: a scenario\n"})
	if _, err := exec(t, "conform", "api.yaml", "--base-url", srv.URL, "--setup", "missing.yaml"); err == nil || !strings.Contains(err.Error(), "--setup") {
		t.Errorf("%v", err)
	}
	if _, err := exec(t, "conform", "api.yaml", "--base-url", srv.URL, "--setup", "bad.yaml"); err == nil || !strings.Contains(err.Error(), "--setup scenario") {
		t.Errorf("%v", err)
	}
}

func TestConformSetup_FromTheProjectFile(t *testing.T) {
	srv, _ := securedServer(t, 200)
	project(t, map[string]string{
		"api.yaml":    setupSpec,
		"login.yaml":  setupLogin,
		"clrsig.yaml": "schemaVersion: \"1\"\nspec: api.yaml\nbaseUrl: " + srv.URL + "\nconform:\n  setup: login.yaml\n",
	})
	out, err := exec(t, "conform")
	if err != nil || !strings.Contains(out, "conform.setup") || !strings.Contains(out, "PASS  GET     /secure") {
		t.Fatalf("%v\n%s", err, out)
	}
	if show, err := exec(t, "config", "show", "conform"); err != nil || !strings.Contains(show, "conform.setup") {
		t.Errorf("config show lists it: %v\n%s", err, show)
	}
}
