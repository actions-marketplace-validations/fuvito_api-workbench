package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---- Demo server shared state ----

type demoUser struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

var (
	demoMu         sync.Mutex
	demoUsers      []demoUser
	demoNextUserID int
	demoStartedAt  time.Time
	demoReqCount   atomic.Int64
	demoToken      = "demo-jwt-abc123"
	demoAPIKey     = "demo-key-12345"
)

func resetDemoState() {
	demoMu.Lock()
	defer demoMu.Unlock()
	demoUsers = []demoUser{
		{ID: 1, Name: "Alice Smith", Email: "alice@example.com", Role: "admin", CreatedAt: "2026-01-01T10:00:00Z"},
		{ID: 2, Name: "Bob Jones", Email: "bob@example.com", Role: "developer", CreatedAt: "2026-01-02T09:30:00Z"},
		{ID: 3, Name: "Carol White", Email: "carol@example.com", Role: "designer", CreatedAt: "2026-01-03T14:15:00Z"},
		{ID: 4, Name: "David Brown", Email: "david@example.com", Role: "manager", CreatedAt: "2026-01-04T11:00:00Z"},
		{ID: 5, Name: "Eve Davis", Email: "eve@example.com", Role: "developer", CreatedAt: "2026-01-05T08:45:00Z"},
	}
	demoNextUserID = 6
	demoStartedAt = time.Now()
	demoReqCount.Store(0)
}

// ---- Mux builder ----

func newDemoMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", withCount(handleDemoHealth))
	mux.HandleFunc("/echo", withCount(handleDemoEcho))
	mux.HandleFunc("/status/", withCount(handleDemoStatus))
	mux.HandleFunc("/slow", withCount(handleDemoSlow))

	mux.HandleFunc("/auth/login", withCount(handleDemoLogin))
	mux.HandleFunc("/auth/profile", withCount(handleDemoProfile))
	mux.HandleFunc("/auth/basic", withCount(handleDemoBasicAuth))
	mux.HandleFunc("/auth/apikey", withCount(handleDemoAPIKey))

	mux.HandleFunc("/users", withCount(handleDemoUsers))
	mux.HandleFunc("/users/", withCount(handleDemoUser))

	mux.HandleFunc("/posts", withCount(handleDemoPosts))

	mux.HandleFunc("/files/upload", withCount(handleDemoUpload))

	mux.HandleFunc("/feed", withCount(handleDemoFeed))
	mux.HandleFunc("/sitemap.xml", withCount(handleDemoSitemap))
	mux.HandleFunc("/data.xml", withCount(handleDemoDataXML))
	mux.HandleFunc("/page", withCount(handleDemoHTMLPage))
	mux.HandleFunc("/report.html", withCount(handleDemoHTMLReport))

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withCount(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		demoReqCount.Add(1)
		h(w, r)
	}
}

func demoWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func demoErrJSON(w http.ResponseWriter, status int, msg string) {
	demoWriteJSON(w, status, map[string]any{"error": msg, "status": status})
}

func demoBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// ---- Handlers ----

func handleDemoHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	uptime := time.Since(demoStartedAt).Round(time.Second)
	demoWriteJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"version":         "1.0.0",
		"uptime":          uptime.String(),
		"requests_served": demoReqCount.Load(),
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"endpoints": []string{
			"GET  /health",
			"ANY  /echo",
			"GET  /status/{code}",
			"GET  /slow?delay=1500",
			"POST /auth/login",
			"GET  /auth/profile   (Bearer)",
			"GET  /auth/basic     (Basic demo:demo)",
			"GET  /auth/apikey    (X-Api-Key: demo-key-12345)",
			"GET  /users",
			"POST /users",
			"GET  /users/{id}",
			"PUT  /users/{id}",
			"DELETE /users/{id}",
			"GET  /posts?page=1&limit=10",
			"POST /files/upload",
			"GET  /feed           (Atom XML)",
			"GET  /sitemap.xml    (XML sitemap)",
			"GET  /data.xml       (structured XML)",
			"GET  /page           (HTML page)",
			"GET  /report.html    (HTML table report)",
		},
	})
}

func handleDemoEcho(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{}
	for name, vals := range r.Header {
		headers[name] = strings.Join(vals, ", ")
	}
	params := map[string]string{}
	for k, v := range r.URL.Query() {
		params[k] = strings.Join(v, ", ")
	}

	var bodyStr string
	if r.Body != nil {
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		bodyStr = string(buf[:n])
	}

	demoWriteJSON(w, http.StatusOK, map[string]any{
		"method":      r.Method,
		"path":        r.URL.Path,
		"headers":     headers,
		"queryParams": params,
		"body":        bodyStr,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"message":     "Echo endpoint — reflects your request back as JSON",
	})
}

