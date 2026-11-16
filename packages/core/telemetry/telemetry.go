package telemetry

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const endpoint = "https://telemetry.clrsig.com/v1/event"

// Event carries the data recorded for one CLI invocation.
// No URLs, headers, bodies, or personal data are included.
type Event struct {
	Command        string // "conform" | "run" | "diff"
	Result         string // "pass" | "fail" | "error"
	DurationMs     int64
	StepCount      int
	AssertionCount int
}

// Client sends telemetry events when opt-in is enabled.
// All network calls are fire-and-forget; errors are silently swallowed.
type Client struct {
	enabled   bool
	installID string
	version   string
}

// New creates a Client. When enabled is false every Record call is a no-op.
func New(enabled bool, dataDir, version string) *Client {
	if !enabled {
		return &Client{}
	}
	return &Client{
		enabled:   true,
		installID: loadOrCreateInstallID(dataDir),
		version:   version,
	}
}

// Record fires the event asynchronously and returns immediately.
func (c *Client) Record(e Event) {
	if !c.enabled {
		return
	}
	go c.send(e)
}

type payload struct {
	InstallID      string `json:"installId"`
	Version        string `json:"version"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Command        string `json:"cmd"`
	Result         string `json:"result"`
	DurationMs     int64  `json:"durationMs"`
	StepCount      int    `json:"stepCount,omitempty"`
	AssertionCount int    `json:"assertionCount,omitempty"`
	Timestamp      string `json:"ts"`
}

func (c *Client) send(e Event) {
	p := payload{
		InstallID:      c.installID,
		Version:        c.version,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		Command:        e.Command,
		Result:         e.Result,
		DurationMs:     e.DurationMs,
		StepCount:      e.StepCount,
		AssertionCount: e.AssertionCount,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func loadOrCreateInstallID(dataDir string) string {
	path := filepath.Join(dataDir, "install-id")
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		return string(data)
	}
	id := newID()
	_ = os.MkdirAll(dataDir, 0o700)
	_ = os.WriteFile(path, []byte(id), 0o600)
	return id
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
