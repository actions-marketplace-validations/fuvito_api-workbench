package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func newRS(t *testing.T) (*RequestService, *WorkspaceService) {
	t.Helper()
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	rs := NewRequestService(ws, NewSettingsService())
	t.Cleanup(rs.close)
	return rs, ws
}

func TestExecute_EndToEndWithEnvironmentAndHistory(t *testing.T) {
	var gotAuth, gotQuery, gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("X-Reply", "yes")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	rs, ws := newRS(t)
	envDir := filepath.Join(ws.workspacePath, "environments")
	os.MkdirAll(envDir, 0o755)
	info, err := ws.SaveEnvironment(EnvironmentIO{Name: "e", Path: filepath.Join(envDir, "e.yaml"), Variables: []EnvironmentVariableIO{
		{Name: "base", Value: srv.URL}, {Name: "tok", Value: "T0K", Secret: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.SelectEnvironment(info.Path); err != nil {
		t.Fatal(err)
	}

	out := rs.Execute(RequestInput{
		Method:      "POST",
		URL:         "{{base}}/items",
		QueryParams: []QueryParamIO{{Name: "a", Value: "1"}},
		Body:        &BodyIO{Type: "json", Content: `{"n":1}`},
		Auth:        &AuthIO{Type: "bearer", Token: "{{tok}}"},
	})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if out.StatusCode != 201 || out.Body != `{"ok":true}` || out.Size != 11 {
		t.Errorf("response: %+v", out)
	}
	if gotAuth != "Bearer T0K" || gotQuery != "a=1" || gotBody != `{"n":1}` || gotCT != "application/json" {
		t.Errorf("server saw auth=%q query=%q body=%q ct=%q", gotAuth, gotQuery, gotBody, gotCT)
	}
	var reply string
	for _, h := range out.Headers {
		if h.Name == "X-Reply" {
			reply = h.Value
		}
	}
	if reply != "yes" {
		t.Error("response headers not mapped")
	}

	hist := rs.GetHistory()
	if len(hist) != 1 {
		t.Fatalf("history: %+v", hist)
	}
	h := hist[0]
	if h.Method != "POST" || h.StatusCode != 201 || h.BodyType != "json" || h.BodyContent != `{"n":1}` {
		t.Errorf("history entry: %+v", h)
	}
	if !strings.Contains(h.URL, "{{base}}") {
		t.Errorf("history should keep the unresolved URL so secrets are not stored: %q", h.URL)
	}
	if h.Auth == nil || h.Auth.Token != "{{tok}}" {
		t.Errorf("variable-referencing auth should be kept: %+v", h.Auth)
	}

	rs.ClearHistory()
	if len(rs.GetHistory()) != 0 {
		t.Error("history should be cleared")
	}
}

func TestExecute_ErrorsAreReturnedNotThrown(t *testing.T) {
	rs, _ := newRS(t)

	if out := rs.Execute(RequestInput{Method: "GET", URL: ""}); out.Error == "" {
		t.Error("invalid request should report an error")
	}
	if out := rs.Execute(RequestInput{Method: "NOPE", URL: "http://x"}); out.Error == "" {
		t.Error("invalid method should report an error")
	}

	out := rs.Execute(RequestInput{Method: "GET", URL: "http://127.0.0.1:1/", TimeoutMs: 500})
	if out.Error == "" {
		t.Fatal("connection failure should be reported")
	}
	if hist := rs.GetHistory(); len(hist) != 1 || hist[0].StatusCode != 0 {
		t.Errorf("failed requests are still recorded: %+v", hist)
	}
}

func TestExecute_LiteralCredentialsNeverStored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	rs, _ := newRS(t)

	rs.Execute(RequestInput{Method: "GET", URL: srv.URL, Auth: &AuthIO{Type: "bearer", Token: "literal-secret"}})
	rs.Execute(RequestInput{Method: "GET", URL: srv.URL, Headers: []HeaderIO{{Name: "Authorization", Value: "Bearer hdr-secret"}, {Name: "X-Ok", Value: "v"}}})

	for _, e := range rs.GetHistory() {
		if e.Auth != nil {
			t.Errorf("literal auth leaked into history: %+v", e.Auth)
		}
		for _, h := range e.Headers {
			if strings.Contains(h.Value, "hdr-secret") {
				t.Errorf("Authorization header value leaked: %+v", h)
			}
			if h.Name == "X-Ok" && h.Value != "v" {
				t.Errorf("non-sensitive header should be kept: %+v", h)
			}
		}
	}
}

func TestCancelRequest(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	rs, _ := newRS(t)
	rs.CancelRequest() // nothing in flight: must be a safe no-op

	done := make(chan ResponseOutput, 1)
	go func() { done <- rs.Execute(RequestInput{Method: "GET", URL: srv.URL, TimeoutMs: 10000}) }()

	deadline := time.Now().Add(3 * time.Second)
	for {
		rs.mu.Lock()
		inFlight := rs.cancelLast != nil
		rs.mu.Unlock()
		if inFlight {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("request never became in-flight")
		}
		time.Sleep(5 * time.Millisecond)
	}
	rs.CancelRequest()

	select {
	case out := <-done:
		if out.Error == "" {
			t.Error("cancelled request should report an error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel did not abort the request")
	}
}

func TestRebuildExecutorAppliesSettings(t *testing.T) {
	rs, _ := newRS(t)
	before := rs.exec
	s := rs.settings.GetSettings()
	s.DefaultTimeoutMs = 1234
	rs.settings.SaveSettings(s) // onSave is not wired here, so call explicitly
	rs.RebuildExecutor()
	if rs.exec == before {
		t.Error("executor should be recreated")
	}
	if buildExecutor(nil) == nil || buildExecutorWithCookies(nil) == nil || buildExecutorWithCookies(rs.settings) == nil {
		t.Error("builders should always return an executor")
	}
}

func TestAuthMapping(t *testing.T) {
	cases := []struct {
		in   AuthIO
		want func(*domain.Auth) bool
	}{
		{AuthIO{Type: "bearer", Token: "t"}, func(a *domain.Auth) bool { return a.Bearer != nil && a.Bearer.Token == "t" }},
		{AuthIO{Type: "basic", Username: "u", Password: "p"}, func(a *domain.Auth) bool { return a.Basic != nil && a.Basic.Username == "u" && a.Basic.Password == "p" }},
		{AuthIO{Type: "apikey", Key: "k", Value: "v", Location: "query"}, func(a *domain.Auth) bool {
			return a.APIKey != nil && a.APIKey.Key == "k" && a.APIKey.Value == "v" && a.APIKey.Location == domain.APIKeyLocation("query")
		}},
		{AuthIO{Type: "none"}, func(a *domain.Auth) bool { return a.Bearer == nil && a.Basic == nil && a.APIKey == nil }},
	}
	for _, c := range cases {
		if got := inputAuthToDomain(&c.in); !c.want(got) {
			t.Errorf("%s mapped wrong: %+v", c.in.Type, got)
		}
	}

	req, err := inputToDomain(RequestInput{
		Method: "GET", URL: "http://x", TimeoutMs: 1500,
		Headers: []HeaderIO{{Name: "A", Value: "b"}}, Auth: &AuthIO{Type: "bearer", Token: "t"},
	})
	if err != nil || req.Timeout != 1500*time.Millisecond || len(req.Headers) != 1 || req.Auth == nil {
		t.Errorf("inputToDomain: %+v %v", req, err)
	}
}

func TestAuthJSONForHistory(t *testing.T) {
	cases := []struct {
		name string
		auth *AuthIO
		want bool
	}{
		{"nil", nil, false},
		{"none", &AuthIO{Type: "none"}, false},
		{"empty type", &AuthIO{}, false},
		{"literal bearer", &AuthIO{Type: "bearer", Token: "x"}, false},
		{"ref bearer", &AuthIO{Type: "bearer", Token: "{{t}}"}, true},
		{"literal basic", &AuthIO{Type: "basic", Username: "u", Password: "p"}, false},
		{"ref basic user", &AuthIO{Type: "basic", Username: "{{u}}", Password: "p"}, true},
		{"ref basic pass", &AuthIO{Type: "basic", Username: "u", Password: "{{p}}"}, true},
		{"literal apikey", &AuthIO{Type: "apikey", Key: "k", Value: "v"}, false},
		{"ref apikey", &AuthIO{Type: "apikey", Key: "k", Value: "{{v}}"}, true},
		{"unknown type", &AuthIO{Type: "weird", Token: "{{x}}"}, false},
	}
	for _, c := range cases {
		if got := authJSONForHistory(c.auth) != ""; got != c.want {
			t.Errorf("%s: stored=%v want %v", c.name, got, c.want)
		}
	}
	if isVarRef("{{a") || !isVarRef("x{{a}}y") {
		t.Error("isVarRef")
	}
}

func TestRequestFileIOMapping(t *testing.T) {
	in := RequestFileIO{
		SchemaVersion: "1", Title: "t", Method: "put", URL: "http://x", TimeoutMs: 2500,
		Headers:     []HeaderIO{{Name: "H", Value: "v"}},
		QueryParams: []QueryParamIO{{Name: "q", Value: "1"}},
		Body:        &BodyIO{Type: "json", Content: "{}"},
		Auth:        &AuthIO{Type: "apikey", Key: "k", Value: "v", Location: "header"},
	}
	dom := requestFileIOToDomain(in)
	back := domainToRequestFileIO(dom)
	if back.Title != "t" || back.URL != "http://x" || back.TimeoutMs != 2500 ||
		len(back.Headers) != 1 || len(back.QueryParams) != 1 || back.Body == nil || back.Body.Type != "json" ||
		back.Auth == nil || back.Auth.Type != "apikey" || back.Auth.Key != "k" || back.Auth.Location != "header" {
		t.Errorf("round trip lost data: %+v", back)
	}

	bearer := domainToRequestFileIO(requestFileIOToDomain(RequestFileIO{Method: "GET", URL: "u", Auth: &AuthIO{Type: "bearer", Token: "t"}}))
	basic := domainToRequestFileIO(requestFileIOToDomain(RequestFileIO{Method: "GET", URL: "u", Auth: &AuthIO{Type: "basic", Username: "a", Password: "b"}}))
	if bearer.Auth == nil || bearer.Auth.Token != "t" || basic.Auth == nil || basic.Auth.Username != "a" || basic.Auth.Password != "b" {
		t.Errorf("bearer/basic: %+v %+v", bearer.Auth, basic.Auth)
	}
	plain := domainToRequestFileIO(requestFileIOToDomain(RequestFileIO{Method: "GET", URL: "u"}))
	if plain.Body != nil || plain.Auth != nil {
		t.Errorf("absent body/auth should stay absent: %+v", plain)
	}
}

func TestExecute_ReturnsTheRequestAsSentWithCredentialsMasked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	rs, ws := newRS(t)
	envDir := filepath.Join(ws.workspacePath, "environments")
	os.MkdirAll(envDir, 0o755)
	info, err := ws.SaveEnvironment(EnvironmentIO{Name: "e", Path: filepath.Join(envDir, "e.yaml"), Variables: []EnvironmentVariableIO{
		{Name: "base", Value: srv.URL}, {Name: "pw", Value: "s3cret-value", Secret: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.SelectEnvironment(info.Path); err != nil {
		t.Fatal(err)
	}

	out := rs.Execute(RequestInput{
		Method:      "POST",
		URL:         "{{base}}/items",
		QueryParams: []QueryParamIO{{Name: "a", Value: "1"}, {Name: "api_key", Value: "k-123"}},
		Headers:     []HeaderIO{{Name: "X-Trace", Value: "t1"}, {Name: "X-Note", Value: "has {{pw}} inside"}},
		Body:        &BodyIO{Type: "json", Content: `{"password":"{{pw}}"}`},
		Auth:        &AuthIO{Type: "bearer", Token: "literal-token"},
	})
	if out.Error != "" {
		t.Fatal(out.Error)
	}
	if out.RequestMethod != "POST" || !strings.HasPrefix(out.RequestURL, srv.URL+"/items?a=1&api_key=***") {
		t.Errorf("sent request: %s %s", out.RequestMethod, out.RequestURL)
	}
	if strings.Contains(out.RequestBody, "s3cret-value") || !strings.Contains(out.RequestBody, "***") {
		t.Errorf("secret value must be masked in the body: %q", out.RequestBody)
	}
	headers := map[string]string{}
	for _, h := range out.RequestHeaders {
		headers[h.Name] = h.Value
	}
	if headers["X-Trace"] != "t1" || headers["X-Note"] != "has *** inside" || headers["Authorization"] != "Bearer ***" {
		t.Errorf("sent headers: %+v", headers)
	}
	for _, h := range out.RequestHeaders {
		if strings.Contains(h.Value, "literal-token") || strings.Contains(h.Value, "s3cret-value") {
			t.Errorf("credential leaked in %+v", h)
		}
	}
}

func TestExecute_ErrorsStillReportTheRequest(t *testing.T) {
	rs, _ := newRS(t)
	out := rs.Execute(RequestInput{Method: "GET", URL: "http://127.0.0.1:1/never"})
	if out.Error == "" {
		t.Fatal("expected a connection error")
	}
	if out.RequestMethod != "GET" || out.RequestURL != "http://127.0.0.1:1/never" {
		t.Errorf("sent request on error: %s %s", out.RequestMethod, out.RequestURL)
	}
}
