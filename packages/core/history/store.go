package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Entry is a single recorded request/response pair.
type Entry struct {
	ID         int64
	ExecutedAt time.Time
	Method     string
	URL        string
	StatusCode int
	StatusText string
	DurationMs int64
	SizeBytes  int64
}

// Store is a persistent history of executed requests.
// Auth tokens and request bodies are never written to protect secrets.
type Store struct {
	db *sql.DB
}

// DefaultPath returns the platform-appropriate path for the history database.
// The parent directory is created if it does not exist.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	dir := filepath.Join(home, ".apidesk")
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
	_, err := s.db.Exec(`
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
	`)
	return err
}

// Record appends a request/response pair to history and returns the new entry ID.
func (s *Store) Record(method, url string, statusCode int, statusText string, durationMs, sizeBytes int64) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO history (executed_at, method, url, status_code, status_text, duration_ms, size_bytes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339),
		method, url, statusCode, statusText, durationMs, sizeBytes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Recent returns up to limit entries, most recent first.
func (s *Store) Recent(limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		`SELECT id, executed_at, method, url, status_code, status_text, duration_ms, size_bytes
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
		var executedAt string
		if err := rows.Scan(&e.ID, &executedAt, &e.Method, &e.URL, &e.StatusCode, &e.StatusText, &e.DurationMs, &e.SizeBytes); err != nil {
			return nil, err
		}
		e.ExecutedAt, _ = time.Parse(time.RFC3339, executedAt)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
