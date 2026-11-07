// Example server for end-to-end testing of ApiDesk.
// Run: go run ./examples/server
// Listens on :8080 and prints all available endpoints on startup.
package main

import (
	"encoding/json"
	"fmt"
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
		{ID: 1, Name: "Alice Smith", Email: "alice@example.com", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: 2, Name: "Bob Jones", Email: "bob@example.com", CreatedAt: "2026-01-02T00:00:00Z"},
	}
	nextID     = 3
	validToken = "demo-token-12345"

	loginMu          sync.Mutex
	loginAttempts    int
	loginWindowStart = time.Now()
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/login", handleLogin)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/", handleUser)
	mux.HandleFunc("/protected", handleProtected)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/maintenance", handleMaintenance)
	mux.HandleFunc("/admin/reset", handleAdminReset)

	fmt.Println("ApiDesk example server")
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

func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// methodNotAllowed sets the Allow header (RFC 9110 §15.5.6) before writing 405.
func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	errJSON(w, http.StatusMethodNotAllowed, "method not allowed")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
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
		errJSON(w, http.StatusBadRequest, "invalid JSON")
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
			errJSON(w, http.StatusTooManyRequests, "too many failed login attempts")
			return
		}
		remaining := rateLimitMax - count
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		errJSON(w, http.StatusUnauthorized, "invalid credentials")
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
	errJSON(w, http.StatusServiceUnavailable, "service temporarily unavailable")
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
		errJSON(w, http.StatusUnauthorized, "missing or invalid token")
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
		writeJSON(w, http.StatusOK, users)

	case http.MethodPost:
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errJSON(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if body.Name == "" || body.Email == "" {
			errJSON(w, http.StatusUnprocessableEntity, "name and email are required")
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
		errJSON(w, http.StatusBadRequest, "invalid user id")
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
			errJSON(w, http.StatusNotFound, "user not found")
			return
		}
		writeJSON(w, http.StatusOK, users[idx])

	case http.MethodPut:
		if idx == -1 {
			errJSON(w, http.StatusNotFound, "user not found")
			return
		}
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errJSON(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if body.Name != "" {
			users[idx].Name = body.Name
		}
		if body.Email != "" {
			users[idx].Email = body.Email
		}
		writeJSON(w, http.StatusOK, users[idx])

	case http.MethodDelete:
		if idx == -1 {
			errJSON(w, http.StatusNotFound, "user not found")
			return
		}
		users = append(users[:idx], users[idx+1:]...)
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w, "GET, PUT, DELETE")
	}
}
