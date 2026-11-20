package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

// ---- Config persistence ----

type workspaceConfig struct {
	WorkspacePath    string   `json:"workspacePath"`
	ActiveEnvPath    string   `json:"activeEnvPath"`
	RecentWorkspaces []string `json:"recentWorkspaces,omitempty"`
	WindowWidth      int      `json:"windowWidth,omitempty"`
	WindowHeight     int      `json:"windowHeight,omitempty"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".clrsig")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadConfig() workspaceConfig {
	p, err := configPath()
	if err != nil {
		return workspaceConfig{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return workspaceConfig{}
	}
	var cfg workspaceConfig
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg workspaceConfig) {
	p, err := configPath()
	if err != nil {
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWrite(p, data)
}

// ---- Wails-serializable I/O types ----

type FileNodeIO struct {
	Name     string       `json:"name"`
	Path     string       `json:"path"`
	IsDir    bool         `json:"isDir"`
	Method   string       `json:"method,omitempty"`
	Children []FileNodeIO `json:"children,omitempty"`
}

type EnvironmentInfoIO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type EnvironmentVariableIO struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type EnvironmentIO struct {
	Path      string                  `json:"path"`
	Name      string                  `json:"name"`
	Variables []EnvironmentVariableIO `json:"variables"`
}

type RequestFileIO struct {
	SchemaVersion string         `json:"schemaVersion"`
	Title         string         `json:"title"`
	Method        string         `json:"method"`
	URL           string         `json:"url"`
	Headers       []HeaderIO     `json:"headers,omitempty"`
	QueryParams   []QueryParamIO `json:"queryParams,omitempty"`
	Body          *BodyIO        `json:"body,omitempty"`
	Auth          *AuthIO        `json:"auth,omitempty"`
	TimeoutMs     int64          `json:"timeoutMs,omitempty"`
	// NoFollowRedirects is the file's followRedirects: false.
	NoFollowRedirects bool `json:"noFollowRedirects,omitempty"`
	// FilePath is the on-disk path of the loaded file; not persisted to YAML.
	FilePath string `json:"filePath,omitempty"`
	// TargetDir overrides the save directory for new (unsaved) requests.
	TargetDir string `json:"targetDir,omitempty"`
}

// ---- WorkspaceService ----

type WorkspaceService struct {
	ctx              context.Context
	wailsReady       bool // true once the real Wails ctx is available
	workspacePath    string
	activeEnvPath    string
	activeEnv        *domain.Environment
	recentWorkspaces []string
}

func NewWorkspaceService() *WorkspaceService {
	cfg := loadConfig()
	cleanupStaleTempWorkspaces()
	// The app always starts on the landing page: the last workspace is only
	// offered in the recent list, never reopened.
	ws := &WorkspaceService{
		ctx:              context.Background(),
		recentWorkspaces: prunedRecentWorkspaces(cfg.RecentWorkspaces),
	}
	ws.persistConfig()
	return ws
}

func isTempWorkspace(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, demoWorkspacePrefix) || strings.HasPrefix(base, conformanceWorkspacePrefix)
}

// prunedRecentWorkspaces drops temporary demo/conformance workspaces and
// folders that no longer exist.
func prunedRecentWorkspaces(in []string) []string {
	var out []string
	for _, p := range in {
		if p == "" || isTempWorkspace(p) {
			continue
		}
		if fi, err := os.Stat(p); err != nil || !fi.IsDir() {
			continue
		}
		out = append(out, p)
	}
	return out
}

// cleanupStaleTempWorkspaces removes demo and conformance temp workspaces left
// behind by a session that ended without stopping its server.
func cleanupStaleTempWorkspaces() {
	base := os.TempDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && isTempWorkspace(e.Name()) {
			os.RemoveAll(filepath.Join(base, e.Name())) //nolint:errcheck
		}
	}
}

func (w *WorkspaceService) setContext(ctx context.Context) {
	w.ctx = ctx
	w.wailsReady = true
}

// syncTitle updates the native window title to reflect the current workspace.
func (w *WorkspaceService) syncTitle() {
	if !w.wailsReady {
		return
	}
	if w.workspacePath == "" {
		runtime.WindowSetTitle(w.ctx, "Clear Signal")
	} else {
		runtime.WindowSetTitle(w.ctx, "Clear Signal — "+filepath.Base(w.workspacePath))
	}
}

func (w *WorkspaceService) ActiveEnvironment() *domain.Environment {
	return w.activeEnv
}

// openWorkspaceDirect sets the workspace path directly without a dialog.
// It also auto-selects the environments/local.yaml if present.
// Does not persist to config (temp demo workspace should not pollute saved state).
func (w *WorkspaceService) openWorkspaceDirect(path string) {
	w.workspacePath = path
	w.syncTitle()
	envPath := filepath.Join(path, "environments", "local.yaml")
	if _, err := os.Stat(envPath); err == nil {
		w.activeEnvPath = envPath
		if env, err := w.loadEnvFile(envPath); err == nil {
			w.activeEnv = env
		}
	}
}

// ---- Wails-bound methods ----

// OpenWorkspace opens a directory picker dialog and sets the workspace.
func (w *WorkspaceService) OpenWorkspace() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(w.ctx, runtime.OpenDialogOptions{
		Title: "Open Workspace Folder",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return w.workspacePath, nil // user cancelled
	}
	w.workspacePath = dir
	w.addRecentWorkspace(dir)
	w.persistConfig()
	w.syncTitle()
	return dir, nil
}

// GetWorkspacePath returns the current workspace directory path.
func (w *WorkspaceService) GetWorkspacePath() string {
	return w.workspacePath
}

// CloseWorkspace clears the current workspace and persists the change.
func (w *WorkspaceService) CloseWorkspace() {
	w.workspacePath = ""
	w.activeEnvPath = ""
	w.activeEnv = nil
	w.persistConfig()
	w.syncTitle()
}

// ListRequests returns the workspace file tree limited to request files (.yaml/.yml/.json).
func (w *WorkspaceService) ListRequests() []FileNodeIO {
	if w.workspacePath == "" {
		return nil
	}
	requestsDir := filepath.Join(w.workspacePath, "requests")
	if _, err := os.Stat(requestsDir); os.IsNotExist(err) {
		return nil
	}
	nodes, err := buildFileTree(requestsDir)
	if err != nil {
		log.Printf("listing requests: %v", err)
		return nil
	}
	return nodes
}

// ListEnvironments returns all environment files found in workspacePath/environments/.
func (w *WorkspaceService) ListEnvironments() []EnvironmentInfoIO {
	if w.workspacePath == "" {
		return nil
	}
	envDir := filepath.Join(w.workspacePath, "environments")
	entries, err := os.ReadDir(envDir)
	if err != nil {
		return nil
	}

	var result []EnvironmentInfoIO
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isDataFile(name) {
			continue
		}
		path := filepath.Join(envDir, name)
		displayName := strings.TrimSuffix(name, filepath.Ext(name))
		// Try to read the declared name from the file
		if data, err := os.ReadFile(path); err == nil {
			if env, err := serialization.ParseEnvironmentFile(data, name); err == nil && env.Name != "" {
				displayName = env.Name
			}
		}
		result = append(result, EnvironmentInfoIO{Name: displayName, Path: path})
	}
	return result
}

// SelectEnvironment sets the active environment by file path.
func (w *WorkspaceService) SelectEnvironment(path string) error {
	if path == "" {
		w.activeEnv = nil
		w.activeEnvPath = ""
		w.persistConfig()
		return nil
	}
	env, err := w.loadEnvFile(path)
	if err != nil {
		return err
	}
	w.activeEnv = env
	w.activeEnvPath = path
	w.persistConfig()
	return nil
}

// GetActiveEnvironment returns info about the currently selected environment (or nil).
func (w *WorkspaceService) GetActiveEnvironment() *EnvironmentInfoIO {
	if w.activeEnv == nil {
		return nil
	}
	return &EnvironmentInfoIO{Name: w.activeEnv.Name, Path: w.activeEnvPath}
}

// LoadRequest reads a request file and returns a RequestFileIO the UI can populate into the editor.
func (w *WorkspaceService) LoadRequest(path string) (*RequestFileIO, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	req, err := serialization.ParseRequestFile(data, filepath.Base(path))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	out := domainToRequestFileIO(req)
	out.FilePath = path
	return out, nil
}

// SaveRequest writes the request to disk without a dialog.
// The target path is derived from req.Title: {dir}/{slug}.yaml.
// If req.FilePath is set (file already exists), the directory is preserved and
// the file is renamed when the name slug changes. Returns the saved path.
func (w *WorkspaceService) SaveRequest(req RequestFileIO) (string, error) {
	name := strings.TrimSpace(req.Title)
	if name == "" {
		name = "request"
	}
	slug := sanitizeDirName(name)
	if slug == "" {
		slug = "request"
	}

	var targetDir string
	if req.FilePath != "" {
		targetDir = filepath.Dir(req.FilePath)
	} else if req.TargetDir != "" {
		targetDir = req.TargetDir
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", err
		}
	} else {
		if w.workspacePath == "" {
			return "", fmt.Errorf("no workspace open — open or create a workspace first")
		}
		targetDir = filepath.Join(w.workspacePath, "requests")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", err
		}
	}

	targetPath := filepath.Join(targetDir, slug+".yaml")

	// Conflict: target exists and is a different file from the one we loaded.
	// Use case-insensitive comparison to handle case-only renames on Windows/macOS.
	if !strings.EqualFold(targetPath, req.FilePath) {
		if _, err := os.Stat(targetPath); err == nil {
			return "", fmt.Errorf("a request named %q already exists in this folder — rename it first", name)
		}
	}

	domReq := requestFileIOToDomain(req)
	data, err := serialization.WriteRequest(domReq, "yaml")
	if err != nil {
		return "", err
	}
	if err := atomicWrite(targetPath, data); err != nil {
		return "", err
	}

	// Name changed → delete the old file after the new one is written.
	// Skip for case-only renames (same file, different case) to avoid self-deletion.
	if req.FilePath != "" && !strings.EqualFold(req.FilePath, targetPath) {
		_ = os.Remove(req.FilePath)
	}

	return targetPath, nil
}

// GetEnvironment loads a full environment (including variable values) for editing.
func (w *WorkspaceService) GetEnvironment(path string) (*EnvironmentIO, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	env, err := serialization.ParseEnvironmentFile(data, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	out := &EnvironmentIO{Path: path, Name: env.Name}
	for _, v := range env.Variables {
		out.Variables = append(out.Variables, EnvironmentVariableIO{Name: v.Name, Value: v.Value, Secret: v.Secret})
	}
	return out, nil
}

// SaveEnvironment creates or updates an environment file.
// If env.Path is empty, a new file is created in the workspace environments directory.
func (w *WorkspaceService) SaveEnvironment(env EnvironmentIO) (*EnvironmentInfoIO, error) {
	path := env.Path
	if path == "" {
		if w.workspacePath == "" {
			return nil, fmt.Errorf("no workspace open — open or create a workspace first")
		}
		envDir := filepath.Join(w.workspacePath, "environments")
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return nil, err
		}
		name := sanitizeFilename(env.Name)
		if name == "" {
			name = "environment"
		}
		path = filepath.Join(envDir, name+".yaml")
	}

	domEnv := &domain.Environment{
		SchemaVersion: domain.CurrentSchemaVersion,
		Name:          env.Name,
	}
	for _, v := range env.Variables {
		domEnv.Variables = append(domEnv.Variables, domain.EnvironmentVariable{
			Name:   v.Name,
			Value:  v.Value,
			Secret: v.Secret,
		})
	}

	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		ext = "yaml"
		path += ".yaml"
	}
	data, err := serialization.WriteEnvironment(domEnv, ext)
	if err != nil {
		return nil, err
	}
	if err := atomicWrite(path, data); err != nil {
		return nil, err
	}
	if path == w.activeEnvPath {
		w.activeEnv = domEnv
	}
	return &EnvironmentInfoIO{Name: env.Name, Path: path}, nil
}

// CreateFolder creates a new subdirectory for organising requests.
// If parentPath is empty, the folder is created under {workspace}/requests/.
func (w *WorkspaceService) CreateFolder(parentPath, name string) (string, error) {
	if parentPath == "" {
		if w.workspacePath == "" {
			return "", fmt.Errorf("no workspace open — open or create a workspace first")
		}
		parentPath = filepath.Join(w.workspacePath, "requests")
		if err := os.MkdirAll(parentPath, 0755); err != nil {
			return "", err
		}
	}
	if w.workspacePath != "" {
		rel, err := filepath.Rel(w.workspacePath, parentPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path is outside the current workspace")
		}
	}
	dirName := sanitizeDirName(name)
	if dirName == "" {
		return "", fmt.Errorf("invalid folder name")
	}
	newPath := filepath.Join(parentPath, dirName)
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("a folder named %q already exists here", name)
	}
	if err := os.MkdirAll(newPath, 0755); err != nil {
		return "", fmt.Errorf("creating folder: %w", err)
	}
	return newPath, nil
}

// DeleteFolder removes an empty folder from the workspace.
// Hidden files (e.g. .DS_Store) are ignored when checking emptiness.
func (w *WorkspaceService) DeleteFolder(path string) error {
	if w.workspacePath != "" {
		rel, err := filepath.Rel(w.workspacePath, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("path is outside the current workspace")
		}
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return fmt.Errorf("folder is not empty")
		}
	}
	return os.RemoveAll(path)
}

// RenameFolder renames a workspace folder.
// Returns the new path.
func (w *WorkspaceService) RenameFolder(path, newName string) (string, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return "", fmt.Errorf("name cannot be empty")
	}
	if w.workspacePath != "" {
		rel, err := filepath.Rel(w.workspacePath, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path is outside the current workspace")
		}
	}
	dirName := sanitizeDirName(newName)
	if dirName == "" {
		return "", fmt.Errorf("invalid folder name")
	}
	newPath := filepath.Join(filepath.Dir(path), dirName)
	if newPath == path {
		return path, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("a folder named %q already exists here", newName)
	}
	if err := os.Rename(path, newPath); err != nil {
		return "", fmt.Errorf("renaming folder: %w", err)
	}
	return newPath, nil
}

// RenameRequest renames a request file, updating the name field inside the YAML.
// Returns the new file path.
func (w *WorkspaceService) RenameRequest(path, newName string) (string, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return "", fmt.Errorf("name cannot be empty")
	}
	if w.workspacePath != "" {
		rel, err := filepath.Rel(w.workspacePath, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path is outside the current workspace")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	req, err := serialization.ParseRequestFile(data, filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}
	req.Title = newName

	slug := sanitizeDirName(newName)
	if slug == "" {
		slug = "request"
	}
	newPath := filepath.Join(filepath.Dir(path), slug+".yaml")
	if newPath != path {
		if _, err := os.Stat(newPath); err == nil {
			return "", fmt.Errorf("a request named %q already exists in this folder", newName)
		}
	}
	newData, err := serialization.WriteRequest(req, "yaml")
	if err != nil {
		return "", err
	}
	if err := atomicWrite(newPath, newData); err != nil {
		return "", err
	}
	if newPath != path {
		_ = os.Remove(path)
	}
	return newPath, nil
}

// MoveRequest moves a request file into a different folder within the workspace.
// Returns the new file path.
func (w *WorkspaceService) MoveRequest(srcPath, destDir string) (string, error) {
	if w.workspacePath != "" {
		for _, p := range []string{srcPath, destDir} {
			rel, err := filepath.Rel(w.workspacePath, p)
			if err != nil || strings.HasPrefix(rel, "..") {
				return "", fmt.Errorf("path is outside the current workspace")
			}
		}
	}
	newPath := filepath.Join(destDir, filepath.Base(srcPath))
	if newPath == srcPath {
		return srcPath, nil // already in this folder
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating destination: %w", err)
	}
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("a file named %q already exists in the destination folder", filepath.Base(srcPath))
	}
	if err := os.Rename(srcPath, newPath); err != nil {
		return "", fmt.Errorf("moving file: %w", err)
	}
	return newPath, nil
}

// CopyRequest copies a request file into a different folder within the workspace.
// Returns the new file path, or an error.
func (w *WorkspaceService) CopyRequest(srcPath, destDir string) (string, error) {
	if w.workspacePath != "" {
		for _, p := range []string{srcPath, destDir} {
			rel, err := filepath.Rel(w.workspacePath, p)
			if err != nil || strings.HasPrefix(rel, "..") {
				return "", fmt.Errorf("path is outside the current workspace")
			}
		}
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating destination: %w", err)
	}
	base := filepath.Base(srcPath)
	newPath := filepath.Join(destDir, base)
	if _, err := os.Stat(newPath); err == nil {
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		for i := 2; ; i++ {
			if i == 2 {
				newPath = filepath.Join(destDir, stem+"-copy"+ext)
			} else {
				newPath = filepath.Join(destDir, fmt.Sprintf("%s-copy-%d%s", stem, i, ext))
			}
			if _, err := os.Stat(newPath); err != nil {
				break
			}
		}
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("reading source: %w", err)
	}
	if err := atomicWrite(newPath, data); err != nil {
		return "", fmt.Errorf("writing copy: %w", err)
	}
	return newPath, nil
}

// DeleteRequest removes a request file from disk.
func (w *WorkspaceService) DeleteRequest(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	// Safety: only allow deletion of files within the workspace.
	if w.workspacePath != "" {
		rel, err := filepath.Rel(w.workspacePath, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("path is outside the current workspace")
		}
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting %s: %w", path, err)
	}
	return nil
}

// ValidateEnvironmentYAML parses YAML content as an environment file.
// Returns "" on success, or a human-readable error on failure.
func (w *WorkspaceService) ValidateEnvironmentYAML(content string) string {
	_, err := serialization.ParseEnvironmentFile([]byte(content), "env.yaml")
	if err != nil {
		return err.Error()
	}
	return ""
}

// GetEnvironmentRaw returns the raw YAML/JSON text of an environment file for direct editing.
func (w *WorkspaceService) GetEnvironmentRaw(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(data), nil
}

// SaveEnvironmentRaw validates that content parses as a valid environment file, writes it
// atomically, and returns the updated environment info.
func (w *WorkspaceService) SaveEnvironmentRaw(path, content string) (*EnvironmentInfoIO, error) {
	if path == "" {
		if w.workspacePath == "" {
			return nil, fmt.Errorf("no workspace open — open or create a workspace first")
		}
		return nil, fmt.Errorf("cannot save raw YAML without a file path — save via the form first")
	}

	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		ext = "yaml"
	}

	env, err := serialization.ParseEnvironmentFile([]byte(content), filepath.Base(path))
	if err != nil {
		return nil, fmt.Errorf("invalid environment YAML: %w", err)
	}
	if err := atomicWrite(path, []byte(content)); err != nil {
		return nil, err
	}

	info := &EnvironmentInfoIO{Path: path, Name: env.Name}
	if path == w.activeEnvPath {
		w.activeEnv = env
	}
	return info, nil
}

// DeleteEnvironment removes an environment file.
// If it was the active environment, the active selection is cleared.
func (w *WorkspaceService) DeleteEnvironment(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting %s: %w", path, err)
	}
	if path == w.activeEnvPath {
		w.activeEnv = nil
		w.activeEnvPath = ""
		w.persistConfig()
	}
	return nil
}

// OpenRequest opens a file picker and loads a request file.
func (w *WorkspaceService) OpenRequest() (*RequestFileIO, error) {
	defaultDir := w.workspacePath
	if defaultDir == "" {
		defaultDir, _ = os.UserHomeDir()
	}
	path, err := runtime.OpenFileDialog(w.ctx, runtime.OpenDialogOptions{
		Title:            "Open Request",
		DefaultDirectory: defaultDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Request files (*.yaml, *.yml, *.json)", Pattern: "*.yaml;*.yml;*.json"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil // user cancelled
	}
	return w.LoadRequest(path)
}

// ---- helpers ----

func (w *WorkspaceService) persistConfig() {
	saveConfig(workspaceConfig{
		WorkspacePath:    w.workspacePath,
		ActiveEnvPath:    w.activeEnvPath,
		RecentWorkspaces: w.recentWorkspaces,
	})
}

func (w *WorkspaceService) addRecentWorkspace(path string) {
	if path == "" || isTempWorkspace(path) {
		return
	}
	seen := make(map[string]bool)
	recent := []string{path}
	seen[path] = true
	for _, p := range w.recentWorkspaces {
		if !seen[p] {
			seen[p] = true
			recent = append(recent, p)
		}
	}
	if len(recent) > 10 {
		recent = recent[:10]
	}
	w.recentWorkspaces = recent
}

// GetRecentWorkspaces returns the list of recently opened workspace paths (most recent first).
func (w *WorkspaceService) GetRecentWorkspaces() []string {
	if len(w.recentWorkspaces) == 0 {
		return []string{}
	}
	return w.recentWorkspaces
}

// OpenWorkspaceByPath opens a workspace directly by path without a dialog.
func (w *WorkspaceService) OpenWorkspaceByPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("workspace not found: %s", path)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}
	w.workspacePath = path
	w.addRecentWorkspace(path)
	w.persistConfig()
	w.syncTitle()
	return path, nil
}

func (w *WorkspaceService) loadEnvFile(path string) (*domain.Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return serialization.ParseEnvironmentFile(data, filepath.Base(path))
}

func isDataFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml" || ext == ".json"
}

func isRequestFile(path string) bool {
	// Quick heuristic: if it's a data file we treat it as a potential request.
	// A full parse happens only when the user loads it.
	return isDataFile(filepath.Base(path))
}

// peekMethod reads just the method field from a request file without full parsing.
func peekMethod(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var out struct {
		Method string `yaml:"method" json:"method"`
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		if err := json.Unmarshal(data, &out); err != nil {
			return ""
		}
	} else {
		if err := yaml.Unmarshal(data, &out); err != nil {
			return ""
		}
	}
	return strings.ToUpper(out.Method)
}

func buildFileTree(dir string) ([]FileNodeIO, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var nodes []FileNodeIO
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		if e.IsDir() {
			children, err := buildFileTree(path)
			if err != nil {
				continue
			}
			nodes = append(nodes, FileNodeIO{Name: name, Path: path, IsDir: true, Children: children})
		} else if isRequestFile(path) {
			nodes = append(nodes, FileNodeIO{Name: name, Path: path, IsDir: false, Method: peekMethod(path)})
		}
	}
	return nodes, nil
}

func domainToRequestFileIO(req *domain.Request) *RequestFileIO {
	out := &RequestFileIO{
		SchemaVersion: req.SchemaVersion,
		Title:         req.Title,
		Method:        string(req.Method),
		URL:           req.URL,
	}
	for _, h := range req.Headers {
		out.Headers = append(out.Headers, HeaderIO{Name: h.Name, Value: h.Value})
	}
	for _, p := range req.QueryParams {
		out.QueryParams = append(out.QueryParams, QueryParamIO{Name: p.Name, Value: p.Value})
	}
	if req.Body != nil {
		out.Body = &BodyIO{Type: string(req.Body.Type), Content: req.Body.Content}
	}
	out.Auth = authDomainToIO(req.Auth)
	if req.Timeout > 0 {
		out.TimeoutMs = req.Timeout.Milliseconds()
	}
	out.NoFollowRedirects = req.NoFollowRedirects
	return out
}

// authDomainToIO maps a request's auth block for the UI; nil stays nil.
func authDomainToIO(a *domain.Auth) *AuthIO {
	if a == nil {
		return nil
	}
	af := &AuthIO{Type: string(a.Type)}
	if a.Bearer != nil {
		af.Token = a.Bearer.Token
	}
	if a.Basic != nil {
		af.Username = a.Basic.Username
		af.Password = a.Basic.Password
	}
	if a.APIKey != nil {
		af.Key = a.APIKey.Key
		af.Value = a.APIKey.Value
		af.Location = string(a.APIKey.Location)
	}
	return af
}

func requestFileIOToDomain(r RequestFileIO) *domain.Request {
	req := &domain.Request{
		Title:  r.Title,
		Method: domain.HTTPMethod(r.Method),
		URL:    r.URL,
	}
	for _, h := range r.Headers {
		req.Headers = append(req.Headers, domain.Header{Name: h.Name, Value: h.Value})
	}
	for _, p := range r.QueryParams {
		req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: p.Name, Value: p.Value})
	}
	if r.Body != nil {
		req.Body = &domain.RequestBody{Type: domain.BodyType(r.Body.Type), Content: r.Body.Content}
	}
	if r.Auth != nil {
		req.Auth = inputAuthToDomain(r.Auth)
	}
	if r.TimeoutMs > 0 {
		req.Timeout = time.Duration(r.TimeoutMs) * time.Millisecond
	}
	req.NoFollowRedirects = r.NoFollowRedirects
	return req
}

// CreateExampleWorkspace creates an example workspace structure in the given directory.
// Called once when the user creates a new workspace.
func createExampleWorkspace(dir string) error {
	for _, sub := range []string{"environments", "requests"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return err
		}
	}

	env := `schemaVersion: "1"
name: Local Dev
variables:
  - name: baseUrl
    value: http://localhost:8080
  - name: authToken
    value: ""
    secret: true
`
	req := `schemaVersion: "1"
title: Health Check
method: GET
url: "{{baseUrl}}/health"
headers:
  - name: Accept
    value: application/json
`
	writeFile := func(path, content string) error {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.WriteString(f, content)
		return err
	}

	// Gitignore environment files (secrets) and the response cache.
	if err := writeFile(filepath.Join(dir, ".gitignore"), ".clrsigcache/\n"); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, "environments", ".gitignore"), "*\n"); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, "environments", "local.yaml"), env); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, "requests", "health-check.yaml"), req); err != nil {
		return err
	}
	return nil
}

// NewWorkspace opens a dialog to pick or create a directory and initialises workspace structure.
func (w *WorkspaceService) NewWorkspace() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(w.ctx, runtime.OpenDialogOptions{
		Title: "Choose Workspace Folder (new or existing)",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return w.workspacePath, nil
	}
	// Create example structure only if directory is empty
	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		if err := createExampleWorkspace(dir); err != nil {
			log.Printf("creating example workspace: %v", err)
		}
	}
	w.workspacePath = dir
	w.addRecentWorkspace(dir)
	w.persistConfig()
	w.syncTitle()
	return dir, nil
}

// ---- Response cache ----

// ResponseCacheIO holds the last response for a request file, persisted to .clrsigcache/.
type ResponseCacheIO struct {
	CachedAt   string     `json:"cachedAt"`
	StatusCode int        `json:"statusCode"`
	Status     string     `json:"status"`
	Headers    []HeaderIO `json:"headers,omitempty"`
	Body       string     `json:"body"`
	DurationMs int64      `json:"durationMs"`
	Size       int64      `json:"size"`
	Timing     TimingIO   `json:"timing"`
}

func (w *WorkspaceService) cachePath(reqPath string) (string, error) {
	if w.workspacePath == "" {
		return "", fmt.Errorf("no workspace open")
	}
	rel, err := filepath.Rel(w.workspacePath, reqPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside the current workspace")
	}
	base := strings.TrimSuffix(strings.TrimSuffix(rel, ".yaml"), ".yml")
	return filepath.Join(w.workspacePath, ".clrsigcache", base+".json"), nil
}

// SaveResponseCache persists the last response for a request file.
func (w *WorkspaceService) SaveResponseCache(reqPath string, response ResponseCacheIO) error {
	path, err := w.cachePath(reqPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	response.CachedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

// ClearResponseCache deletes the cached response for a request file.
func (w *WorkspaceService) ClearResponseCache(reqPath string) error {
	path, err := w.cachePath(reqPath)
	if err != nil {
		return nil // no workspace — nothing to clear
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// LoadResponseCache returns the last cached response for a request file, or nil if none exists.
func (w *WorkspaceService) LoadResponseCache(reqPath string) (*ResponseCacheIO, error) {
	path, err := w.cachePath(reqPath)
	if err != nil {
		return nil, nil // no workspace — silently return nothing
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cache ResponseCacheIO
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// SaveContentToFile opens a system save dialog and writes content to the chosen path.
// Returns the saved path, or an empty string if the user cancelled.
func (w *WorkspaceService) SaveContentToFile(content, defaultName string) (string, error) {
	path, err := runtime.SaveFileDialog(w.ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: defaultName,
	})
	if err != nil || path == "" {
		return "", nil
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return path, nil
}

// PickFile opens a native file dialog and returns the selected path.
// Returns an empty string if the user cancels.
func (w *WorkspaceService) PickFile(title string) string {
	path, err := runtime.OpenFileDialog(w.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return ""
	}
	return path
}

// ---- Workspace export / import ----

type ScanImportResultIO struct {
	ZipPath    string   `json:"zipPath"`
	TotalFiles int      `json:"totalFiles"`
	Conflicts  []string `json:"conflicts"`
	Error      string   `json:"error"`
}

type ApplyImportResultIO struct {
	Imported int    `json:"imported"`
	Skipped  int    `json:"skipped"`
	Renamed  int    `json:"renamed"`
	Error    string `json:"error"`
}

// WorkspaceHasSecrets returns true if any environment file in the workspace
// contains at least one variable marked secret: true with a non-empty value.
func (w *WorkspaceService) WorkspaceHasSecrets() bool {
	if w.workspacePath == "" {
		return false
	}
	envDir := filepath.Join(w.workspacePath, "environments")
	entries, err := os.ReadDir(envDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(envDir, name))
		if err != nil {
			continue
		}
		env, err := serialization.ParseEnvironmentFile(data, name)
		if err != nil {
			continue
		}
		for _, v := range env.Variables {
			if v.Secret && v.Value != "" {
				return true
			}
		}
	}
	return false
}

// ExportWorkspace packs requests/, scenarios/, and environments/ into a ZIP
// and saves it via a file dialog. Returns "" on success or an error message.
func (w *WorkspaceService) ExportWorkspace() string {
	if w.workspacePath == "" {
		return "no workspace open"
	}
	wsName := filepath.Base(w.workspacePath)
	defaultName := wsName + "-" + time.Now().Format("2006-01-02") + ".zip"
	savePath, err := runtime.SaveFileDialog(w.ctx, runtime.SaveDialogOptions{
		Title:           "Export Workspace",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP archive (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil || savePath == "" {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(savePath), ".zip") {
		savePath += ".zip"
	}
	if err := createWorkspaceZip(w.workspacePath, savePath); err != nil {
		return err.Error()
	}
	return ""
}

// ScanWorkspaceImport opens a file picker for a ZIP, reads its manifest, and
// returns conflict information without extracting anything.
// A ZipPath of "" means the user cancelled.
func (w *WorkspaceService) ScanWorkspaceImport() ScanImportResultIO {
	if w.workspacePath == "" {
		return ScanImportResultIO{Error: "no workspace open"}
	}
	path, err := runtime.OpenFileDialog(w.ctx, runtime.OpenDialogOptions{
		Title: "Import Workspace ZIP",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP archive (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil || path == "" {
		return ScanImportResultIO{}
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		return ScanImportResultIO{Error: "invalid ZIP: " + err.Error()}
	}
	defer r.Close()
	var conflicts []string
	total := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		total++
		dest := filepath.Join(w.workspacePath, filepath.FromSlash(f.Name))
		if _, err := os.Stat(dest); err == nil {
			conflicts = append(conflicts, f.Name)
		}
	}
	return ScanImportResultIO{ZipPath: path, TotalFiles: total, Conflicts: conflicts}
}

// ApplyWorkspaceImport extracts zipPath into the current workspace.
// strategy: "skip" | "overwrite" | "rename"
func (w *WorkspaceService) ApplyWorkspaceImport(zipPath, strategy string) ApplyImportResultIO {
	if w.workspacePath == "" {
		return ApplyImportResultIO{Error: "no workspace open"}
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return ApplyImportResultIO{Error: "invalid ZIP: " + err.Error()}
	}
	defer r.Close()
	result := ApplyImportResultIO{}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		dest := filepath.Join(w.workspacePath, filepath.FromSlash(f.Name))
		if rel, err := filepath.Rel(w.workspacePath, dest); err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		_, existErr := os.Stat(dest)
		exists := existErr == nil
		if exists {
			switch strategy {
			case "skip":
				result.Skipped++
				continue
			case "rename":
				dest = findNonConflictingPath(dest)
				result.Renamed++
			default: // "overwrite" — fall through
			}
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			result.Error = err.Error()
			return result
		}
		rc, err := f.Open()
		if err != nil {
			result.Error = err.Error()
			return result
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			result.Error = readErr.Error()
			return result
		}
		if err := atomicWrite(dest, data); err != nil {
			result.Error = err.Error()
			return result
		}
		result.Imported++
	}
	return result
}

// createWorkspaceZip writes requests/, scenarios/, and environments/ into a ZIP
// at destPath using an atomic temp-file + rename.
func createWorkspaceZip(workspacePath, destPath string) error {
	tmp := destPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("creating zip: %w", err)
	}
	success := false
	defer func() {
		if !success {
			os.Remove(tmp)
		}
	}()
	zw := zip.NewWriter(f)
	for _, dir := range []string{"requests", "scenarios", "environments"} {
		dirPath := filepath.Join(workspacePath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}
		if err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || strings.HasPrefix(info.Name(), ".") {
				return nil
			}
			rel, err := filepath.Rel(workspacePath, path)
			if err != nil {
				return nil
			}
			w, err := zw.Create(filepath.ToSlash(rel))
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, err = w.Write(data)
			return err
		}); err != nil {
			f.Close()
			return fmt.Errorf("walking %s: %w", dir, err)
		}
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return fmt.Errorf("closing zip: %w", err)
	}
	f.Close()
	success = true
	return os.Rename(tmp, destPath)
}

// findNonConflictingPath appends -2, -3, … to the stem until the path is free.
func findNonConflictingPath(path string) string {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}
