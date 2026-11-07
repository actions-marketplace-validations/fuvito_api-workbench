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

//go:embed demo_workspace
var demoWorkspaceFS embed.FS

// DemoResult is returned to the frontend by StartDemo.
type DemoResult struct {
	WorkspacePath string `json:"workspacePath"`
	Port          int    `json:"port"`
	Error         string `json:"error,omitempty"`
}

type DemoService struct {
	ctx              context.Context
	ws               *WorkspaceService
	server           *http.Server
	tempDir          string
	port             int
	mu               sync.Mutex
	prevWorkspace    string
	prevEnvPath      string
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

// StartDemo starts the in-process mock server, extracts the demo workspace to a
// temp directory, and opens it. Safe to call when already running (returns current state).
func (d *DemoService) StartDemo() DemoResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.server != nil {
		return DemoResult{WorkspacePath: d.tempDir, Port: d.port}
	}

	port, err := findFreePort(8090)
	if err != nil {
		return DemoResult{Error: "could not find free port: " + err.Error()}
	}

	tmpDir, err := os.MkdirTemp("", "apidesk-demo-*")
	if err != nil {
		return DemoResult{Error: "could not create temp directory: " + err.Error()}
	}

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

// ---- helpers ----

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

	// Protect environment files from accidental git commits in the temp dir.
	return os.WriteFile(filepath.Join(dstDir, "environments", ".gitignore"), []byte("*\n"), 0644)
}
