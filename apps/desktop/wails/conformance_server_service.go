package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const conformanceWorkspacePrefix = "clrsig-conformance-"

// ConformanceServerService runs the deliberately imperfect Conformance Server.
// Unlike the demo it opens no workspace: it only listens on a local port.
type ConformanceServerService struct {
	mu     sync.Mutex
	server *http.Server
	port   int
	portFn func() int // configured port from Settings; nil or 0 = automatic

	// Set while OpenWorkspace has swapped the open workspace for a temporary one.
	ws            *WorkspaceService
	tempDir       string
	prevWorkspace string
	prevEnvPath   string
}

func NewConformanceServerService(ws *WorkspaceService) *ConformanceServerService {
	return &ConformanceServerService{ws: ws}
}

// ConformanceServerResult is returned to the frontend by Start.
type ConformanceServerResult struct {
	Port int `json:"port"`
	// WorkspacePath is set by OpenWorkspace: the temporary workspace that was opened.
	WorkspacePath string `json:"workspacePath,omitempty"`
	Error         string `json:"error,omitempty"`
}

func (c *ConformanceServerService) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.server != nil
}

func (c *ConformanceServerService) GetPort() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.port
}

func (c *ConformanceServerService) Start() ConformanceServerResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startLocked()
}

func (c *ConformanceServerService) startLocked() ConformanceServerResult {
	if c.server != nil {
		return ConformanceServerResult{Port: c.port}
	}
	configured := 0
	if c.portFn != nil {
		configured = c.portFn()
	}
	port, err := choosePort(configured, 8120, "the Conformance Server port")
	if err != nil {
		return ConformanceServerResult{Error: err.Error()}
	}
	srv := &http.Server{
		Addr:         fmt.Sprintf("localhost:%d", port),
		Handler:      newConformanceMux(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return ConformanceServerResult{Error: "could not bind port: " + err.Error()}
	}
	go srv.Serve(ln) //nolint:errcheck
	c.server, c.port = srv, port
	return ConformanceServerResult{Port: port}
}

func (c *ConformanceServerService) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		c.server.Shutdown(ctx) //nolint:errcheck
		c.server, c.port = nil, 0
	}
	c.restoreWorkspaceLocked()
}

// OpenWorkspace starts the server (if it is not running) and opens a temporary workspace with its
// spec and a local environment, so the Check Conformance page works in one click. Stop removes the
// workspace and reopens the previous one.
func (c *ConformanceServerService) OpenWorkspace() ConformanceServerResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tempDir != "" {
		return ConformanceServerResult{Port: c.port, WorkspacePath: c.tempDir}
	}
	if c.ws != nil && strings.HasPrefix(filepath.Base(c.ws.workspacePath), demoWorkspacePrefix) {
		return ConformanceServerResult{Error: "stop the Demo Server first: its workspace is open"}
	}
	res := c.startLocked()
	if res.Error != "" {
		return res
	}
	dir, err := os.MkdirTemp("", conformanceWorkspacePrefix+"*")
	if err != nil {
		return ConformanceServerResult{Port: res.Port, Error: "could not create the workspace: " + err.Error()}
	}
	claimTempWorkspace(dir)
	if err := writeConformanceWorkspace(dir, res.Port); err != nil {
		os.RemoveAll(dir)
		return ConformanceServerResult{Port: res.Port, Error: "could not create the workspace: " + err.Error()}
	}
	if c.ws != nil {
		c.prevWorkspace, c.prevEnvPath = c.ws.workspacePath, c.ws.activeEnvPath
		c.ws.openWorkspaceDirect(dir)
	}
	c.tempDir = dir
	return ConformanceServerResult{Port: res.Port, WorkspacePath: dir}
}

func (c *ConformanceServerService) restoreWorkspaceLocked() {
	if c.tempDir == "" {
		return
	}
	os.RemoveAll(c.tempDir)
	c.tempDir = ""
	if c.ws == nil {
		return
	}
	c.ws.workspacePath = c.prevWorkspace
	c.ws.activeEnvPath = c.prevEnvPath
	c.ws.activeEnv = nil
	if c.prevEnvPath != "" {
		if env, err := c.ws.loadEnvFile(c.prevEnvPath); err == nil {
			c.ws.activeEnv = env
		}
	}
	c.ws.syncTitle()
	c.prevWorkspace, c.prevEnvPath = "", ""
}

// writeConformanceWorkspace lays out the temporary workspace: the spec, and a local environment
// that sets the base URL and the values the generated requests use as placeholders.
func writeConformanceWorkspace(dir string, port int) error {
	for _, sub := range []string{"specs", "environments"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "specs", "conformance-openapi.yaml"), conformanceOpenAPIYAML, 0o644); err != nil {
		return err
	}
	env := strings.Replace(string(conformanceEnvironmentYAML), "name: conformance-server", "name: local", 1) +
		fmt.Sprintf("  - name: baseUrl\n    value: \"http://localhost:%d\"\n", port)
	if err := os.WriteFile(filepath.Join(dir, "environments", "local.yaml"), []byte(env), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "environments", ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return err
	}
	for name, content := range conformanceWorkspaceFiles {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// conformanceWorkspaceFiles are the sample requests and the scenario that make the Requests and
// Scenarios panels useful in the Conformance workspace. The scenario names the spec, so a run
// shows the spec conformance summary: its second and third steps hit the two operations whose
// responses break the spec.
var conformanceWorkspaceFiles = map[string]string{
	"requests/health.yaml": `schemaVersion: "1"
title: Health
method: GET
url: "{{baseUrl}}/health"
`,
	"requests/products/list-products.yaml": `schemaVersion: "1"
title: List products
method: GET
url: "{{baseUrl}}/products"
`,
	"requests/products/get-product.yaml": `schemaVersion: "1"
title: Get a product (price is sent as text)
method: GET
url: "{{baseUrl}}/products/{{id}}"
`,
	"requests/categories/list-categories.yaml": `schemaVersion: "1"
title: List categories (name is missing)
method: GET
url: "{{baseUrl}}/categories"
`,
	"scenarios/product-lifecycle.yaml": `schemaVersion: "1"
title: Product lifecycle
description: Reads products and categories from the Conformance Server and checks every response against its spec. Steps 2 and 3 pass their own assertions but break the spec; open Setup and its Result tab to see why.
specFile: ../specs/conformance-openapi.yaml

steps:
  - label: List products
    notes: Conforms to the spec.
    request:
      method: GET
      url: "{{baseUrl}}/products"
    assertions:
      - type: status
        operator: eq
        expected: 200
    failPolicy: continue

  - label: Get a product
    notes: The server sends price as text; the spec says number.
    request:
      method: GET
      url: "{{baseUrl}}/products/{{id}}"
    assertions:
      - type: status
        operator: eq
        expected: 200
    failPolicy: continue

  - label: List categories
    notes: The server leaves out the required name field.
    request:
      method: GET
      url: "{{baseUrl}}/categories"
    assertions:
      - type: status
        operator: eq
        expected: 200
    failPolicy: continue
`,
}

// OpenSpecInBrowser opens the Swagger UI page; a non-empty string is an error.
func (c *ConformanceServerService) OpenSpecInBrowser() string {
	port := c.GetPort()
	if port == 0 {
		return "conformance server is not running"
	}
	if err := openURL(fmt.Sprintf("http://localhost:%d/docs", port)); err != nil {
		return err.Error()
	}
	return ""
}
