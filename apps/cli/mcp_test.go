package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/coverage"
)

func rpcExchange(t *testing.T, lines ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	srv := &mcpServer{
		scanner: bufio.NewScanner(strings.NewReader(strings.Join(lines, "\n") + "\n")),
		enc:     json.NewEncoder(&out),
	}
	if err := srv.serve(); err != nil {
		t.Fatal(err)
	}
	var replies []map[string]any
	dec := json.NewDecoder(&out)
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		replies = append(replies, m)
	}
	return replies
}

func callTool(name string, args any) string {
	b, _ := json.Marshal(args)
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, b)
}

func toolText(t *testing.T, reply map[string]any) (string, bool) {
	t.Helper()
	res, ok := reply["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %v", reply)
	}
	content := res["content"].([]any)[0].(map[string]any)
	return content["text"].(string), res["isError"].(bool)
}

func TestMCP_Protocol(t *testing.T) {
	replies := rpcExchange(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		``,
		`not json`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"bogus"}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":"oops"}`,
		callTool("nope", map[string]any{}),
	)
	if len(replies) != 5 {
		t.Fatalf("notifications and malformed lines must not get replies; got %d replies", len(replies))
	}

	info := replies[0]["result"].(map[string]any)["serverInfo"].(map[string]any)
	if info["name"] != "clrsig" {
		t.Errorf("serverInfo: %v", info)
	}

	tools := replies[1]["result"].(map[string]any)["tools"].([]any)
	var names []string
	for _, tl := range tools {
		names = append(names, tl.(map[string]any)["name"].(string))
	}
	if strings.Join(names, ",") != "conform,run_scenario,diff,coverage" {
		t.Errorf("tools: %v", names)
	}

	if code := replies[2]["error"].(map[string]any)["code"].(float64); code != -32601 {
		t.Errorf("unknown method code %v", code)
	}
	if code := replies[3]["error"].(map[string]any)["code"].(float64); code != -32602 {
		t.Errorf("invalid params code %v", code)
	}
	if msg := replies[4]["error"].(map[string]any)["message"].(string); !strings.Contains(msg, "unknown tool") {
		t.Errorf("unknown tool message %q", msg)
	}
}

func TestMCP_ToolValidationErrors(t *testing.T) {
	tests := []struct {
		tool string
		args any
		want string
	}{
		{"conform", map[string]any{}, "spec is required"},
		{"conform", map[string]any{"spec": "/no/such/spec.yaml"}, ""},
		{"conform", map[string]any{"spec": 5}, "invalid arguments"},
		{"run_scenario", map[string]any{}, "scenario is required"},
		{"run_scenario", map[string]any{"scenario": "/no/such.yaml"}, "reading scenario"},
		{"diff", map[string]any{"baseSpec": "a"}, "required"},
		{"diff", map[string]any{"baseSpec": "/no/a.yaml", "revisionSpec": "/no/b.yaml"}, ""},
		{"coverage", map[string]any{}, "workspace is required"},
	}
	for _, tc := range tests {
		t.Run(tc.tool+"/"+tc.want, func(t *testing.T) {
			text, isErr := toolText(t, rpcExchange(t, callTool(tc.tool, tc.args))[0])
			if !isErr {
				t.Fatalf("expected tool error, got %q", text)
			}
			if !strings.Contains(text, tc.want) {
				t.Errorf("error %q should contain %q", text, tc.want)
			}
		})
	}
}

func TestMCP_RunScenario(t *testing.T) {
	srv := pingServer(t)
	scenarioPath := writeTempFile(t, "s.yaml", scenarioYAML(srv.URL))

	text, isErr := toolText(t, rpcExchange(t, callTool("run_scenario", map[string]any{"scenario": scenarioPath}))[0])
	if isErr || !strings.Contains(text, "get root") {
		t.Fatalf("unexpected result (err=%v): %s", isErr, text)
	}

	bad := writeTempFile(t, "bad.yaml", "steps: [")
	if text, isErr := toolText(t, rpcExchange(t, callTool("run_scenario", map[string]any{"scenario": bad}))[0]); !isErr || !strings.Contains(text, "parsing scenario") {
		t.Errorf("expected parse error, got %q", text)
	}

	missingEnv := map[string]any{"scenario": scenarioPath, "envFile": "/no/env.yaml"}
	if text, isErr := toolText(t, rpcExchange(t, callTool("run_scenario", missingEnv))[0]); !isErr || !strings.Contains(text, "reading env file") {
		t.Errorf("expected env error, got %q", text)
	}

	// An unusable specFile must fall back to a plain runner rather than failing the run.
	withBadSpec := map[string]any{"scenario": scenarioPath, "specFile": "/no/spec.yaml"}
	if text, isErr := toolText(t, rpcExchange(t, callTool("run_scenario", withBadSpec))[0]); isErr {
		t.Errorf("bad specFile should be ignored: %s", text)
	}

	spec := writeTempFile(t, "spec.yaml", minimalSpec)
	withSpec := map[string]any{"scenario": scenarioPath, "specFile": spec}
	if text, isErr := toolText(t, rpcExchange(t, callTool("run_scenario", withSpec))[0]); isErr {
		t.Errorf("valid specFile should work: %s", text)
	}
}

