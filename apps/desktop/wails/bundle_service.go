package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fuvito/api-workbench/packages/core/bundle"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxBundleBytes keeps a mistaken selection (a huge file) from being read into memory.
const maxBundleBytes = 4 << 20

// BundleService imports a bundle file — environments, requests and scenarios in one
// document — into the open workspace. The logic lives in packages/core/bundle so the
// CLI can reuse it.
type BundleService struct {
	ctx       context.Context
	workspace *WorkspaceService
}

func NewBundleService(ws *WorkspaceService) *BundleService {
	return &BundleService{ctx: context.Background(), workspace: ws}
}

func (s *BundleService) setContext(ctx context.Context) { s.ctx = ctx }

type BundleFileIO struct {
	Path  string `json:"path"`
	Text  string `json:"text"`
	Error string `json:"error,omitempty"`
}

type BundleIssueIO struct {
	Severity string `json:"severity"` // "error" | "warning"
	Where    string `json:"where"`
	Message  string `json:"message"`
}

type BundleEntryIO struct {
	Kind          string `json:"kind"` // environment | request | scenario
	Name          string `json:"name"`
	Path          string `json:"path"`          // where it will be written, relative to the workspace
	RequestedPath string `json:"requestedPath"` // where the bundle asked for it
	Action        string `json:"action"`        // create | overwrite | rename | skip
}

type BundlePreviewIO struct {
	Title      string          `json:"title"`
	SpecFile   string          `json:"specFile"` // the bundle's own default spec, for display
	Issues     []BundleIssueIO `json:"issues"`
	Entries    []BundleEntryIO `json:"entries"`
	HasErrors  bool            `json:"hasErrors"`
	ParseError string          `json:"parseError,omitempty"`
}

type BundleImportResultIO struct {
	Created     int      `json:"created"`
	Overwritten int      `json:"overwritten"`
	Renamed     int      `json:"renamed"`
	Skipped     int      `json:"skipped"`
	Written     []string `json:"written"`
	Error       string   `json:"error,omitempty"`
}

// PickBundleFile opens a file dialog and returns the chosen file's text. An empty Path
// with no Error means the user cancelled.
func (s *BundleService) PickBundleFile() BundleFileIO {
	path, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select bundle file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Bundle (*.yaml, *.yml, *.json, *.txt, *.md)", Pattern: "*.yaml;*.yml;*.json;*.txt;*.md"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return BundleFileIO{Error: err.Error()}
	}
	if path == "" {
		return BundleFileIO{}
	}
	return readBundleFile(path)
}

func readBundleFile(path string) BundleFileIO {
	info, err := os.Stat(path)
	if err != nil {
		return BundleFileIO{Path: path, Error: err.Error()}
	}
	if info.Size() > maxBundleBytes {
		return BundleFileIO{Path: path, Error: fmt.Sprintf("file is too large for a bundle (%d bytes, limit %d)", info.Size(), maxBundleBytes)}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return BundleFileIO{Path: path, Error: err.Error()}
	}
	return BundleFileIO{Path: path, Text: string(data)}
}

// PreviewBundle validates the text and reports the files an import would write, without
// writing anything. specFile, when non-empty, replaces the spec of every scenario.
func (s *BundleService) PreviewBundle(text, specFile, strategy string) BundlePreviewIO {
	out := BundlePreviewIO{Issues: []BundleIssueIO{}, Entries: []BundleEntryIO{}}
	if s.workspace.workspacePath == "" {
		out.ParseError = "no workspace open — open or create a workspace first"
		out.HasErrors = true
		return out
	}
	b, err := bundle.ParseText(text)
	if err != nil {
		out.ParseError = err.Error()
		out.HasErrors = true
		return out
	}
	out.Title = b.Title
	out.SpecFile = b.SpecFile

	plan, err := bundle.Prepare(b, s.workspace.workspacePath, bundle.Options{SpecFile: specFile}, bundle.Strategy(strategy))
	if err != nil {
		out.ParseError = err.Error()
		out.HasErrors = true
		return out
	}
	for _, i := range plan.Issues {
		out.Issues = append(out.Issues, BundleIssueIO{Severity: string(i.Severity), Where: i.Where, Message: i.Message})
	}
	for _, e := range plan.Entries {
		out.Entries = append(out.Entries, BundleEntryIO{
			Kind: string(e.Kind), Name: e.Name, Path: e.FinalPath, RequestedPath: e.RelPath, Action: string(e.Action),
		})
	}
	out.HasErrors = plan.HasErrors()
	return out
}

// ImportBundle validates the text and, if it is error-free, writes its files into the workspace.
func (s *BundleService) ImportBundle(text, specFile, strategy string) BundleImportResultIO {
	ws := s.workspace.workspacePath
	if ws == "" {
		return BundleImportResultIO{Error: "no workspace open — open or create a workspace first"}
	}
	b, err := bundle.ParseText(text)
	if err != nil {
		return BundleImportResultIO{Error: err.Error()}
	}
	plan, err := bundle.Prepare(b, ws, bundle.Options{SpecFile: specFile}, bundle.Strategy(strategy))
	if err != nil {
		return BundleImportResultIO{Error: err.Error()}
	}
	res, err := plan.Write(ws)
	if err != nil {
		out := BundleImportResultIO{Error: err.Error()}
		if res != nil {
			out.Written = res.Written
		}
		return out
	}
	return BundleImportResultIO{
		Created: res.Created, Overwritten: res.Overwritten, Renamed: res.Renamed, Skipped: res.Skipped,
		Written: res.Written,
	}
}
