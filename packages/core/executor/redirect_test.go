package executor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

func redirectServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new", http.StatusFound)
	})
	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("final")) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRedirectsAreFollowedByDefault(t *testing.T) {
	srv := redirectServer(t)
	resp, err := executor.New(executor.Options{}).Execute(context.Background(),
		&domain.Request{Method: domain.MethodGET, URL: srv.URL + "/old"})
	if err != nil || resp.StatusCode != 200 || resp.BodyString() != "final" {
		t.Fatalf("default behaviour must be unchanged: %v %+v", err, resp)
	}
}

func TestNoFollowRedirectsReturnsTheRedirectItself(t *testing.T) {
	srv := redirectServer(t)
	ex := executor.New(executor.Options{WithCookies: true})
	resp, err := ex.Execute(context.Background(),
		&domain.Request{Method: domain.MethodGET, URL: srv.URL + "/old", NoFollowRedirects: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	var loc string
	for _, h := range resp.Headers {
		if h.Name == "Location" {
			loc = h.Value
		}
	}
	if loc != "/new" {
		t.Errorf("Location = %q", loc)
	}
	// the same executor still follows redirects for the next request
	next, err := ex.Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: srv.URL + "/old"})
	if err != nil || next.StatusCode != 200 {
		t.Errorf("an opt-out is per request: %v %+v", err, next)
	}
}
