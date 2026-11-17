package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/fuvito/api-workbench/packages/core/clibuild"
	"github.com/fuvito/api-workbench/packages/core/clispec"
)

// CliBuilderService backs the CLI Command Builder page. It describes the clrsig commands,
// lists the workspace files a command can use, and builds a checked, explained command line.
// It never runs anything; all logic lives in packages/core/clispec and clibuild.
type CliBuilderService struct {
	ws  *WorkspaceService
	ctx context.Context
}

func NewCliBuilderService(ws *WorkspaceService) *CliBuilderService {
	return &CliBuilderService{ws: ws}
}

func (s *CliBuilderService) setContext(ctx context.Context) { s.ctx = ctx }

type CliInputIO struct {
	Name       string   `json:"name"`
	Short      string   `json:"short"`
	Positional bool     `json:"positional"`
	Required   bool     `json:"required"`
	Kind       string   `json:"kind"`
	Role       string   `json:"role"`
	Accepts    []string `json:"accepts"`
	Enum       []string `json:"enum"`
	Default    string   `json:"default"`
	Label      string   `json:"label"`
	Help       string   `json:"help"`
	Example    string   `json:"example"`
	Group      string   `json:"group"`
}

type CliExitCodeIO struct {
	Code    int    `json:"code"`
	Meaning string `json:"meaning"`
	ForCI   string `json:"forCI"`
}

type CliCommandIO struct {
	Name      string          `json:"name"`
	Goal      string          `json:"goal"`
	Summary   string          `json:"summary"`
	When      string          `json:"when"`
	Inputs    []CliInputIO    `json:"inputs"`
	Outputs   []string        `json:"outputs"`
	ExitCodes []CliExitCodeIO `json:"exitCodes"`
}

type CliFileIO struct {
	Path  string `json:"path"`  // absolute
	Label string `json:"label"` // relative to the workspace, for display
}

type CliContextIO struct {
	Workspace    string      `json:"workspace"`
	RunFrom      string      `json:"runFrom"`
	Scenarios    []CliFileIO `json:"scenarios"`
	Environments []CliFileIO `json:"environments"`
	Specs        []CliFileIO `json:"specs"`
}

type CliBuildInput struct {
	Command string            `json:"command"`
	Values  map[string]string `json:"values"`
	RunFrom string            `json:"runFrom"`
}

