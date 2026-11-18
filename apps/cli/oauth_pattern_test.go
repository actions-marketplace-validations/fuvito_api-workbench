package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The OAuth2 client-credentials pattern from the user guide, run for real: a token step, an
// extraction, then bearer auth with the extracted value. Keep it in step with the guide.
const oauthScenario = `schemaVersion: "1"
title: OAuth2 client credentials
steps:
  - label: Get token
    request:
      method: POST
      url: "{{authUrl}}/oauth/token"
      body:
        type: form-urlencoded
        content: "grant_type=client_credentials&client_id={{clientId}}&client_secret={{clientSecret}}"
    assertions:
      - {type: status, operator: eq, expected: 200}
    extractions:
      - {name: accessToken, source: jsonpath, path: $.access_token}
  - label: Call the API
    request:
      method: GET
      url: "{{baseUrl}}/me"
      auth:
        type: bearer
        token: "{{accessToken}}"
    assertions:
      - {type: status, operator: eq, expected: 200}
`

func TestRun_OAuth2ClientCredentialsPattern(t *testing.T) {
	const secret = "s3cr3t-client-secret"
	const token = "access-token-12345"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			body, _ := io.ReadAll(r.Body)
			if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" ||
				!strings.Contains(string(body), "grant_type=client_credentials") ||
				!strings.Contains(string(body), "client_secret="+secret) {
				w.WriteHeader(400)
				return
			}
			fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, token)
		case "/me":
			if r.Header.Get("Authorization") != "Bearer "+token {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(`{"id":1}`))
		}
	}))
	defer srv.Close()

	sc := writeTempFile(t, "oauth.yaml", oauthScenario)
	env := writeTempFile(t, "env.yaml", fmt.Sprintf(`schemaVersion: "1"
name: ci
variables:
  - {name: authUrl, value: "%s"}
  - {name: baseUrl, value: "%s"}
  - {name: clientId, value: my-client}
  - {name: clientSecret, value: "${CLIENT_SECRET}", secret: true}
`, srv.URL, srv.URL))
	t.Setenv("CLIENT_SECRET", secret)

	out, err := exec(t, "run", "--env", env, sc)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, secret) || strings.Contains(out, token) {
		t.Errorf("neither the client secret nor the token may appear in the report:\n%s", out)
	}
}
