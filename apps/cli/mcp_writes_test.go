package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// The conform tool of the MCP server is used by an AI agent: it sends nothing that changes data
// unless the person who started the server allowed it and the call asks for it.

const writingSpec = `openapi: "3.0.3"
info: {title: m, version: "1"}
paths:
  /a:
    get: {responses: {"200": {description: ok}}}
    head: {responses: {"200": {description: ok}}}
    post: {responses: {"200": {description: ok}}}
    put: {responses: {"200": {description: ok}}}
    patch: {responses: {"200": {description: ok}}}
    delete: {responses: {"200": {description: ok}}}
`

type recorder struct {
	mu   sync.Mutex
	sent []string
}

func (r *recorder) methods() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := slices.Clone(r.sent)
	slices.Sort(out)
	return out
}

// callConform starts `clrsig mcp` with serverArgs in the folder dir, calls the conform tool and
// returns the reply, whether it was an error, and the methods the API received.
func callConform(t *testing.T, serverArgs []string, dir string, arguments map[string]any) (string, bool, []string) {
	t.Helper()
	rec := &recorder{}
	api := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.sent = append(rec.sent, r.Method)
		rec.mu.Unlock()
	}))
	t.Cleanup(api.Close)
	arguments["baseUrl"] = api.URL
	call, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "conform", "arguments": arguments}})
	if err != nil {
		t.Fatal(err)
	}

	in, out := filepath.Join(t.TempDir(), "in"), filepath.Join(t.TempDir(), "out")
	if err := os.WriteFile(in, append(call, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	stdin, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = stdin, stdout
	t.Chdir(dir)
	err = runQuietly(t, append([]string{"mcp"}, serverArgs...)...)
	os.Stdin, os.Stdout = oldIn, oldOut
	_ = stdin.Close()
	_ = stdout.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	lines.Buffer(make([]byte, 1<<20), 1<<20)
	if !lines.Scan() {
		t.Fatal("the server answered nothing")
	}
	var reply struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(lines.Bytes(), &reply); err != nil || len(reply.Result.Content) == 0 {
		t.Fatalf("%v: %s", err, lines.Text())
	}
	return reply.Result.Content[0].Text, reply.Result.IsError, rec.methods()
}

func folder(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const readOnlyProject = "schemaVersion: \"1\"\nconform:\n  readOnly: true\n"

func TestMCPConformSendsWritesOnlyWhenBothAllowIt(t *testing.T) {
	reads := []string{"GET", "HEAD"}
	all := []string{"DELETE", "GET", "HEAD", "PATCH", "POST", "PUT"}
	plain := map[string]string{"spec.yaml": writingSpec}
	readOnly := map[string]string{"spec.yaml": writingSpec, "clrsig.yaml": readOnlyProject}
	writable := map[string]string{"spec.yaml": writingSpec, "clrsig.yaml": "schemaVersion: \"1\"\nconform:\n  readOnly: false\n"}

	for name, c := range map[string]struct {
		server  []string
		files   map[string]string
		ask     any // the allowWrites argument; nil leaves it out
		sent    []string
		refused string
	}{
		"by default":                       {nil, plain, nil, reads, ""},
		"the flag alone":                   {[]string{"--allow-writes"}, plain, nil, reads, ""},
		"asked for false":                  {[]string{"--allow-writes"}, plain, false, reads, ""},
		"the agent alone cannot":           {nil, plain, true, nil, "--allow-writes"},
		"both":                             {[]string{"--allow-writes"}, plain, true, all, ""},
		"a project that allows writes":     {[]string{"--allow-writes"}, writable, true, all, ""},
		"a read-only project forbids":      {[]string{"--allow-writes"}, readOnly, true, nil, "conform.readOnly: true"},
		"a writable project is not enough": {nil, writable, true, nil, "--allow-writes"},
		"--no-config ignores the project":  {[]string{"--allow-writes", "--no-config"}, readOnly, true, all, ""},
		"a project that cannot be read":    {[]string{"--allow-writes"}, map[string]string{"spec.yaml": writingSpec, "clrsig.yaml": "conform: ["}, true, nil, "cannot be read"},
	} {
		t.Run(name, func(t *testing.T) {
			args := map[string]any{"spec": "spec.yaml"}
			if c.ask != nil {
				args["allowWrites"] = c.ask
			}
			text, isError, sent := callConform(t, c.server, folder(t, c.files), args)
			if !slices.Equal(sent, c.sent) {
				t.Errorf("the API received %v, want %v\n%s", sent, c.sent, text)
			}
			if isError != (c.refused != "") || !strings.Contains(text, c.refused) {
				t.Errorf("error=%v, want it to say %q:\n%s", isError, c.refused, text)
			}
			if c.refused != "" {
				return
			}
			if note := strings.HasPrefix(text, "READ-ONLY"); note != (len(c.sent) == len(reads)) {
				t.Errorf("the reply must say it was read-only exactly when it was:\n%s", text)
			}
		})
	}
}

// An agent app may start the server in any folder: the read-only setting beside the spec counts.
func TestMCPConformObeysTheProjectBesideTheSpec(t *testing.T) {
	project := folder(t, map[string]string{"spec.yaml": writingSpec, "clrsig.yaml": readOnlyProject})
	text, isError, sent := callConform(t, []string{"--allow-writes"}, t.TempDir(), map[string]any{"spec": filepath.Join(project, "spec.yaml"), "allowWrites": true})
	if !isError || len(sent) != 0 || !strings.Contains(text, "conform.readOnly: true") {
		t.Errorf("error=%v sent=%v\n%s", isError, sent, text)
	}
}

func TestMCPToolListOffersAllowWrites(t *testing.T) {
	b, err := json.Marshal(mcpToolList())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"allowWrites"`, "Read-only unless", "--allow-writes", "including requests that change data"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the tool list does not say %q", want)
		}
	}
}
