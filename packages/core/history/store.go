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
		`ALTER TABLE history ADD COLUMN headers        TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE history ADD COLUMN body_type      TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE history ADD COLUMN body_content   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE history ADD COLUMN workspace_path TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := s.db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}

	// conform_runs: one row per clrsig conform execution.
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS conform_runs (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at         TEXT    NOT NULL,
			spec_hash          TEXT    NOT NULL,
			spec_path          TEXT    NOT NULL,
			total_operations   INTEGER NOT NULL DEFAULT 0,
			covered_operations INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return err
	}
	// conform_coverage: per-operation result for each run.
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS conform_coverage (
			run_id       INTEGER NOT NULL REFERENCES conform_runs(id),
			operation_id TEXT    NOT NULL,
			method       TEXT    NOT NULL,
			path         TEXT    NOT NULL,
			covered      INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return err
	}
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_conform_runs_hash     ON conform_runs(spec_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_conform_coverage_run  ON conform_coverage(run_id)`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// Record appends a request/response pair to history and returns the new entry ID.
// Sensitive header values (Authorization, Cookie, etc.) are redacted before storage.
func (s *Store) Record(workspacePath, method, url string, statusCode int, statusText string, durationMs, sizeBytes int64, headers []Header, bodyType, bodyContent string) (int64, error) {
	hJSON, err := json.Marshal(redactHeaders(headers))
	if err != nil {
		hJSON = []byte("[]")
	}
	res, err := s.db.Exec(
		`INSERT INTO history (executed_at, workspace_path, method, url, status_code, status_text, duration_ms, size_bytes, headers, body_type, body_content)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339),
		workspacePath, method, url, statusCode, statusText, durationMs, sizeBytes,
		string(hJSON), bodyType, bodyContent,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Recent returns up to limit entries, most recent first.
// If workspacePath is non-empty, only entries for that workspace are returned.
func (s *Store) Recent(workspacePath string, limit int) ([]Entry, error) {
	var rows *sql.Rows
	var err error
	if workspacePath != "" {
		rows, err = s.db.Query(
			`SELECT id, executed_at, method, url, status_code, status_text, duration_ms, size_bytes, headers, body_type, body_content
			 FROM history WHERE workspace_path = ? ORDER BY id DESC LIMIT ?`,
			workspacePath, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, executed_at, method, url, status_code, status_text, duration_ms, size_bytes, headers, body_type, body_content
			 FROM history ORDER BY id DESC LIMIT ?`,
			limit,
		)
	}
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

// Clear deletes history entries.
// If workspacePath is non-empty, only entries for that workspace are deleted.
func (s *Store) Clear(workspacePath string) error {
	if workspacePath != "" {
		_, err := s.db.Exec(`DELETE FROM history WHERE workspace_path = ?`, workspacePath)
		return err
	}
	_, err := s.db.Exec(`DELETE FROM history`)
	return err
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ConformOp captures per-operation coverage data for a single conform run.
type ConformOp struct {
	OperationID string
	Method      string
	Path        string
	Covered     bool // true when an HTTP response was received (StatusCode > 0)
}

// CumulativeReport summarises operation coverage aggregated across all recorded
// runs for a given spec (identified by its content hash).
type CumulativeReport struct {
	SpecPath        string
	SpecHash        string
	RunCount        int
	TotalOperations int
	EverCoveredOps  []string // "METHOD /path" for ops covered in at least one run
	NeverCoveredOps []string // "METHOD /path" for ops never covered
	Percent         float64
}

// RecordConformRun persists the per-operation results of a conformance run.
// specPath is a human-readable label (file path or URL); specHash is a SHA256
// hex string of the spec bytes used to group runs of the same spec together.
func (s *Store) RecordConformRun(specPath, specHash string, ops []ConformOp) error {
	total := len(ops)
	covered := 0
	for _, op := range ops {
		if op.Covered {
			covered++
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO conform_runs (created_at, spec_hash, spec_path, total_operations, covered_operations) VALUES (?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339), specHash, specPath, total, covered,
	)
	if err != nil {
		return err
	}
	runID, _ := res.LastInsertId()

	for _, op := range ops {
		cov := 0
		if op.Covered {
			cov = 1
		}
		if _, err := tx.Exec(
			`INSERT INTO conform_coverage (run_id, operation_id, method, path, covered) VALUES (?, ?, ?, ?, ?)`,
			runID, op.OperationID, op.Method, op.Path, cov,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CumulativeCoverage returns aggregated coverage for all runs of the spec
// identified by specHash. Returns nil, nil when no runs have been recorded yet.
func (s *Store) CumulativeCoverage(specHash string) (*CumulativeReport, error) {
	var runCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM conform_runs WHERE spec_hash = ?`, specHash,
	).Scan(&runCount); err != nil {
		return nil, err
	}
	if runCount == 0 {
		return nil, nil
	}

	var specPath string
	_ = s.db.QueryRow(
		`SELECT spec_path FROM conform_runs WHERE spec_hash = ? ORDER BY id DESC LIMIT 1`, specHash,
	).Scan(&specPath)

	rows, err := s.db.Query(`
		SELECT cc.operation_id, cc.method, cc.path, MAX(cc.covered) AS ever_covered
		FROM conform_coverage cc
		JOIN conform_runs cr ON cc.run_id = cr.id
		WHERE cr.spec_hash = ?
		GROUP BY cc.operation_id, cc.method, cc.path
		ORDER BY cc.method, cc.path
	`, specHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rep := &CumulativeReport{
		SpecPath: specPath,
		SpecHash: specHash,
		RunCount: runCount,
	}
	for rows.Next() {
		var opID, method, path string
		var everCovered int
		if err := rows.Scan(&opID, &method, &path, &everCovered); err != nil {
			return nil, err
		}
		rep.TotalOperations++
		label := method + " " + path
		if everCovered == 1 {
			rep.EverCoveredOps = append(rep.EverCoveredOps, label)
		} else {
			rep.NeverCoveredOps = append(rep.NeverCoveredOps, label)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if rep.TotalOperations > 0 {
		rep.Percent = float64(len(rep.EverCoveredOps)) / float64(rep.TotalOperations) * 100
	}
	return rep, nil
}
