// Example server for end-to-end testing of Clear Signal.
// Run: go run ./examples/server
// Listens on :8080 and prints all available endpoints on startup.
package main

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

const (
	rateLimitMax    = 2             // failed login attempts before 429
	rateLimitWindow = time.Minute
)

var (
	mu      sync.Mutex
	users   = []User{
		{ID: 1, Name: "Alice Smith",   Email: "alice@example.com",   CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: 2, Name: "Bob Jones",     Email: "bob@example.com",     CreatedAt: "2026-01-02T00:00:00Z"},
		{ID: 3, Name: "Carol White",   Email: "carol@example.com",   CreatedAt: "2026-01-03T00:00:00Z"},
		{ID: 4, Name: "David Brown",   Email: "david@example.com",   CreatedAt: "2026-01-04T00:00:00Z"},
		{ID: 5, Name: "Eva Green",     Email: "eva@example.com",     CreatedAt: "2026-01-05T00:00:00Z"},
	}
	nextID     = 6
	validToken = "demo-token-12345"

	loginMu          sync.Mutex
	loginAttempts    int
	loginWindowStart = time.Now()
)

// newMux builds and returns the server's request mux.
// Extracted so tests can spin up a fresh httptest.Server without binding :8080.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", handleLogin)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/", handleUser)
	mux.HandleFunc("/protected", handleProtected)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/maintenance", handleMaintenance)
	mux.HandleFunc("/admin/reset", handleAdminReset)
	return mux
}

func main() {
	mux := newMux()

	fmt.Println("Clear Signal example server")
	fmt.Println("Listening on http://localhost:8080")
	fmt.Println()
	fmt.Println("  POST   /auth/login           { \"username\": \"admin\", \"password\": \"secret\" }")
	fmt.Println("  GET    /health")
	fmt.Println("  GET    /users")
	fmt.Println("  POST   /users                { \"name\": \"...\", \"email\": \"...\" }")
	fmt.Println("  GET    /users/:id")
	fmt.Println("  PUT    /users/:id            { \"name\": \"...\", \"email\": \"...\" }")
	fmt.Println("  DELETE /users/:id")
	fmt.Println("  GET    /protected            requires Authorization: Bearer demo-token-12345")
	fmt.Println("  GET    /maintenance          always 503 with Retry-After (conformance testing)")
	fmt.Println("  POST   /admin/reset          reset rate limit counters (testing only)")
	fmt.Println()

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("server error:", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// userETag returns a quoted ETag derived from the user's JSON representation.
func userETag(u User) string {
	data, _ := json.Marshal(u)
	return fmt.Sprintf(`"%x"`, crc32.ChecksumIEEE(data))
}

// problemJSON writes an RFC 7807 Problem Details response (application/problem+json).
func problemJSON(w http.ResponseWriter, status int, problemType, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type":   problemType,
		"title":  title,
		"status": status,
		"detail": detail,
	})
}

// methodNotAllowed sets the Allow header (RFC 9110 §15.5.6) before writing 405.
func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	problemJSON(w, http.StatusMethodNotAllowed, "/errors/method-not-allowed", "Method Not Allowed", "allowed methods: "+allowed)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	// Content negotiation: honour Accept header for JSON vs plain text.
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/plain") && !strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "status=ok time=%s\n", time.Now().UTC().Format(time.RFC3339))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		problemJSON(w, http.StatusBadRequest, "/errors/bad-request", "Bad Request", "request body must be valid JSON")
		return
	}
	if body.Username != "admin" || body.Password != "secret" {
		loginMu.Lock()
		now := time.Now()
		if now.Sub(loginWindowStart) > rateLimitWindow {
			loginAttempts = 0
			loginWindowStart = now
		}
		loginAttempts++
		count := loginAttempts
		reset := loginWindowStart.Add(rateLimitWindow).Unix()
		loginMu.Unlock()

		if count > rateLimitMax {
			w.Header().Set("Retry-After", "60")
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
			problemJSON(w, http.StatusTooManyRequests, "/errors/rate-limit-exceeded", "Too Many Requests", "too many failed login attempts, retry after 60 seconds")
			return
		}
		remaining := rateLimitMax - count
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
		problemJSON(w, http.StatusUnauthorized, "/errors/unauthorized", "Unauthorized", "invalid credentials")
		return
	}
	loginMu.Lock()
	loginAttempts = 0
	loginWindowStart = time.Now()
	loginMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{
		"token":   validToken,
		"message": "Login successful",
	})
}