type CliMessageIO struct {
	Field    string `json:"field"`
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

type CliNeedIO struct {
	Name   string `json:"name"`
	Secret bool   `json:"secret"`
	From   string `json:"from"`
}

type CliSuggestionIO struct {
	Field string `json:"field"`
	Value string `json:"value"`
	Why   string `json:"why"`
}

type CliPartIO struct {
	Token   string `json:"token"`
	Meaning string `json:"meaning"`
}

type CliRenderingIO struct {
	Shell     string `json:"shell"`
	Multiline string `json:"multiline"`
	Oneline   string `json:"oneline"`
}

type CliExplanationIO struct {
	Goal      string          `json:"goal"`
	Summary   string          `json:"summary"`
	When      string          `json:"when"`
	Parts     []CliPartIO     `json:"parts"`
	Outputs   []string        `json:"outputs"`
	ExitCodes []CliExitCodeIO `json:"exitCodes"`
}

type CliBuildOutput struct {
	Command     string            `json:"command"`
	Valid       bool              `json:"valid"`
	Argv        []string          `json:"argv"`
	Renderings  []CliRenderingIO  `json:"renderings"`
	Messages    []CliMessageIO    `json:"messages"`
	Needs       []CliNeedIO       `json:"needs"`
	Suggestions []CliSuggestionIO `json:"suggestions"`
	Explanation CliExplanationIO  `json:"explanation"`
}

// GetCliCatalog returns the commands the builder offers, without inputs that are not
// exposed yet.
func (s *CliBuilderService) GetCliCatalog() []CliCommandIO {
	var out []CliCommandIO
	for _, c := range clispec.All() {
		io := CliCommandIO{
			Name: c.Name, Goal: c.Goal, Summary: c.Summary, When: c.When,
			Inputs: []CliInputIO{}, Outputs: c.Outputs, ExitCodes: exitCodesIO(c.ExitCodes),
		}
		for _, in := range c.Inputs {
			if in.Deferred {
				continue
			}
			io.Inputs = append(io.Inputs, CliInputIO{
				Name: in.Name, Short: in.Short, Positional: in.Positional, Required: in.Required,
				Kind: string(in.Kind), Role: string(in.Role), Accepts: nonNil(in.Accepts), Enum: nonNil(in.Enum),
				Default: in.Default, Label: in.Label, Help: in.Help, Example: in.Example, Group: string(in.Group),
			})
		}
		out = append(out, io)
	}
	return out
}

// GetCliContext lists the workspace files that can feed a command, and the folder commands
// are best run from.
func (s *CliBuilderService) GetCliContext() CliContextIO {
	ws := s.ws.GetWorkspacePath()
	ctx := CliContextIO{Workspace: ws, Scenarios: []CliFileIO{}, Environments: []CliFileIO{}, Specs: []CliFileIO{}}
	if ws == "" {
		return ctx
	}
	ctx.RunFrom = clibuild.DefaultRunFrom(ws)
	ctx.Scenarios = listFiles(ws, "scenarios")
	ctx.Environments = listFiles(ws, "environments")
	ctx.Specs = findSpecs(ws)
	return ctx
}

// BuildCliCommand validates the choices and renders the command for every shell.
func (s *CliBuilderService) BuildCliCommand(in CliBuildInput) CliBuildOutput {
	runFrom := in.RunFrom
	if runFrom == "" {
		if ws := s.ws.GetWorkspacePath(); ws != "" {
			runFrom = clibuild.DefaultRunFrom(ws)
		}
	}
	res := clibuild.Build(clibuild.Request{Command: in.Command, Values: in.Values}, runFrom)

	out := CliBuildOutput{
		Command: res.Command, Valid: res.Valid, Argv: nonNil(res.Argv),
		Renderings: []CliRenderingIO{}, Messages: []CliMessageIO{}, Needs: []CliNeedIO{}, Suggestions: []CliSuggestionIO{},
		Explanation: CliExplanationIO{
			Goal: res.Explanation.Goal, Summary: res.Explanation.Summary, When: res.Explanation.When,
			Parts: []CliPartIO{}, Outputs: nonNil(res.Explanation.Outputs), ExitCodes: exitCodesIO(res.Explanation.ExitCodes),
		},
	}
	for _, r := range res.Renderings {
		out.Renderings = append(out.Renderings, CliRenderingIO{Shell: string(r.Shell), Multiline: r.Multiline, Oneline: r.Oneline})
	}
	for _, m := range res.Messages {
		out.Messages = append(out.Messages, CliMessageIO{Field: m.Field, Severity: string(m.Severity), Text: m.Text})
	}
	for _, n := range res.Needs {
		out.Needs = append(out.Needs, CliNeedIO{Name: n.Name, Secret: n.Secret, From: n.From})
	}
	for _, sg := range res.Suggestions {
		out.Suggestions = append(out.Suggestions, CliSuggestionIO{Field: sg.Field, Value: sg.Value, Why: sg.Why})
	}
	for _, p := range res.Explanation.Parts {
		out.Explanation.Parts = append(out.Explanation.Parts, CliPartIO{Token: p.Token, Meaning: p.Meaning})
	}
	return out
}

// PickCliFile opens a file picker for a file the command should use; it returns "" if cancelled.
func (s *CliBuilderService) PickCliFile(title string) string {
	if s.ctx == nil {
		return ""
	}
	path, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML / JSON (*.yaml, *.yml, *.json)", Pattern: "*.yaml;*.yml;*.json"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

// PickCliFolder opens a folder picker; it returns "" if cancelled.
func (s *CliBuilderService) PickCliFolder(title string) string {
	if s.ctx == nil {
		return ""
	}
	path, err := runtime.OpenDirectoryDialog(s.ctx, runtime.OpenDialogOptions{Title: title})
	if err != nil {
		return ""
	}
	return path
}

func exitCodesIO(in []clispec.ExitCode) []CliExitCodeIO {
	out := make([]CliExitCodeIO, 0, len(in))
	for _, e := range in {
		out = append(out, CliExitCodeIO{Code: e.Code, Meaning: e.Meaning, ForCI: e.ForCI})
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

var dataExts = map[string]bool{".yaml": true, ".yml": true, ".json": true}

// listFiles returns the YAML/JSON files under workspace/sub (recursively), sorted.
func listFiles(workspace, sub string) []CliFileIO {
	out := []CliFileIO{}
	root := filepath.Join(workspace, sub)
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !dataExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		out = append(out, CliFileIO{Path: path, Label: relLabel(workspace, path)})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "dist": true, "build": true, "vendor": true}

// findSpecs finds OpenAPI / Swagger documents in the workspace by looking at the start of
// each YAML/JSON file. It stays shallow and bounded so a large repository does not stall the UI.
func findSpecs(workspace string) []CliFileIO {
	const maxDepth, maxFiles, maxSize, sniff = 4, 40, 4 << 20, 4096
	out := []CliFileIO{}
	_ = filepath.WalkDir(workspace, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != workspace && (skipDirs[d.Name()] || strings.Count(relLabel(workspace, path), "/") >= maxDepth) {
				return fs.SkipDir
			}
			return nil
		}
		if len(out) >= maxFiles || !dataExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		if info, err := d.Info(); err != nil || info.Size() > maxSize {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		buf := make([]byte, sniff)
		n, _ := f.Read(buf)
		head := string(buf[:n])
		if strings.Contains(head, "openapi:") || strings.Contains(head, `"openapi"`) ||
			strings.Contains(head, "swagger:") || strings.Contains(head, `"swagger"`) {
			out = append(out, CliFileIO{Path: path, Label: relLabel(workspace, path)})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

func relLabel(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
