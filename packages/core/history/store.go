package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Header is a name/value pair stored with a history entry.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Entry is a single recorded request/response pair.
type Entry struct {
	ID          int64
	ExecutedAt  time.Time
	Method      string
	URL         string
	StatusCode  int
	StatusText  string
	DurationMs  int64
	SizeBytes   int64
	Headers     []Header
	BodyType    string
	BodyContent string
}

// Store is a persistent history of executed requests.
// Sensitive headers (Authorization, Cookie, etc.) are redacted before storage.
type Store struct {
	db *sql.DB
}

var sensitiveHeaderNames = map[string]bool{
	"authorization":     true,
	"cookie":            true,
	"proxy-authorization": true,
	"x-api-key":         true,
	"x-auth-token":      true,
	"x-access-token":    true,
	"x-secret":          true,
}

func redactHeaders(headers []Header) []Header {
	out := make([]Header, len(headers))
	for i, h := range headers {
		if sensitiveHeaderNames[strings.ToLower(h.Name)] {
			out[i] = Header{Name: h.Name, Value: "[redacted]"}
		} else {
			out[i] = h
		}
	}
	return out
}

// DefaultPath returns the platform-appropriate path for the history database.
// The parent directory is created if it does not exist.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	dir := filepath.Join(home, ".clrsig")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating data directory: %w", err)
	}
	return filepath.Join(dir, "history.db"), nil
}

// Open opens (or creates) the history database at path and runs schema migrations.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening history database: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite supports one writer at a time
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating history database: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			executed_at TEXT    NOT NULL,
			method      TEXT    NOT NULL,
			url         TEXT    NOT NULL,
			status_code INTEGER NOT NULL DEFAULT 0,
			status_text TEXT    NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			size_bytes  INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return err
	}
	// Additive migration: add request-detail columns if not present.
	for _, stmt := range []string{
		`ALTER TABLE history ADD COLUMN headers      TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE history ADD COLUMN body_type    TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE history ADD COLUMN body_content TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := s.db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}
	return nil
}

// Record appends a request/response pair to history and returns the new entry ID.
// Sensitive header values (Authorization, Cookie, etc.) are redacted before storage.
func (s *Store) Record(method, url string, statusCode int, statusText string, durationMs, sizeBytes int64, headers []Header, bodyType, bodyContent string) (int64, error) {
	hJSON, err := json.Marshal(redactHeaders(headers))
	if err != nil {
		hJSON = []byte("[]")
	}
	res, err := s.db.Exec(
		`INSERT INTO history (executed_at, method, url, status_code, status_text, duration_ms, size_bytes, headers, body_type, body_content)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339),
		method, url, statusCode, statusText, durationMs, sizeBytes,
		string(hJSON), bodyType, bodyContent,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Recent returns up to limit entries, most recent first.
func (s *Store) Recent(limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		`SELECT id, executed_at, method, url, status_code, status_text, duration_ms, size_bytes, headers, body_type, body_content
		 FROM history ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var executedAt, headersJSON string
		if err := rows.Scan(&e.ID, &executedAt, &e.Method, &e.URL, &e.StatusCode, &e.StatusText, &e.DurationMs, &e.SizeBytes, &headersJSON, &e.BodyType, &e.BodyContent); err != nil {
			return nil, err
		}
		e.ExecutedAt, _ = time.Parse(time.RFC3339, executedAt)
		if headersJSON != "" {
			_ = json.Unmarshal([]byte(headersJSON), &e.Headers)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Clear deletes all history entries.
func (s *Store) Clear() error {
	_, err := s.db.Exec(`DELETE FROM history`)
	return err
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
