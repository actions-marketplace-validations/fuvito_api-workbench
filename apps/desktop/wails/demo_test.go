package main

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The demo server keeps package-level state, so these tests must not run in parallel.

type demoCall struct {
	method, path, body string
	headers            map[string]string
}

func doDemo(t *testing.T, h http.Handler, c demoCall) (int, http.Header, string) {
	t.Helper()
	req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	b, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, rec.Header(), string(b)
}

func decode(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, body)
	}
	return m
}

func TestDemoServer_Basics(t *testing.T) {
	resetDemoState()
	h := newDemoMux()

	code, hdr, body := doDemo(t, h, demoCall{method: "GET", path: "/health"})
	if code != 200 || decode(t, body)["status"] != "ok" || !strings.Contains(hdr.Get("Content-Type"), "json") {
		t.Errorf("health: %d %s", code, body)
	}
	if hdr.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header missing")
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "OPTIONS", path: "/users"}); code != http.StatusNoContent {
		t.Errorf("preflight: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/health"}); code != 405 {
		t.Errorf("health POST: %d", code)
	}

	_, _, body = doDemo(t, h, demoCall{method: "POST", path: "/echo?x=1&x=2", body: "hello", headers: map[string]string{"X-Test": "yes"}})
	echo := decode(t, body)
	if echo["method"] != "POST" || echo["body"] != "hello" || echo["queryParams"].(map[string]any)["x"] != "1, 2" ||
		echo["headers"].(map[string]any)["X-Test"] != "yes" {
		t.Errorf("echo: %v", echo)
	}

	for path, want := range map[string]int{"/status/418": 418, "/status/200": 200, "/status/999": 400, "/status/abc": 400, "/status/99": 400} {
		if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: path}); code != want {
			t.Errorf("%s → %d, want %d", path, code, want)
		}
	}
	if _, _, b := doDemo(t, h, demoCall{method: "GET", path: "/status/599"}); decode(t, b)["text"] != "Unknown Status" {
		t.Errorf("unknown status text: %s", b)
	}

	start := time.Now()
	if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: "/slow?delay=30"}); code != 200 || time.Since(start) < 25*time.Millisecond {
		t.Errorf("slow should delay (code %d, %v)", code, time.Since(start))
	}
	for _, p := range []string{"/slow?delay=-1", "/slow?delay=abc"} {
		if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: p}); code != 400 {
			t.Errorf("%s → %d", p, code)
		}
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/slow"}); code != 405 {
		t.Errorf("slow POST: %d", code)
	}

	if code, hdr, b := doDemo(t, h, demoCall{method: "GET", path: "/openapi.yaml"}); code != 200 || !strings.Contains(hdr.Get("Content-Type"), "yaml") || !strings.Contains(b, "openapi") {
		t.Errorf("spec: %d", code)
	}
	if code, hdr, b := doDemo(t, h, demoCall{method: "GET", path: "/docs"}); code != 200 || !strings.Contains(hdr.Get("Content-Type"), "html") || !strings.Contains(b, "swagger-ui") {
		t.Errorf("docs: %d", code)
	}
	for _, p := range []string{"/openapi.yaml", "/docs"} {
		if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: p}); code != 405 {
			t.Errorf("%s POST: %d", p, code)
		}
	}

	before := demoReqCount.Load()
	doDemo(t, h, demoCall{method: "GET", path: "/health"})
	if demoReqCount.Load() != before+1 {
		t.Error("counted endpoints should bump the request counter")
	}
	doDemo(t, h, demoCall{method: "GET", path: "/docs"})
	if demoReqCount.Load() != before+1 {
		t.Error("docs/spec are not counted")
	}
}

