package executor_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

type seen struct {
	contentType string
	fields      map[string]string
	files       map[string]string // field -> "filename|content-type|content"
	cookie      string
	raw         string
}

func captureServer(t *testing.T) (*httptest.Server, *seen) {
	t.Helper()
	s := &seen{fields: map[string]string{}, files: map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.contentType = r.Header.Get("Content-Type")
		if c, err := r.Cookie("sid"); err == nil {
			s.cookie = c.Value
		}
		if strings.HasPrefix(s.contentType, "multipart/") {
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("parse multipart: %v", err)
			}
			for k, v := range r.MultipartForm.Value {
				s.fields[k] = v[0]
			}
			for k, fh := range r.MultipartForm.File {
				f, _ := fh[0].Open()
				b, _ := io.ReadAll(f)
				f.Close()
				s.files[k] = fh[0].Filename + "|" + fh[0].Header.Get("Content-Type") + "|" + string(b)
			}
		} else {
			b, _ := io.ReadAll(r.Body)
			s.raw = string(b)
		}
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc"})
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, s
}

func TestMultipart_JSONFieldsWithFile(t *testing.T) {
	srv, s := captureServer(t)
	dir := t.TempDir()
	txt := filepath.Join(dir, "note.txt")
	bin := filepath.Join(dir, "blob.unknownext")
	os.WriteFile(txt, []byte("hello"), 0o600)
	os.WriteFile(bin, []byte{1, 2, 3}, 0o600)
	content, _ := json.Marshal([]map[string]string{
		{"n": "title", "t": "text", "v": "doc"},
		{"n": "note", "t": "file", "v": txt},
		{"n": "blob", "t": "file", "v": bin},
	})

	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST, URL: srv.URL,
		Body: &domain.RequestBody{Type: domain.BodyMultipart, Content: string(content)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s.contentType, "multipart/form-data; boundary=") {
		t.Errorf("content type %q", s.contentType)
	}
	if s.fields["title"] != "doc" {
		t.Errorf("fields: %v", s.fields)
	}
	if !strings.HasPrefix(s.files["note"], "note.txt|text/plain") || !strings.HasSuffix(s.files["note"], "|hello") {
		t.Errorf("note file: %q", s.files["note"])
	}
	if !strings.Contains(s.files["blob"], "|application/octet-stream|") {
		t.Errorf("unknown extension should fall back to octet-stream: %q", s.files["blob"])
	}
}

func TestMultipart_LegacyUrlEncodedAndEmpty(t *testing.T) {
	srv, s := captureServer(t)
	_, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST, URL: srv.URL,
		Body: &domain.RequestBody{Type: domain.BodyMultipart, Content: "a=1&b=2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.fields["a"] != "1" || s.fields["b"] != "2" {
		t.Errorf("fields: %v", s.fields)
	}

	if _, err := newExec().Execute(context.Background(), &domain.Request{
		Method: domain.MethodPOST, URL: srv.URL,
		Body: &domain.RequestBody{Type: domain.BodyMultipart},
	}); err != nil {
		t.Fatalf("empty multipart should work: %v", err)
	}
}

func TestMultipart_Errors(t *testing.T) {
	for name, content := range map[string]string{
		"bad json":     `[{"n":`,
		"missing file": `[{"n":"f","t":"file","v":"/definitely/not/here.bin"}]`,
		"bad query":    "a=%zz",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newExec().Execute(context.Background(), &domain.Request{
				Method: domain.MethodPOST, URL: "http://127.0.0.1:1",
				Body: &domain.RequestBody{Type: domain.BodyMultipart, Content: content},
			})
			if err == nil || !strings.Contains(err.Error(), "multipart") {
				t.Errorf("expected multipart error, got %v", err)
			}
		})
	}
}

func TestBodyContentTypes(t *testing.T) {
	srv, s := captureServer(t)
	tests := []struct {
		name   string
		body   *domain.RequestBody
		header []domain.Header
		want   string
	}{
		{"form", &domain.RequestBody{Type: domain.BodyFormURLEncoded, Content: "a=1"}, nil, "application/x-www-form-urlencoded"},
		{"raw has no default", &domain.RequestBody{Type: domain.BodyRaw, Content: "hi"}, nil, ""},
		{"explicit header wins", &domain.RequestBody{Type: domain.BodyJSON, Content: "{}"}, []domain.Header{{Name: "Content-Type", Value: "application/vnd.x+json"}}, "application/vnd.x+json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newExec().Execute(context.Background(), &domain.Request{
				Method: domain.MethodPOST, URL: srv.URL, Body: tc.body, Headers: tc.header,
			})
			if err != nil {
				t.Fatal(err)
			}
			if s.contentType != tc.want {
				t.Errorf("content type %q, want %q", s.contentType, tc.want)
			}
		})
	}
}

func TestWithCookies_ReplaysSetCookie(t *testing.T) {
	srv, s := captureServer(t)
	ex := executor.New(executor.Options{WithCookies: true})
	for i := 0; i < 2; i++ {
		if _, err := ex.Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: srv.URL}); err != nil {
			t.Fatal(err)
		}
	}
	if s.cookie != "abc" {
		t.Errorf("cookie not replayed, got %q", s.cookie)
	}

	s.cookie = ""
	stateless := executor.New(executor.Options{})
	for i := 0; i < 2; i++ {
		stateless.Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: srv.URL})
	}
	if s.cookie != "" {
		t.Error("stateless executor must not keep cookies")
	}
}

func TestProxyURL(t *testing.T) {
	var hit bool
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer proxy.Close()

	ex := executor.New(executor.Options{ProxyURL: proxy.URL})
	resp, err := ex.Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: "http://upstream.invalid/x"})
	if err != nil {
		t.Fatal(err)
	}
	if !hit || resp.StatusCode != http.StatusTeapot {
		t.Errorf("request should have gone through proxy (hit=%v status=%d)", hit, resp.StatusCode)
	}
}

func TestTLSSkipVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	req := &domain.Request{Method: domain.MethodGET, URL: srv.URL}

	if _, err := executor.New(executor.Options{}).Execute(context.Background(), req); err == nil {
		t.Error("self-signed cert should fail verification by default")
	}
	resp, err := executor.New(executor.Options{TLSSkipVerify: true}).Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Timing.TLSMs < 0 || resp.Timing.TotalMs() < 0 {
		t.Errorf("timing should be non-negative: %+v", resp.Timing)
	}
}
