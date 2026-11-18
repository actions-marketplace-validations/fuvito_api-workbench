package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func TestExchangeRequest_MasksCredentials(t *testing.T) {
	req := &domain.Request{
		Method:      domain.MethodPOST,
		URL:         "http://x/items",
		QueryParams: []domain.QueryParam{{Name: "page", Value: "2"}, {Name: "api_key", Value: "k-123"}},
		Headers: []domain.Header{
			{Name: "X-Trace", Value: "abc"}, {Name: "Authorization", Value: "Bearer raw"}, {Name: "X-Note", Value: "uses hunter2 inside"},
		},
		Auth: &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "tok-secret"}},
		Body: &domain.RequestBody{Type: domain.BodyJSON, Content: `{"pw":"hunter2"}`},
	}
	method, url, headers, body := exchangeRequest(req, []string{"hunter2"})
	if method != "POST" || url != "http://x/items?page=2&api_key=***" {
		t.Errorf("method/url = %s %s", method, url)
	}
	var all []string
	for _, h := range headers {
		all = append(all, h.Name+": "+h.Value)
	}
	joined := strings.Join(all, "|")
	for _, want := range []string{"X-Trace: abc", "Authorization: ***", "X-Note: uses *** inside", "Authorization: Bearer ***"} {
		if !strings.Contains(joined, want) {
			t.Errorf("headers missing %q: %s", want, joined)
		}
	}
	for _, leaked := range []string{"raw", "tok-secret", "hunter2", "k-123"} {
		if strings.Contains(joined+url+body, leaked) {
			t.Errorf("%q leaked", leaked)
		}
	}
	if body != `{"pw":"***"}` {
		t.Errorf("body = %s", body)
	}
}

func TestRunScenario_ReturnsTheRequestAsSent(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Reply", "yes")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	ws := t.TempDir()
	scn := writeFile(t, filepath.Join(ws, "scenarios", "ex.yaml"), `schemaVersion: "1"
title: Ex
variables:
  - name: id
    value: "user-{{$randomString 6}}"
steps:
  - label: go
    request:
      method: POST
      url: "`+srv.URL+`/things/{{id}}"
      queryParams:
        - {name: v, value: "1"}
      headers:
        - {name: X-Req, value: "{{id}}"}
      body: {type: json, content: '{"id":"{{id}}"}'}
`)
	res := NewScenarioService(newWS(t, ws), NewSettingsService()).RunScenario(scn, "", 0)
	if res.Error != "" || len(res.Steps) != 1 {
		t.Fatalf("%+v", res)
	}
	st := res.Steps[0]
	id := res.Variables[0].Value
	if st.RequestMethod != "POST" || !strings.HasSuffix(st.RequestURL, "/things/"+id+"?v=1") {
		t.Errorf("url = %s", st.RequestURL)
	}
	if len(st.RequestHeaders) != 1 || st.RequestHeaders[0].Value != id || st.RequestBody != `{"id":"`+id+`"}` {
		t.Errorf("headers/body = %+v %s", st.RequestHeaders, st.RequestBody)
	}
	if st.ResponseBody != `{"ok":true}` || len(st.ResponseHeaders) == 0 {
		t.Errorf("response = %q %+v", st.ResponseBody, st.ResponseHeaders)
	}
}
