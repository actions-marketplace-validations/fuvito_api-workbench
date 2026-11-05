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
		w.Header().Set("WWW-Authenticate", `Basic realm="ApiDesk Demo"`)
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
			"body":       fmt.Sprintf("This is the full content of post #%d. It demonstrates how the ApiDesk demo server generates realistic paginated data for testing.", n),
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