func TestMCP_Conform(t *testing.T) {
	srv := pingServer(t)
	spec := writeTempFile(t, "spec.yaml", minimalSpec)

	text, isErr := toolText(t, rpcExchange(t, callTool("conform", map[string]any{"spec": spec, "baseUrl": srv.URL}))[0])
	if isErr || !strings.Contains(text, "getPing") && !strings.Contains(text, "/ping") {
		t.Fatalf("unexpected conform output (err=%v): %s", isErr, text)
	}

	bad := writeTempFile(t, "bad.yaml", "openapi: [")
	if text, isErr := toolText(t, rpcExchange(t, callTool("conform", map[string]any{"spec": bad}))[0]); !isErr || !strings.Contains(text, "parsing spec") {
		t.Errorf("expected parse error, got %q", text)
	}
	if text, isErr := toolText(t, rpcExchange(t, callTool("conform", map[string]any{"spec": spec, "envFile": "/no/env.yaml"}))[0]); !isErr || !strings.Contains(text, "env file") {
		t.Errorf("expected env error, got %q", text)
	}
}

func TestMCP_Diff(t *testing.T) {
	base := writeTempFile(t, "base.yaml", minimalSpec)
	revised := writeTempFile(t, "rev.yaml", strings.Replace(minimalSpec, "/ping:", "/pong:", 1))
	same := writeTempFile(t, "same.yaml", minimalSpec)

	run := func(args map[string]any) string {
		text, isErr := toolText(t, rpcExchange(t, callTool("diff", args))[0])
		if isErr {
			t.Fatalf("diff failed: %s", text)
		}
		return text
	}

	if out := run(map[string]any{"baseSpec": base, "revisionSpec": same}); !strings.Contains(out, "No changes.") {
		t.Errorf("identical specs: %q", out)
	}
	if out := run(map[string]any{"baseSpec": base, "revisionSpec": same, "breakingOnly": true}); !strings.Contains(out, "No breaking changes.") {
		t.Errorf("identical specs breakingOnly: %q", out)
	}
	out := run(map[string]any{"baseSpec": base, "revisionSpec": revised})
	if !strings.Contains(out, "change(s)") || !strings.Contains(out, "breaking") {
		t.Errorf("removed path should be reported: %q", out)
	}
	if out := run(map[string]any{"baseSpec": base, "revisionSpec": revised, "breakingOnly": true}); !strings.Contains(out, "!") {
		t.Errorf("breaking marker missing: %q", out)
	}

	bad := writeTempFile(t, "bad.yaml", "openapi: [")
	for _, args := range []map[string]any{
		{"baseSpec": bad, "revisionSpec": base},
		{"baseSpec": base, "revisionSpec": bad},
	} {
		if text, isErr := toolText(t, rpcExchange(t, callTool("diff", args))[0]); !isErr || !strings.Contains(text, "parsing") {
			t.Errorf("expected parse error, got %q", text)
		}
	}
}

func TestMCP_Coverage(t *testing.T) {
	ws := t.TempDir()
	text, isErr := toolText(t, rpcExchange(t, callTool("coverage", map[string]any{"workspace": ws}))[0])
	if isErr || !strings.Contains(text, "No coverage data") {
		t.Fatalf("empty workspace: %q", text)
	}

	err := coverage.Save(ws, coverage.StoredCoverage{
		SpecFile: filepath.Join("specs", "api.yaml"), Exercised: []string{"GET /a"}, TotalOps: 2, TotalRuns: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	text, isErr = toolText(t, rpcExchange(t, callTool("coverage", map[string]any{"workspace": ws}))[0])
	if isErr || !strings.Contains(text, "Covered 1 (50%) · Not covered 1 (50%) · Passing 0 (0%) · Failing 0 (0%) · No result 1 (50%) · of 2 operations") || !strings.Contains(text, "3 run(s)") {
		t.Errorf("coverage output: %q", text)
	}
}

func TestMCP_LoadEnv(t *testing.T) {
	if env, err := mcpLoadEnv(""); env != nil || err != nil {
		t.Errorf("empty path should yield nil, nil; got %v %v", env, err)
	}
	good := writeTempFile(t, "env.yaml", "schemaVersion: \"1\"\nname: local\nvariables:\n  - name: host\n    value: x\n")
	if env, err := mcpLoadEnv(good); err != nil || env == nil || env.Name != "local" {
		t.Errorf("good env: %v %v", env, err)
	}
	bad := writeTempFile(t, "bad.yaml", "name: [")
	if _, err := mcpLoadEnv(bad); err == nil || !strings.Contains(err.Error(), "parsing env file") {
		t.Errorf("bad env: %v", err)
	}
	if _, err := mcpLoadEnv(filepath.Join(os.TempDir(), "definitely-missing-env.yaml")); err == nil {
		t.Error("missing env should fail")
	}
}