func TestDemoServer_Auth(t *testing.T) {
	resetDemoState()
	h := newDemoMux()
	login := func(user, pass string) (int, map[string]any) {
		c, _, b := doDemo(t, h, demoCall{method: "POST", path: "/auth/login", body: `{"username":"` + user + `","password":"` + pass + `"}`})
		return c, decode(t, b)
	}

	code, out := login("admin", "secret")
	if code != 200 || out["token"] != demoToken {
		t.Fatalf("login: %d %v", code, out)
	}
	if code, _ := login("admin", "wrong"); code != 401 {
		t.Errorf("bad password: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/auth/login", body: "not json"}); code != 400 {
		t.Errorf("bad body: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: "/auth/login"}); code != 405 {
		t.Errorf("login GET: %d", code)
	}

	bearer := map[string]string{"Authorization": "Bearer " + demoToken}
	if code, _, b := doDemo(t, h, demoCall{method: "GET", path: "/auth/profile", headers: bearer}); code != 200 || decode(t, b)["role"] != "admin" {
		t.Errorf("profile: %d %s", code, b)
	}
	for name, hdr := range map[string]map[string]string{"none": nil, "wrong": {"Authorization": "Bearer nope"}, "scheme": {"Authorization": "Basic abc"}} {
		if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: "/auth/profile", headers: hdr}); code != 401 {
			t.Errorf("profile with %s credentials: %d", name, code)
		}
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/auth/profile"}); code != 405 {
		t.Errorf("profile POST: %d", code)
	}

	req := httptest.NewRequest("GET", "/auth/basic", nil)
	req.SetBasicAuth("demo", "demo")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("basic ok: %d", rec.Code)
	}
	code, hdr, _ := doDemo(t, h, demoCall{method: "GET", path: "/auth/basic"})
	if code != 401 || !strings.Contains(hdr.Get("WWW-Authenticate"), "Basic") {
		t.Errorf("basic challenge: %d %v", code, hdr)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/auth/basic"}); code != 405 {
		t.Errorf("basic POST: %d", code)
	}

	if code, _, b := doDemo(t, h, demoCall{method: "GET", path: "/auth/apikey", headers: map[string]string{"X-Api-Key": demoAPIKey}}); code != 200 || strings.Contains(b, demoAPIKey) {
		t.Errorf("apikey header: %d (the full key must not be echoed back) %s", code, b)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: "/auth/apikey?api_key=" + demoAPIKey}); code != 200 {
		t.Errorf("apikey query: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: "/auth/apikey?api_key=bad"}); code != 401 {
		t.Errorf("apikey bad: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/auth/apikey"}); code != 405 {
		t.Errorf("apikey POST: %d", code)
	}
}

func TestDemoServer_UsersCRUD(t *testing.T) {
	resetDemoState()
	h := newDemoMux()

	_, _, b := doDemo(t, h, demoCall{method: "GET", path: "/users"})
	list := decode(t, b)
	if list["total"].(float64) != 5 || len(list["data"].([]any)) != 5 {
		t.Fatalf("seed: %v", list)
	}
	_, _, b = doDemo(t, h, demoCall{method: "GET", path: "/users?page=2&limit=2"})
	page := decode(t, b)
	if page["page"].(float64) != 2 || len(page["data"].([]any)) != 2 || page["total_pages"].(float64) != 3 {
		t.Errorf("paging: %v", page)
	}
	_, _, b = doDemo(t, h, demoCall{method: "GET", path: "/users?page=99"})
	if len(decode(t, b)["data"].([]any)) != 0 {
		t.Error("past the end is empty")
	}
	_, _, b = doDemo(t, h, demoCall{method: "GET", path: "/users?limit=1000&page=-4"})
	if p := decode(t, b); p["limit"].(float64) != 10 || p["page"].(float64) != 1 {
		t.Errorf("out-of-range params fall back to defaults: %v", p)
	}

	code, _, b := doDemo(t, h, demoCall{method: "POST", path: "/users", body: `{"name":"Zed","email":"z@example.com"}`})
	created := decode(t, b)
	if code != 201 || created["id"].(float64) != 6 || created["role"] != "viewer" {
		t.Fatalf("create: %d %v", code, created)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/users", body: `{"name":"x"}`}); code != 422 {
		t.Errorf("missing email: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/users", body: "{"}); code != 400 {
		t.Errorf("bad json: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "PATCH", path: "/users"}); code != 405 {
		t.Errorf("users PATCH: %d", code)
	}

	if code, _, b := doDemo(t, h, demoCall{method: "GET", path: "/users/6"}); code != 200 || decode(t, b)["name"] != "Zed" {
		t.Errorf("get: %d %s", code, b)
	}
	code, _, b = doDemo(t, h, demoCall{method: "PUT", path: "/users/6", body: `{"name":"Zed2","role":"admin"}`})
	if upd := decode(t, b); code != 200 || upd["name"] != "Zed2" || upd["role"] != "admin" || upd["email"] != "z@example.com" {
		t.Errorf("partial update: %d %v", code, upd)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "PUT", path: "/users/6", body: "{"}); code != 400 {
		t.Errorf("put bad json: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "DELETE", path: "/users/6"}); code != 204 {
		t.Errorf("delete: %d", code)
	}
	for _, c := range []demoCall{
		{method: "GET", path: "/users/6"}, {method: "PUT", path: "/users/6", body: "{}"}, {method: "DELETE", path: "/users/6"}, {method: "GET", path: "/users/999"},
	} {
		if code, _, _ := doDemo(t, h, c); code != 404 {
			t.Errorf("%s %s after delete: %d", c.method, c.path, code)
		}
	}
	for _, p := range []string{"/users/abc", "/users/0", "/users/-1"} {
		if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: p}); code != 400 {
			t.Errorf("%s: %d", p, code)
		}
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "PATCH", path: "/users/1"}); code != 405 {
		t.Errorf("user PATCH: %d", code)
	}

	resetDemoState()
	_, _, b = doDemo(t, h, demoCall{method: "GET", path: "/users"})
	if decode(t, b)["total"].(float64) != 5 {
		t.Error("resetDemoState should restore the seed data")
	}
}

