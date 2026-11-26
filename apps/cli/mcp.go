package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/application"
	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/coverage"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/projectconfig"
	"github.com/fuvito/api-workbench/packages/core/reporting"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	var allowWrites bool
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP tool server over stdio",
		Long: `Start a Model Context Protocol (MCP) server over stdio. AI agents (Claude
Desktop, Cursor, etc.) can call clrsig tools — conform, run_scenario, diff,
and coverage — without leaving their workflow.

The conform tool is read-only: it sends only GET and HEAD. To let it send POST, PUT, PATCH and
DELETE, start the server with --allow-writes; the agent must then also ask for it on the call
(allowWrites: true). A project file with conform.readOnly: true (the one found from this folder
or --config, or the one beside the spec) still forbids writes; --no-config ignores project files.
The run_scenario tool is not limited by this: it sends whatever the scenario says.

Connect via stdio transport (newline-delimited JSON-RPC 2.0).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := &mcpServer{
				scanner: bufio.NewScanner(os.Stdin),
				enc:     json.NewEncoder(os.Stdout),
				writes:  mcpWritePolicy{allowed: allowWrites, cmd: cmd},
			}
			srv.scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
			return srv.serve()
		},
	}
	cmd.Flags().BoolVar(&allowWrites, "allow-writes", false, "Let the conform tool send POST, PUT, PATCH and DELETE when a call asks for it")
	return cmd
}

// mcpWritePolicy says whether a conform call may send requests that change data. The zero value
// allows none. The call's own argument is not enough: the agent writes that, and only the person
// who starts the server can write the flag.
type mcpWritePolicy struct {
	allowed bool
	cmd     *cobra.Command // for --config and --no-config; nil in a server built without them
}

// check returns nil when a call for this spec may send writes, and otherwise says why not.
func (p mcpWritePolicy) check(spec string) error {
	if !p.allowed {
		return fmt.Errorf("writes are not allowed: this server was started without --allow-writes, which only the person who configures it can add. Nothing was sent. Call again without allowWrites to check the GET and HEAD operations")
	}
	var found []*projectconfig.Config
	if p.cmd != nil {
		cfg, ignored, _, err := findProjectConfig(p.cmd)
		if err != nil {
			return fmt.Errorf("writes are not allowed: the project file cannot be read, so it may forbid them (%w). Nothing was sent", err)
		}
		if ignored {
			return nil // --no-config
		}
		if cfg != nil {
			found = append(found, cfg)
		}
	}
	// an agent app may start the server in any folder: the project of the spec counts too
	if !strings.HasPrefix(spec, "http://") && !strings.HasPrefix(spec, "https://") {
		abs, err := filepath.Abs(spec)
		if err != nil {
			return fmt.Errorf("writes are not allowed: %w. Nothing was sent", err)
		}
		home, _ := os.UserHomeDir()
		path, err := projectconfig.Discover(filepath.Dir(abs), home)
		if err == nil && path != "" {
			var cfg *projectconfig.Config
			if cfg, err = projectconfig.Load(path); err == nil {
				found = append(found, cfg)
			}
		}
		if err != nil {
			return fmt.Errorf("writes are not allowed: the project file of the spec cannot be read, so it may forbid them (%w). Nothing was sent", err)
		}
	}
	for _, cfg := range found {
		if cfg.Conform != nil && cfg.Conform.ReadOnly != nil && *cfg.Conform.ReadOnly {
			return fmt.Errorf("writes are not allowed: %s says conform.readOnly: true. Nothing was sent", cfg.Path())
		}
	}
	return nil
}

const mcpReadOnlyNote = "READ-ONLY: only GET and HEAD operations were sent. Operations shown as SKIP were NOT checked.\n\n"

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
	writes  mcpWritePolicy
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
		text, toolErr = mcpConform(params.Arguments, s.writes)
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
			"description": "Check a live server against an OpenAPI spec: send requests built from the spec and validate every response. Read-only unless the call sets allowWrites and the server was started with --allow-writes: only GET and HEAD are sent, and the other operations are reported as SKIP (not checked). Returns a conformance report.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec":               prop("Path or URL to OpenAPI spec (YAML or JSON)"),
					"baseUrl":            prop("Base URL of the API under test, e.g. http://localhost:8080"),
					"envFile":            prop("Path to environment YAML/JSON file for variable resolution (optional)"),
					"allowWrites":        map[string]interface{}{"type": "boolean", "description": "Also send POST, PUT, PATCH and DELETE, which change data on the server (default: false). Refused unless the person who configured the server started it with --allow-writes, and when the project file says conform.readOnly: true. Ask the user before using it."},
					"failOn":             map[string]interface{}{"type": "string", "enum": []string{"failure", "warning"}, "description": "failure (default) or warning: warnings are reported but only fail the run when this is warning"},
					"maxResponseTime":    map[string]interface{}{"type": "integer", "description": "Slowest acceptable response in milliseconds; a slower response is a violation (optional, 0 = off)"},
					"maxResponseTimeFor": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Different limits for path prefixes, each as /prefix=ms, for example /reports=2000 (optional; the longest prefix wins; 0 = no limit there)"},
				},
				"required": []string{"spec"},
			},
		},
		{
			"name":        "run_scenario",
			"description": "Execute a multi-step scenario file and return assertion results, extracted variables, and step-by-step pass/fail status. It sends whatever the scenario says, including requests that change data.",
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

func mcpConform(raw json.RawMessage, writes mcpWritePolicy) (string, error) {
	var args struct {
		Spec    string `json:"spec"`
		BaseURL string `json:"baseUrl"`
		EnvFile string `json:"envFile"`

		AllowWrites        bool     `json:"allowWrites"`
		FailOn             string   `json:"failOn"`
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
	failOn, err := checks.ParseFailOn(args.FailOn)
	if err != nil {
		return "", fmt.Errorf("failOn: %w", err)
	}
	if args.Spec == "" {
		return "", fmt.Errorf("spec is required")
	}
	if args.AllowWrites {
		if err := writes.check(args.Spec); err != nil {
			return "", err
		}
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

		ReadOnly:          !args.AllowWrites,
		MaxResponseTimeMs: args.MaxResponseTime,
		PathLimits:        pathLimits,
		FailOn:            failOn,
	})
	text := reporting.ConformText(out.Result, nil)
	if !args.AllowWrites {
		text = mcpReadOnlyNote + text
	}
	return text, nil
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