func handleMaintenance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	w.Header().Set("Retry-After", "120")
	problemJSON(w, http.StatusServiceUnavailable, "/errors/service-unavailable", "Service Unavailable", "service is temporarily unavailable, retry after 120 seconds")
}

func handleAdminReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	loginMu.Lock()
	loginAttempts = 0
	loginWindowStart = time.Now()
	loginMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"message": "rate limit counters reset"})
}

func handleProtected(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+validToken {
		w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
		problemJSON(w, http.StatusUnauthorized, "/errors/unauthorized", "Unauthorized", "missing or invalid authentication token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Access granted",
		"user":    "admin",
	})
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.Lock()
		defer mu.Unlock()

		// Parse pagination params: ?page=1&limit=1  (1-based page, default limit 100)
		page := 1
		limit := 100
		if v := r.URL.Query().Get("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 {
				page = n
			}
		}
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
				limit = n
			}
		}

		total := len(users)
		start := (page - 1) * limit
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		page_users := users[start:end]

		w.Header().Set("X-Total-Count", strconv.Itoa(total))

		// Build Link header (RFC 8288)
		base := "http://localhost:8080/users"
		var links []string
		if page > 1 {
			links = append(links, fmt.Sprintf(`<%s?page=%d&limit=%d>; rel="prev"`, base, page-1, limit))
			links = append(links, fmt.Sprintf(`<%s?page=1&limit=%d>; rel="first"`, base, limit))
		}
		totalPages := (total + limit - 1) / limit
		if totalPages == 0 {
			totalPages = 1
		}
		if page < totalPages {
			links = append(links, fmt.Sprintf(`<%s?page=%d&limit=%d>; rel="next"`, base, page+1, limit))
			links = append(links, fmt.Sprintf(`<%s?page=%d&limit=%d>; rel="last"`, base, totalPages, limit))
		}
		if len(links) > 0 {
			w.Header().Set("Link", strings.Join(links, ", "))
		}

		writeJSON(w, http.StatusOK, page_users)

	case http.MethodPost:
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			problemJSON(w, http.StatusBadRequest, "/errors/bad-request", "Bad Request", "request body must be valid JSON")
			return
		}
		if body.Name == "" || body.Email == "" {
			problemJSON(w, http.StatusUnprocessableEntity, "/errors/validation-error", "Unprocessable Entity", "name and email are required")
			return
		}
		mu.Lock()
		u := User{
			ID:        nextID,
			Name:      body.Name,
			Email:     body.Email,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		users = append(users, u)
		nextID++
		mu.Unlock()
		writeJSON(w, http.StatusCreated, u)

	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		problemJSON(w, http.StatusBadRequest, "/errors/bad-request", "Bad Request", "user id must be a positive integer")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	idx := -1
	for i, u := range users {
		if u.ID == id {
			idx = i
			break
		}
	}

	switch r.Method {
	case http.MethodGet:
		if idx == -1 {
			problemJSON(w, http.StatusNotFound, "/errors/not-found", "Not Found", "user not found")
			return
		}
		etag := userETag(users[idx])
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		writeJSON(w, http.StatusOK, users[idx])

	case http.MethodPut:
		if idx == -1 {
			problemJSON(w, http.StatusNotFound, "/errors/not-found", "Not Found", "user not found")
			return
		}
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			problemJSON(w, http.StatusBadRequest, "/errors/bad-request", "Bad Request", "request body must be valid JSON")
			return
		}
		if body.Name != "" {
			users[idx].Name = body.Name
		}
		if body.Email != "" {
			users[idx].Email = body.Email
		}
		w.Header().Set("ETag", userETag(users[idx]))
		writeJSON(w, http.StatusOK, users[idx])

	case http.MethodDelete:
		if idx == -1 {
			problemJSON(w, http.StatusNotFound, "/errors/not-found", "Not Found", "user not found")
			return
		}
		users = append(users[:idx], users[idx+1:]...)
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w, "GET, PUT, DELETE")
	}
}
