package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// ConformanceServerService runs the deliberately imperfect Conformance Server.
// Unlike the demo it opens no workspace: it only listens on a local port.
type ConformanceServerService struct {
	mu     sync.Mutex
	server *http.Server
	port   int
	portFn func() int // configured port from Settings; nil or 0 = automatic
}

func NewConformanceServerService() *ConformanceServerService {
	return &ConformanceServerService{}
}

// ConformanceServerResult is returned to the frontend by Start.
type ConformanceServerResult struct {
	Port  int    `json:"port"`
	Error string `json:"error,omitempty"`
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
	if c.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c.server.Shutdown(ctx) //nolint:errcheck
	c.server, c.port = nil, 0
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
