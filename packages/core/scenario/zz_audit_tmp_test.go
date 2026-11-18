package scenario

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

func TestAuditExtraction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"small":42,"mid":1234567,"big":9007199254740993,"f":1.50,"neg":-1000000,"s":"x","nested":{"a":[1,2]}}`)
	}))
	defer srv.Close()
	sc := &domain.Scenario{Title: "t", Steps: []domain.ScenarioStep{
		{Label: "a", Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL},
			Extractions: []domain.Extraction{
				{Name: "small", Source: domain.ExtractJSONPath, Path: "$.small"},
				{Name: "mid", Source: domain.ExtractJSONPath, Path: "$.mid"},
				{Name: "big", Source: domain.ExtractJSONPath, Path: "$.big"},
				{Name: "f", Source: domain.ExtractJSONPath, Path: "$.f"},
				{Name: "neg", Source: domain.ExtractJSONPath, Path: "$.neg"},
				{Name: "nested", Source: domain.ExtractJSONPath, Path: "$.nested"},
				{Name: "missing", Source: domain.ExtractJSONPath, Path: "$.nope"},
			}},
		{Label: "b", Request: &domain.Request{Method: domain.MethodGET, URL: srv.URL + "/{{missing}}/{{mid}}"}},
	}}
	res := New(executor.New(executor.Options{})).Run(context.Background(), sc, nil)
	t.Logf("AUDIT scenario Passed=%v", res.Passed)
	for _, s := range res.Steps {
		t.Logf("AUDIT step %q passed=%v err=%q extracted=%v url=%v", s.Label, s.Passed, s.Error, s.Extracted, func() string {
			if s.Request != nil {
				return s.Request.URL
			}
			return ""
		}())
	}
}

func TestAuditExecutor(t *testing.T) {
	var gotQuery, gotHost string
	var gotAccept []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery, gotHost, gotAccept = r.URL.RawQuery, r.Host, r.Header.Values("Accept")
		fmt.Fprint(w, "0123456789012345678901234567890123456789")
	}))
	defer srv.Close()
	ex := executor.New(executor.Options{MaxBodySize: 10})
	resp, err := ex.Execute(context.Background(), &domain.Request{Method: domain.MethodGET,
		URL:     srv.URL + "/p?b=2&a=1&flag&x=a%3Bb&sig=ab%2Bc%3D&arr[]=1",
		Headers: []domain.Header{{Name: "Accept", Value: "a"}, {Name: "Accept", Value: "b"}, {Name: "Host", Value: "custom.example"}}})
	t.Logf("AUDIT err=%v", err)
	t.Logf("AUDIT sent query: sent=%q got=%q", "b=2&a=1&flag&x=a%3Bb&sig=ab%2Bc%3D&arr[]=1", gotQuery)
	t.Logf("AUDIT host=%q accept=%v", gotHost, gotAccept)
	t.Logf("AUDIT truncated body len=%d size=%d (limit 10)", len(resp.Body), resp.Size)
}

func TestAuditProxyEnv(t *testing.T) {
	hit := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true; fmt.Fprint(w, "via proxy") }))
	defer proxy.Close()
	os.Setenv("HTTP_PROXY", proxy.URL)
	os.Setenv("http_proxy", proxy.URL)
	defer os.Unsetenv("HTTP_PROXY")
	defer os.Unsetenv("http_proxy")
	ex := executor.New(executor.Options{})
	_, err := ex.Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: "http://example.invalid/x"})
	t.Logf("AUDIT system proxy used=%v err=%v", hit, err)
	ex2 := executor.New(executor.Options{ProxyURL: "http://bad host:99999"})
	_, err = ex2.Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: "http://example.invalid/x"})
	t.Logf("AUDIT invalid ProxyURL silently ignored? err=%v", err)
}
