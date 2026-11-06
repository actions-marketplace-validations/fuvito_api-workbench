package executor_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

func newExec() *executor.Executor {
	return executor.New(executor.Options{DefaultTimeout: 5 * time.Second})
}

func TestExecute_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("want GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	resp, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL + "/test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status: want 200, got %d", resp.StatusCode)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("body: want %q, got %q", `{"ok":true}`, resp.Body)
	}
	if resp.Duration == 0 {
		t.Error("duration should be non-zero")
	}
}

func TestExecute_POST_JSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type: want application/json, got %q", ct)
		}
		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	}))
	defer srv.Close()

	resp, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST,
		URL:    srv.URL + "/items",
		Body:   &domain.RequestBody{Type: domain.BodyJSON, Content: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: want 201, got %d", resp.StatusCode)
	}
}

func TestExecute_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page: want 2, got %q", got)
		}
		if got := r.URL.Query().Get("size"); got != "10" {
			t.Errorf("size: want 10, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
		QueryParams: []domain.QueryParam{
			{Name: "page", Value: "2"},
			{Name: "size", Value: "10"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_Headers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom"); got != "hello" {
			t.Errorf("X-Custom: want hello, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method:  domain.MethodGET,
		URL:     srv.URL,
		Headers: []domain.Header{{Name: "X-Custom", Value: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_BearerAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken" {
			t.Errorf("Authorization: want %q, got %q", "Bearer mytoken", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
		Auth: &domain.Auth{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuth{Token: "mytoken"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_BasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("BasicAuth: want admin/secret, got %q/%q ok=%v", user, pass, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
		Auth: &domain.Auth{
			Type:  domain.AuthBasic,
			Basic: &domain.BasicAuth{Username: "admin", Password: "secret"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_APIKeyHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "abc123" {
			t.Errorf("X-API-Key: want abc123, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
		Auth: &domain.Auth{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuth{
				Key:      "X-API-Key",
				Value:    "abc123",
				Location: domain.APIKeyInHeader,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_APIKeyQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("api_key"); got != "abc123" {
			t.Errorf("api_key: want abc123, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
		Auth: &domain.Auth{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuth{
				Key:      "api_key",
				Value:    "abc123",
				Location: domain.APIKeyInQuery,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_AllMethods(t *testing.T) {
	methods := []domain.HTTPMethod{
		domain.MethodGET, domain.MethodPOST, domain.MethodPUT,
		domain.MethodPATCH, domain.MethodDELETE, domain.MethodHEAD, domain.MethodOPTIONS,
	}
	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != string(method) {
					t.Errorf("want %s, got %s", method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			_, err := newExec().Execute(context.Background(), &domain.Request{
				Method: method,
				URL:    srv.URL,
			})
			if err != nil {
				t.Fatalf("%s: %v", method, err)
			}
		})
	}
}

func TestExecute_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method:  domain.MethodGET,
		URL:     srv.URL,
		Timeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := newExec().Execute(ctx, &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
	})
	if err == nil {
		t.Error("expected cancellation error, got nil")
	}
}

func TestExecute_InvalidRequest(t *testing.T) {
	_, err := newExec().Execute(context.Background(), &domain.Request{})
	if err == nil {
		t.Error("expected validation error for empty request")
	}
}

func TestExecute_MultipartFormBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type: want multipart/form-data prefix, got %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("username"); got != "alice" {
			t.Errorf("username: want alice, got %q", got)
		}
		if got := r.FormValue("role"); got != "admin" {
			t.Errorf("role: want admin, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST,
		URL:    srv.URL + "/upload",
		Body: &domain.RequestBody{
			Type:    domain.BodyMultipart,
			Content: "username=alice&role=admin",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_MultipartEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type: want multipart/form-data prefix, got %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	resp, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST,
		URL:    srv.URL,
		Body:   &domain.RequestBody{Type: domain.BodyMultipart, Content: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		t.Errorf("status: want 204, got %d", resp.StatusCode)
	}
}

func TestExecute_MultipartSpecialChars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("note"); got != "hello world & more" {
			t.Errorf("note: want %q, got %q", "hello world & more", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST,
		URL:    srv.URL,
		Body: &domain.RequestBody{
			Type:    domain.BodyMultipart,
			Content: "note=" + "hello+world+%26+more",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_ResponseSize(t *testing.T) {
	body := strings.Repeat("x", 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, body)
	}))
	defer srv.Close()

	resp, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodGET,
		URL:    srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Size != 1024 {
		t.Errorf("Size: want 1024, got %d", resp.Size)
	}
}
