package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP tool server over stdio",
		Long: `Start a Model Context Protocol (MCP) server over stdio. AI agents (Claude
Desktop, Cursor, etc.) can call clrsig tools — conform, run_scenario, diff,
and coverage — without leaving their workflow.

Connect via stdio transport (newline-delimited JSON-RPC 2.0).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := &mcpServer{
				scanner: bufio.NewScanner(os.Stdin),
				enc:     json.NewEncoder(os.Stdout),
			}
			srv.scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
			return srv.serve()
		},
	}
}

// --- JSON-RPC types ---

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- MCP server ---

type mcpServer struct {
	scanner *bufio.Scanner
	enc     *json.Encoder
}

func (s *mcpServer) serve() error {
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue // skip malformed messages
		}
		// Notifications (no id) expect no response.
		if req.ID == nil {
			continue
		}
		_ = s.enc.Encode(s.dispatch(req))
	}
	return s.scanner.Err()
}

func (s *mcpServer) dispatch(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]interface{}{"name": "clrsig", "version": version},
			},
		}

	case "tools/list":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"tools": mcpToolList()},
		}

	case "tools/call":
		return s.toolCall(req)

	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "method not found: " + req.Method},
		}
	}
}

func (s *mcpServer) toolCall(req rpcRequest) rpcResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID,
			Error: &rpcError{Code: -32602, Message: "invalid params"}}
	}

	var text string
	var toolErr error

	switch params.Name {
	case "conform":
		text, toolErr = mcpConform(params.Arguments)
	case "run_scenario":
		text, toolErr = mcpRunScenario(params.Arguments)
	case "diff":
		text, toolErr = mcpDiff(params.Arguments)
	case "coverage":
		text, toolErr = mcpCoverage(params.Arguments)
	default:
		return rpcResponse{JSONRPC: "2.0", ID: req.ID,
			Error: &rpcError{Code: -32602, Message: "unknown tool: " + params.Name}}
	}

	isError := toolErr != nil
	if isError {
		text = toolErr.Error()
	}
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": text}},
			"isError": isError,
		},
	}
}

// --- Tool definitions ---

func mcpToolList() []map[string]interface{} {
	prop := func(desc string) map[string]interface{} {
		return map[string]interface{}{"type": "string", "description": desc}
	}
	return []map[string]interface{}{
		{
			"name":        "conform",
			"description": "Execute all operations from an OpenAPI spec against a live server and validate every response against the declared schema. Returns a full conformance report.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec":               prop("Path or URL to OpenAPI spec (YAML or JSON)"),
					"baseUrl":            prop("Base URL of the API under test, e.g. http://localhost:8080"),
					"envFile":            prop("Path to environment YAML/JSON file for variable resolution (optional)"),
					"maxResponseTime":    map[string]interface{}{"type": "integer", "description": "Slowest acceptable response in milliseconds; a slower response is a violation (optional, 0 = off)"},
					"maxResponseTimeFor": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Different limits for path prefixes, each as /prefix=ms, for example /reports=2000 (optional; the longest prefix wins; 0 = no limit there)"},
				},
				"required": []string{"spec"},
			},
		},
		{
			"name":        "run_scenario",
			"description": "Execute a multi-step scenario file and return assertion results, extracted variables, and step-by-step pass/fail status.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"scenario": prop("Path to scenario YAML file"),
					"envFile":  prop("Path to environment file for variable resolution (optional)"),
					"specFile": prop("Path or URL to OpenAPI spec for schema validation (optional)"),
				},
				"required": []string{"scenario"},
			},
		},
		{
			"name":        "diff",
			"description": "Compare two OpenAPI specs and classify every change as breaking (ERR), potentially breaking (WARN), or informational (INFO).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"baseSpec":     prop("Path to the base (older) spec"),
					"revisionSpec": prop("Path to the revision (newer) spec"),
					"breakingOnly": map[string]interface{}{"type": "boolean", "description": "Report only breaking changes (default: false)"},
				},
				"required": []string{"baseSpec", "revisionSpec"},
			},
		},
		{
			"name":        "coverage",
			"description": "List accumulated spec coverage for all specs in a workspace directory. Shows how many operations have been exercised across all scenario runs.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workspace": prop("Path to workspace directory"),
				},
				"required": []string{"workspace"},
			},
		},
	}
}

// --- Tool handlers ---

func mcpConform(raw json.RawMessage) (string, error) {
	var args struct {
		Spec    string `json:"spec"`
		BaseURL string `json:"baseUrl"`
		EnvFile string `json:"envFile"`

		MaxResponseTime    int      `json:"maxResponseTime"`
		MaxResponseTimeFor []string `json:"maxResponseTimeFor"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.MaxResponseTime < 0 {
		return "", fmt.Errorf("maxResponseTime must not be negative")
	}
	pathLimits, err := conformance.ParsePathLimits(args.MaxResponseTimeFor)
	if err != nil {
		return "", err
	}
	if args.Spec == "" {
		return "", fmt.Errorf("spec is required")
	}

	data, specFmt, err := fetchSpec(args.Spec)
	if err != nil {
		return "", err
	}
	spec, err := openapi.ParseBytes(data, specFmt)
	if err != nil {
		return "", fmt.Errorf("parsing spec: %w", err)
	}

	env, err := mcpLoadEnv(args.EnvFile)
	if err != nil {
		return "", err
	}

	out := application.RunConformance(context.Background(), application.ConformInput{
		Exec:    newExecutor(),
		Spec:    spec,
		BaseURL: args.BaseURL,
		Env:     env,

		MaxResponseTimeMs: args.MaxResponseTime,
		PathLimits:        pathLimits,
	})
	return reporting.ConformText(out.Result, nil), nil
}