func handleDemoStatus(w http.ResponseWriter, r *http.Request) {
	codeStr := strings.TrimPrefix(r.URL.Path, "/status/")
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 100 || code > 599 {
		demoErrJSON(w, http.StatusBadRequest, "invalid status code — use /status/200, /status/404, etc.")
		return
	}
	text := http.StatusText(code)
	if text == "" {
		text = "Unknown Status"
	}
	demoWriteJSON(w, code, map[string]any{
		"code":      code,
		"text":      text,
		"message":   fmt.Sprintf("You requested status %d (%s)", code, text),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleDemoSlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	delayStr := r.URL.Query().Get("delay")
	if delayStr == "" {
		delayStr = "2000"
	}
	delayMs, err := strconv.Atoi(delayStr)
	if err != nil || delayMs < 0 {
		demoErrJSON(w, http.StatusBadRequest, "delay must be a non-negative integer (ms)")
		return
	}
	if delayMs > 10000 {
		delayMs = 10000
	}
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
	demoWriteJSON(w, http.StatusOK, map[string]any{
		"delay":     delayMs,
		"message":   fmt.Sprintf("Responded after %dms — useful for testing timeouts and loading states", delayMs),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleDemoLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		demoErrJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Username != "admin" || body.Password != "secret" {
		demoErrJSON(w, http.StatusUnauthorized, "invalid credentials — try username: admin, password: secret")
		return
	}
	demoWriteJSON(w, http.StatusOK, map[string]any{
		"token":      demoToken,
		"token_type": "Bearer",
		"expires_in": 3600,
		"user": map[string]any{
			"id":    1,
			"name":  "Admin User",
			"email": "admin@example.com",
			"role":  "admin",
		},
		"hint": "Use this token as Bearer {{authToken}} in subsequent requests",
	})
}

func handleDemoProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token := demoBearerToken(r)
	if token != demoToken {
		demoErrJSON(w, http.StatusUnauthorized, "missing or invalid Bearer token — login first to get a token")
		return
	}
	demoWriteJSON(w, http.StatusOK, map[string]any{
		"id":          1,
		"name":        "Admin User",
		"email":       "admin@example.com",
		"role":        "admin",
		"permissions": []string{"read", "write", "delete", "admin"},
		"last_login":  time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
		"created_at":  "2026-01-01T00:00:00Z",
	})
}

func handleDemoBasicAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	user, pass, ok := r.BasicAuth()
	if !ok || user != "demo" || pass != "demo" {
		w.Header().Set("WWW-Authenticate", `Basic realm="Clear Signal Demo"`)
		demoErrJSON(w, http.StatusUnauthorized, "Basic auth required — username: demo, password: demo")
		return
	}
	demoWriteJSON(w, http.StatusOK, map[string]any{
		"message":      "Authenticated via Basic auth",
		"username":     user,
		"auth_method":  "Basic",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

func handleDemoAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	key := r.Header.Get("X-Api-Key")
	if key == "" {
		key = r.URL.Query().Get("api_key")
	}
	if key != demoAPIKey {
		demoErrJSON(w, http.StatusUnauthorized, "missing or invalid API key — use X-Api-Key: demo-key-12345")
		return
	}
	masked := key[:9] + strings.Repeat("*", len(key)-9)
	demoWriteJSON(w, http.StatusOK, map[string]any{
		"message":    "API key accepted",
		"key_prefix": masked,
		"scope":      []string{"read", "write"},
		"auth_method": "API Key",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

func handleDemoUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		page := 1
		limit := 10
		if n, err := strconv.Atoi(pageStr); err == nil && n > 0 {
			page = n
		}
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}

		demoMu.Lock()
		all := make([]demoUser, len(demoUsers))
		copy(all, demoUsers)
		demoMu.Unlock()

		total := len(all)
		start := (page - 1) * limit
		end := start + limit
		if start >= total {
			all = []demoUser{}
		} else {
			if end > total {
				end = total
			}
			all = all[start:end]
		}
		totalPages := (total + limit - 1) / limit
		if totalPages == 0 {
			totalPages = 1
		}

		demoWriteJSON(w, http.StatusOK, map[string]any{
			"data":        all,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		})

	case http.MethodPost:
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			demoErrJSON(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if body.Name == "" || body.Email == "" {
			demoErrJSON(w, http.StatusUnprocessableEntity, "name and email are required")
			return
		}
		if body.Role == "" {
			body.Role = "viewer"
		}
		demoMu.Lock()
		u := demoUser{
			ID:        demoNextUserID,
			Name:      body.Name,
			Email:     body.Email,
			Role:      body.Role,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		demoUsers = append(demoUsers, u)
		demoNextUserID++
		demoMu.Unlock()
		demoWriteJSON(w, http.StatusCreated, u)

	default:
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleDemoUser(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		demoErrJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}

	demoMu.Lock()
	defer demoMu.Unlock()

	idx := -1
	for i, u := range demoUsers {
		if u.ID == id {
			idx = i
			break
		}
	}

	switch r.Method {
	case http.MethodGet:
		if idx == -1 {
			demoErrJSON(w, http.StatusNotFound, fmt.Sprintf("user %d not found", id))
			return
		}
		demoWriteJSON(w, http.StatusOK, demoUsers[idx])

	case http.MethodPut:
		if idx == -1 {
			demoErrJSON(w, http.StatusNotFound, fmt.Sprintf("user %d not found", id))
			return
		}
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			demoErrJSON(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if body.Name != "" {
			demoUsers[idx].Name = body.Name
		}
		if body.Email != "" {
			demoUsers[idx].Email = body.Email
		}
		if body.Role != "" {
			demoUsers[idx].Role = body.Role
		}
		demoWriteJSON(w, http.StatusOK, demoUsers[idx])

	case http.MethodDelete:
		if idx == -1 {
			demoErrJSON(w, http.StatusNotFound, fmt.Sprintf("user %d not found", id))
			return
		}
		demoUsers = append(demoUsers[:idx], demoUsers[idx+1:]...)
		w.WriteHeader(http.StatusNoContent)

	default:
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleDemoPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page := 1
	limit := 10
	if n, err := strconv.Atoi(pageStr); err == nil && n > 0 {
		page = n
	}
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 50 {
		limit = n
	}

	total := 100
	start := (page - 1) * limit
	if start >= total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	totalPages := (total + limit - 1) / limit

	authors := []string{"alice@example.com", "bob@example.com", "carol@example.com"}
	tags := [][]string{
		{"api", "rest", "tutorial"},
		{"go", "backend", "performance"},
		{"testing", "automation", "ci"},
		{"design", "ux", "frontend"},
		{"devops", "kubernetes", "cloud"},
	}

	posts := make([]map[string]any, 0, end-start)
	for i := start; i < end; i++ {
		n := i + 1
		posts = append(posts, map[string]any{
			"id":         n,
			"title":      fmt.Sprintf("Post #%d: %s", n, []string{"Getting Started", "Advanced Patterns", "Best Practices", "Deep Dive", "Quick Tips"}[n%5]),
			"body":       fmt.Sprintf("This is the full content of post #%d. It demonstrates how the Clear Signal demo server generates realistic paginated data for testing.", n),
			"author":     authors[n%len(authors)],
			"tags":       tags[n%len(tags)],
			"views":      (n * 137) % 10000,
			"created_at": fmt.Sprintf("2026-%02d-%02dT%02d:00:00Z", (n%12)+1, (n%28)+1, n%24),
		})
	}

	demoWriteJSON(w, http.StatusOK, map[string]any{
		"data":        posts,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func handleDemoUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ct := r.Header.Get("Content-Type")

	// Accept multipart/form-data
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			demoErrJSON(w, http.StatusBadRequest, "invalid multipart body")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			demoErrJSON(w, http.StatusBadRequest, "field 'file' required in multipart form")
			return
		}
		defer file.Close()
		demoWriteJSON(w, http.StatusCreated, map[string]any{
			"id":          fmt.Sprintf("file-%d", time.Now().UnixMilli()%100000),
			"name":        header.Filename,
			"size":        header.Size,
			"content_type": header.Header.Get("Content-Type"),
			"uploaded_at": time.Now().UTC().Format(time.RFC3339),
			"message":     "File uploaded successfully via multipart/form-data",
		})
		return
	}

	// Accept application/json with simulated upload
	var body struct {
		Filename string `json:"filename"`
		MimeType string `json:"mimeType"`
		SizeBytes int64  `json:"sizeBytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Filename == "" {
		demoErrJSON(w, http.StatusBadRequest, "provide {\"filename\":\"photo.jpg\",\"mimeType\":\"image/jpeg\",\"sizeBytes\":204800} or use multipart/form-data with field 'file'")
		return
	}
	if body.MimeType == "" {
		body.MimeType = "application/octet-stream"
	}
	if body.SizeBytes == 0 {
		body.SizeBytes = 1024
	}
	demoWriteJSON(w, http.StatusCreated, map[string]any{
		"id":          fmt.Sprintf("file-%d", time.Now().UnixMilli()%100000),
		"name":        body.Filename,
		"size":        body.SizeBytes,
		"content_type": body.MimeType,
		"uploaded_at": time.Now().UTC().Format(time.RFC3339),
		"url":         fmt.Sprintf("/files/%s", body.Filename),
		"message":     "Simulated upload — send multipart/form-data for a real file upload",
	})
}

// ---- XML / HTML format handlers ----

func handleDemoFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	now := time.Now().UTC()
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Clear Signal Demo Feed</title>
  <subtitle>Sample Atom feed for testing XML response handling</subtitle>
  <link href="http://localhost/feed" rel="self"/>
  <id>urn:uuid:demo-feed-clrsig</id>
  <updated>%s</updated>
  <author>
    <name>Clear Signal Demo Server</name>
    <email>demo@clrsig.dev</email>
  </author>
  <entry>
    <id>urn:uuid:entry-001</id>
    <title>Getting Started with Clear Signal</title>
    <link href="http://localhost/posts/1"/>
    <updated>%s</updated>
    <summary>Learn how to import OpenAPI specs, run requests, and build multi-step scenarios.</summary>
    <category term="tutorial" scheme="http://localhost/categories"/>
    <content type="html">&lt;p&gt;Clear Signal is an &lt;strong&gt;open-source&lt;/strong&gt; API contract testing platform.&lt;/p&gt;</content>
  </entry>
  <entry>
    <id>urn:uuid:entry-002</id>
    <title>Contract Validation Deep Dive</title>
    <link href="http://localhost/posts/2"/>
    <updated>2026-10-15T09:00:00Z</updated>
    <summary>How Clear Signal validates API responses against OpenAPI 3.1 schemas automatically.</summary>
    <category term="api-testing" scheme="http://localhost/categories"/>
    <content type="text">Clear Signal supports both OpenAPI 3.0 and 3.1 for schema-based response validation.</content>
  </entry>
  <entry>
    <id>urn:uuid:entry-003</id>
    <title>CLI Integration for CI/CD</title>
    <link href="http://localhost/posts/3"/>
    <updated>2026-09-20T14:30:00Z</updated>
    <summary>Run your API scenarios in GitHub Actions with clrsig run and proper exit codes.</summary>
    <category term="devops" scheme="http://localhost/categories"/>
    <content type="text">Exit code 0 = all pass, 1 = assertion failures, 2 = execution errors.</content>
  </entry>
</feed>
`, now.Format(time.RFC3339), now.Format(time.RFC3339))
}

func handleDemoSitemap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	now := time.Now().UTC().Format("2006-01-02")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9
          http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd">
  <url>
    <loc>http://localhost/</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>http://localhost/users</loc>
    <lastmod>%s</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>http://localhost/posts</loc>
    <lastmod>%s</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>http://localhost/feed</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.6</priority>
  </url>
  <url>
    <loc>http://localhost/page</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>
`, now, now, now, now, now)
}

func handleDemoDataXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	demoMu.Lock()
	users := make([]demoUser, len(demoUsers))
	copy(users, demoUsers)
	demoMu.Unlock()

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<users xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
       xsi:noNamespaceSchemaLocation="users.xsd"
       count="%d"
       generated="%s">
`, len(users), time.Now().UTC().Format(time.RFC3339))
	for _, u := range users {
		fmt.Fprintf(w, `  <user id="%d" role="%s">
    <name>%s</name>
    <email>%s</email>
    <createdAt>%s</createdAt>
  </user>
`, u.ID, u.Role, u.Name, u.Email, u.CreatedAt)
	}
	fmt.Fprint(w, `</users>
`)
}

func handleDemoHTMLPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	demoMu.Lock()
	userCount := len(demoUsers)
	demoMu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>Clear Signal Demo — Welcome</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 760px; margin: 2rem auto; padding: 0 1rem; color: #1a1a1a; }
    h1 { color: #7c3aed; }
    nav a { margin-right: 1rem; color: #2563eb; text-decoration: none; }
    nav a:hover { text-decoration: underline; }
    .badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: .85em; font-weight: 600; }
    .ok { background: #d1fae5; color: #065f46; }
    .info { background: #dbeafe; color: #1e40af; }
    footer { margin-top: 3rem; font-size: .8em; color: #6b7280; border-top: 1px solid #e5e7eb; padding-top: 1rem; }
  </style>
</head>
<body>
  <header>
    <h1>Clear Signal Demo Server</h1>
    <nav>
      <a href="/health">Health</a>
      <a href="/users">Users</a>
      <a href="/posts">Posts</a>
      <a href="/feed">Atom Feed</a>
      <a href="/sitemap.xml">Sitemap</a>
      <a href="/data.xml">XML Data</a>
      <a href="/report.html">Report</a>
    </nav>
  </header>
  <main>
    <p>This server demonstrates <strong>multiple response formats</strong> for testing Clear Signal's response viewer.</p>
    <ul>
      <li>Status: <span class="badge ok">Running</span></li>
      <li>Users in store: <span class="badge info">%d</span></li>
      <li>Requests served: <span class="badge info">%d</span></li>
      <li>Started: <code>%s</code></li>
    </ul>
    <h2>Format Examples</h2>
    <table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;width:100%%">
      <thead style="background:#f3f4f6">
        <tr><th>Endpoint</th><th>Content-Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><a href="/feed">/feed</a></td><td>application/atom+xml</td><td>Atom syndication feed</td></tr>
        <tr><td><a href="/sitemap.xml">/sitemap.xml</a></td><td>application/xml</td><td>XML sitemap (sitemaps.org schema)</td></tr>
        <tr><td><a href="/data.xml">/data.xml</a></td><td>application/xml</td><td>Users as structured XML with attributes</td></tr>
        <tr><td><a href="/page">/page</a></td><td>text/html</td><td>This HTML page</td></tr>
        <tr><td><a href="/report.html">/report.html</a></td><td>text/html</td><td>HTML table report</td></tr>
      </tbody>
    </table>
    <h2>Try a curl command</h2>
    <pre style="background:#f9fafb;padding:1rem;border-radius:6px;overflow-x:auto">curl http://localhost:%d/feed
curl http://localhost:%d/data.xml
curl -H "Accept: text/html" http://localhost:%d/report.html</pre>
  </main>
  <footer>Clear Signal Demo Server &mdash; generated at %s</footer>
</body>
</html>
`, userCount, demoReqCount.Load(), demoStartedAt.UTC().Format(time.RFC3339),
		0, 0, 0, time.Now().UTC().Format(time.RFC3339))
}

func handleDemoHTMLReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	demoMu.Lock()
	users := make([]demoUser, len(demoUsers))
	copy(users, demoUsers)
	demoMu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <title>User Report — Clear Signal Demo</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 1.5rem; color: #111; }
    h1 { font-size: 1.25rem; color: #374151; }
    table { border-collapse: collapse; width: 100%%; font-size: .9rem; }
    th { background: #1e1b4b; color: #fff; padding: 8px 12px; text-align: left; }
    td { padding: 7px 12px; border-bottom: 1px solid #e5e7eb; }
    tr:hover td { background: #f5f3ff; }
    .role-admin { color: #7c3aed; font-weight: 600; }
    .role-developer { color: #2563eb; }
    .role-designer { color: #db2777; }
    .role-manager { color: #d97706; }
    .role-viewer { color: #6b7280; }
    .meta { margin-top: 1rem; font-size: .8rem; color: #9ca3af; }
  </style>
</head>
<body>
  <h1>User Report</h1>
  <table>
    <thead>
      <tr><th>#</th><th>Name</th><th>Email</th><th>Role</th><th>Created</th></tr>
    </thead>
    <tbody>
`)
	for _, u := range users {
		fmt.Fprintf(w, `      <tr>
        <td>%d</td>
        <td>%s</td>
        <td>%s</td>
        <td><span class="role-%s">%s</span></td>
        <td>%s</td>
      </tr>
`, u.ID, u.Name, u.Email, u.Role, u.Role, u.CreatedAt)
	}
	fmt.Fprintf(w, `    </tbody>
  </table>
  <p class="meta">%d user(s) &mdash; generated at %s &mdash; Clear Signal Demo Server</p>
</body>
</html>
`, len(users), time.Now().UTC().Format(time.RFC3339))
}