func TestDemoServer_PostsAndUpload(t *testing.T) {
	resetDemoState()
	h := newDemoMux()

	_, _, b := doDemo(t, h, demoCall{method: "GET", path: "/posts?page=3&limit=20"})
	p := decode(t, b)
	if p["total"].(float64) != 100 || len(p["data"].([]any)) != 20 || p["total_pages"].(float64) != 5 {
		t.Errorf("posts: %v", p)
	}
	_, _, b = doDemo(t, h, demoCall{method: "GET", path: "/posts?page=1000"})
	if len(decode(t, b)["data"].([]any)) != 0 {
		t.Error("past the end is empty")
	}
	_, _, b = doDemo(t, h, demoCall{method: "GET", path: "/posts?limit=500"})
	if decode(t, b)["limit"].(float64) != 10 {
		t.Error("limit above max falls back")
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/posts"}); code != 405 {
		t.Errorf("posts POST: %d", code)
	}

	code, _, b := doDemo(t, h, demoCall{method: "POST", path: "/files/upload", body: `{"filename":"a.png"}`})
	up := decode(t, b)
	if code != 201 || up["name"] != "a.png" || up["size"].(float64) != 1024 || up["content_type"] != "application/octet-stream" {
		t.Errorf("json upload defaults: %d %v", code, up)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/files/upload", body: `{}`}); code != 400 {
		t.Errorf("json upload without filename: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "GET", path: "/files/upload"}); code != 405 {
		t.Errorf("upload GET: %d", code)
	}

	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "hello.txt")
	fw.Write([]byte("12345"))
	mw.Close()
	code, _, b = doDemo(t, h, demoCall{method: "POST", path: "/files/upload", body: buf.String(), headers: map[string]string{"Content-Type": mw.FormDataContentType()}})
	if mp := decode(t, b); code != 201 || mp["name"] != "hello.txt" || mp["size"].(float64) != 5 {
		t.Errorf("multipart upload: %d %v", code, mp)
	}

	var wrong strings.Builder
	mw2 := multipart.NewWriter(&wrong)
	mw2.WriteField("other", "x")
	mw2.Close()
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/files/upload", body: wrong.String(), headers: map[string]string{"Content-Type": mw2.FormDataContentType()}}); code != 400 {
		t.Errorf("multipart without file field: %d", code)
	}
	if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: "/files/upload", body: "x", headers: map[string]string{"Content-Type": "multipart/form-data; boundary=zzz"}}); code != 400 {
		t.Errorf("malformed multipart: %d", code)
	}
}