func mcpRunScenario(raw json.RawMessage) (string, error) {
	var args struct {
		Scenario string `json:"scenario"`
		EnvFile  string `json:"envFile"`
		SpecFile string `json:"specFile"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Scenario == "" {
		return "", fmt.Errorf("scenario is required")
	}

	data, err := os.ReadFile(args.Scenario)
	if err != nil {
		return "", fmt.Errorf("reading scenario: %w", err)
	}
	sc, err := serialization.ParseScenarioFile(data, args.Scenario)
	if err != nil {
		return "", fmt.Errorf("parsing scenario: %w", err)
	}

	env, err := mcpLoadEnv(args.EnvFile)
	if err != nil {
		return "", err
	}

	exec := newExecutor()
	var runner *scenario.Runner
	if args.SpecFile != "" {
		specData, specFmt, fetchErr := fetchSpec(args.SpecFile)
		if fetchErr == nil {
			if spec, parseErr := openapi.ParseBytes(specData, specFmt); parseErr == nil {
				runner = scenario.NewWithSpec(exec, spec)
			}
		}
	}
	if runner == nil {
		runner = scenario.New(exec)
	}

	result := runner.Run(context.Background(), sc, env)
	return reporting.Text(result), nil
}

func mcpDiff(raw json.RawMessage) (string, error) {
	var args struct {
		BaseSpec     string `json:"baseSpec"`
		RevisionSpec string `json:"revisionSpec"`
		BreakingOnly bool   `json:"breakingOnly"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.BaseSpec == "" || args.RevisionSpec == "" {
		return "", fmt.Errorf("baseSpec and revisionSpec are required")
	}

	baseData, baseFmt, err := fetchSpec(args.BaseSpec)
	if err != nil {
		return "", err
	}
	revData, revFmt, err := fetchSpec(args.RevisionSpec)
	if err != nil {
		return "", err
	}
	baseSpec, err := openapi.ParseBytes(baseData, baseFmt)
	if err != nil {
		return "", fmt.Errorf("parsing base spec: %w", err)
	}
	revSpec, err := openapi.ParseBytes(revData, revFmt)
	if err != nil {
		return "", fmt.Errorf("parsing revision spec: %w", err)
	}
	result, err := openapi.Diff(baseSpec, revSpec)
	if err != nil {
		return "", fmt.Errorf("diffing specs: %w", err)
	}

	changes := result.Changes
	if args.BreakingOnly {
		filtered := changes[:0]
		for _, c := range changes {
			if c.Breaking {
				filtered = append(filtered, c)
			}
		}
		changes = filtered
	}

	var sb strings.Builder
	if len(changes) == 0 {
		if args.BreakingOnly {
			fmt.Fprintln(&sb, "No breaking changes.")
		} else {
			fmt.Fprintln(&sb, "No changes.")
		}
	} else {
		for _, c := range changes {
			mark := " "
			if c.Breaking {
				mark = "!"
			}
			loc := ""
			if c.Method != "" && c.Path != "" {
				loc = fmt.Sprintf("  %s %s", c.Method, c.Path)
			}
			fmt.Fprintf(&sb, "  [%s] %s %s%s\n", c.Level, mark, c.Text, loc)
		}
		breaking := 0
		for _, c := range changes {
			if c.Breaking {
				breaking++
			}
		}
		fmt.Fprintln(&sb)
		if result.HasBreaking {
			fmt.Fprintf(&sb, "%d change(s)  ·  %d breaking\n", len(changes), breaking)
		} else {
			fmt.Fprintf(&sb, "%d change(s)  ·  no breaking changes\n", len(changes))
		}
	}
	return sb.String(), nil
}

func mcpCoverage(raw json.RawMessage) (string, error) {
	var args struct {
		Workspace string `json:"workspace"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Workspace == "" {
		return "", fmt.Errorf("workspace is required")
	}

	reports := coverage.ListAll(args.Workspace)
	if len(reports) == 0 {
		return "No coverage data found in workspace.", nil
	}

	var sb strings.Builder
	for _, r := range reports {
		fmt.Fprintf(&sb, "%s\n  %s\n  scenario runs: %d run(s) · %d passed · %d failed\n\n",
			r.SpecFile, reporting.OutcomesLine(r.Outcomes()), r.TotalRuns, r.PassedRuns, r.FailedRuns)
	}
	return sb.String(), nil
}

func mcpLoadEnv(envFile string) (*domain.Environment, error) {
	if envFile == "" {
		return nil, nil
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	env, err := serialization.ParseEnvironmentFile(data, envFile)
	if err != nil {
		return nil, fmt.Errorf("parsing env file: %w", err)
	}
	return env, nil
}
