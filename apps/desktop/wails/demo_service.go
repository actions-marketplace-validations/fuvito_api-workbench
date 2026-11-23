package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// demoWorkspacePrefix names the temporary workspace the Demo Server opens.
const demoWorkspacePrefix = "clrsig-demo-"

//go:embed demo_workspace
var demoWorkspaceFS embed.FS

//go:embed help/demo-openapi.yaml
var demoOpenAPIYAML []byte

// DemoResult is returned to the frontend by StartDemo.
type DemoResult struct {
	WorkspacePath string `json:"workspacePath"`
	Port          int    `json:"port"`
	Error         string `json:"error,omitempty"`
}

type DemoService struct {
	ctx           context.Context
	ws            *WorkspaceService
	server        *http.Server
	tempDir       string
	port          int
	mu            sync.Mutex
	portFn        func() int // configured port from Settings; nil or 0 = automatic
	prevWorkspace string
	prevEnvPath   string
}

func NewDemoService(ws *WorkspaceService) *DemoService {
	return &DemoService{ctx: context.Background(), ws: ws}
}

func (d *DemoService) setContext(ctx context.Context) {
	d.ctx = ctx
}

func (d *DemoService) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.server != nil
}

func (d *DemoService) GetPort() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.port
}

// cleanupStaleDemoTempDirs removes clrsig-demo-* directories left behind by a session that ended without
// stopping its server. A folder another running copy of the app keeps alive is left alone.
func cleanupStaleDemoTempDirs(currentTempDir string) {
	removeStaleTempWorkspaces(os.TempDir(), currentTempDir, demoWorkspacePrefix)
}

// StartDemo starts the in-process mock server, extracts the demo workspace to a
// temp directory, and opens it. Safe to call when already running (returns current state).
func (d *DemoService) StartDemo() DemoResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.server != nil {
		return DemoResult{WorkspacePath: d.tempDir, Port: d.port}
	}

	if strings.HasPrefix(filepath.Base(d.ws.workspacePath), conformanceWorkspacePrefix) {
		return DemoResult{Error: "stop the Conformance Server first: its workspace is open"}
	}

	cleanupStaleDemoTempDirs("")

	configured := 0
	if d.portFn != nil {
		configured = d.portFn()
	}
	port, err := choosePort(configured, 8090, "the Demo Server port")
	if err != nil {
		return DemoResult{Error: err.Error()}
	}

	tmpDir, err := os.MkdirTemp("", "clrsig-demo-*")
	if err != nil {
		return DemoResult{Error: "could not create temp directory: " + err.Error()}
	}

	claimTempWorkspace(tmpDir)
	if err := extractDemoWorkspace(tmpDir, port); err != nil {
		os.RemoveAll(tmpDir)
		return DemoResult{Error: "could not extract demo workspace: " + err.Error()}
	}

	resetDemoState()

	mux := newDemoMux()
	srv := &http.Server{
		Addr:         fmt.Sprintf("localhost:%d", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		os.RemoveAll(tmpDir)
		return DemoResult{Error: "could not bind port: " + err.Error()}
	}
	go srv.Serve(ln) //nolint:errcheck

	d.server = srv
	d.tempDir = tmpDir
	d.port = port
	d.prevWorkspace = d.ws.workspacePath
	d.prevEnvPath = d.ws.activeEnvPath

	d.ws.openWorkspaceDirect(tmpDir)

	return DemoResult{WorkspacePath: tmpDir, Port: port}
}

// StopDemo shuts down the server and removes the temp workspace.
func (d *DemoService) StopDemo() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Without this guard, a stop while nothing is running would overwrite the
	// open workspace with the empty "previous workspace" values.
	if d.server == nil && d.tempDir == "" {
		return
	}

	if d.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		d.server.Shutdown(ctx) //nolint:errcheck
		d.server = nil
	}

	if d.tempDir != "" {
		os.RemoveAll(d.tempDir)
		d.tempDir = ""
	}
	d.port = 0

	// Restore the workspace that was open before the demo started.
	d.ws.workspacePath = d.prevWorkspace
	d.ws.activeEnvPath = d.prevEnvPath
	d.ws.activeEnv = nil
	if d.prevEnvPath != "" {
		if env, err := d.ws.loadEnvFile(d.prevEnvPath); err == nil {
			d.ws.activeEnv = env
		}
	}
	d.prevWorkspace = ""
	d.prevEnvPath = ""
}

// OpenSpecInBrowser opens the Swagger UI docs page for the running demo server
// in the system default browser. Returns a non-empty error string on failure.
func (d *DemoService) OpenSpecInBrowser() string {
	d.mu.Lock()
	port := d.port
	d.mu.Unlock()
	if port == 0 {
		return "demo server is not running"
	}
	if err := openURL(fmt.Sprintf("http://localhost:%d/docs", port)); err != nil {
		return err.Error()
	}
	return ""
}

// ---- helpers ----

// sanitizePort keeps a configured port in the unprivileged range; anything else means automatic.
func sanitizePort(p int) int {
	if p < 1024 || p > 65535 {
		return 0
	}
	return p
}

// choosePort returns the configured port when it is free, or the first free port from start when
// none is configured. A configured port that is taken is an error the user can act on.
func choosePort(configured, start int, setting string) (int, error) {
	if configured == 0 {
		return findFreePort(start)
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", configured))
	if err != nil {
		return 0, fmt.Errorf("port %d is already in use. Change %s in Settings, or set it to 0 to pick a free port automatically", configured, setting)
	}
	ln.Close()
	return configured, nil
}

func findFreePort(start int) (int, error) {
	for port := start; port < start+30; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d–%d", start, start+30)
}

func extractDemoWorkspace(dstDir string, port int) error {
	if err := fs.WalkDir(demoWorkspaceFS, "demo_workspace", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("demo_workspace", path)
		dst := filepath.Join(dstDir, filepath.FromSlash(rel))

		if entry.IsDir() {
			return os.MkdirAll(dst, 0755)
		}

		data, err := demoWorkspaceFS.ReadFile(path)
		if err != nil {
			return err
		}

		content := strings.ReplaceAll(string(data), "DEMO_PORT", fmt.Sprintf("%d", port))

		return os.WriteFile(dst, []byte(content), 0644)
	}); err != nil {
		return err
	}

	// The scenarios name the spec by URL, but the Check Conformance page lists spec files found in
	// the workspace, so the same spec is also written as a file, pointing at this port.
	if err := os.MkdirAll(filepath.Join(dstDir, "specs"), 0755); err != nil {
		return err
	}
	spec := strings.ReplaceAll(string(demoOpenAPIYAML), "http://localhost:8090", fmt.Sprintf("http://localhost:%d", port))
	if err := os.WriteFile(filepath.Join(dstDir, "specs", "openapi.yaml"), []byte(spec), 0644); err != nil {
		return err
	}

	// Protect environment files from accidental git commits in the temp dir.
	return os.WriteFile(filepath.Join(dstDir, "environments", ".gitignore"), []byte("*\n"), 0644)
}