func TestDemoServer_XMLAndHTML(t *testing.T) {
	resetDemoState()
	h := newDemoMux()
	for path, wantCT := range map[string]string{
		"/feed": "xml", "/sitemap.xml": "xml", "/data.xml": "xml", "/page": "html", "/report.html": "html",
	} {
		code, hdr, body := doDemo(t, h, demoCall{method: "GET", path: path})
		if code != 200 || !strings.Contains(hdr.Get("Content-Type"), wantCT) || len(body) < 50 {
			t.Errorf("%s: %d %q (%d bytes)", path, code, hdr.Get("Content-Type"), len(body))
		}
		if code, _, _ := doDemo(t, h, demoCall{method: "POST", path: path}); code != 405 {
			t.Errorf("%s POST: %d", path, code)
		}
	}
}

func TestDemoService_Lifecycle(t *testing.T) {
	isolateHome(t)
	original := t.TempDir()
	ws := newWS(t, original)
	d := NewDemoService(ws)

	if d.IsRunning() || d.GetPort() != 0 {
		t.Fatal("not running initially")
	}
	if msg := d.OpenSpecInBrowser(); msg == "" {
		t.Error("opening the spec while stopped should report an error")
	}

	res := d.StartDemo()
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	defer d.StopDemo()
	if !d.IsRunning() || res.Port < 8090 || d.GetPort() != res.Port || ws.GetWorkspacePath() != res.WorkspacePath {
		t.Fatalf("state after start: %+v running=%v ws=%q", res, d.IsRunning(), ws.GetWorkspacePath())
	}
	for _, sub := range []string{"environments", "requests", "scenarios"} {
		if _, err := os.Stat(filepath.Join(res.WorkspacePath, sub)); err != nil {
			t.Errorf("demo workspace missing %s: %v", sub, err)
		}
	}
	if again := d.StartDemo(); again.Port != res.Port || again.WorkspacePath != res.WorkspacePath {
		t.Errorf("second start should return the running state: %+v", again)
	}

	resp, err := http.Get("http://localhost:" + strconv.Itoa(res.Port) + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("demo server health: %d", resp.StatusCode)
	}

	env := ws.GetActiveEnvironment()
	if env == nil || env.Name != "local" {
		t.Fatalf("demo should select its local environment: %+v", env)
	}
	envFull, err := ws.GetEnvironment(env.Path)
	if err != nil {
		t.Fatal(err)
	}
	var base string
	for _, v := range envFull.Variables {
		if v.Name == "baseUrl" {
			base = v.Value
		}
	}
	if !strings.Contains(base, strconv.Itoa(res.Port)) {
		t.Errorf("extracted environment must point at the chosen port %d, got baseUrl=%q", res.Port, base)
	}

	d.StopDemo()
	if d.IsRunning() || d.GetPort() != 0 {
		t.Error("stopped")
	}
	if _, err := os.Stat(res.WorkspacePath); !os.IsNotExist(err) {
		t.Error("temp workspace should be removed")
	}
	if ws.GetWorkspacePath() != original || ws.GetActiveEnvironment() != nil {
		t.Errorf("previous workspace should be restored: %q", ws.GetWorkspacePath())
	}
	if _, err := http.Get("http://localhost:" + strconv.Itoa(res.Port) + "/health"); err == nil {
		t.Error("server should be down")
	}
}

func TestFindFreePortAndStaleCleanup(t *testing.T) {
	p, err := findFreePort(18090)
	if err != nil || p < 18090 {
		t.Fatalf("%d %v", p, err)
	}

	stale := filepath.Join(os.TempDir(), "clrsig-demo-staletest")
	keep := filepath.Join(os.TempDir(), "clrsig-demo-keeptest")
	other := filepath.Join(os.TempDir(), "unrelated-staletest")
	for _, d := range []string{stale, keep, other} {
		os.MkdirAll(d, 0o755)
	}
	defer os.RemoveAll(keep)
	defer os.RemoveAll(other)
	cleanupStaleDemoTempDirs(keep)
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale demo dir should be removed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("the current demo dir must be kept")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("unrelated dirs must never be touched")
	}
}

func TestStopDemo_WhenNotRunningKeepsWorkspace(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	ws := newWS(t, dir)
	d := NewDemoService(ws)
	d.StopDemo()
	if ws.GetWorkspacePath() != dir {
		t.Errorf("stopping a demo that is not running must not touch the workspace, got %q", ws.GetWorkspacePath())
	}
}
