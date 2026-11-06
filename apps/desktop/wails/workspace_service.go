package main

import (
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
)

// ---- Config persistence ----

type workspaceConfig struct {
	WorkspacePath string `json:"workspacePath"`
	ActiveEnvPath string `json:"activeEnvPath"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".apidesk")
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
	SchemaVersion string          `json:"schemaVersion"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Method        string          `json:"method"`
	URL           string          `json:"url"`
	Headers       []HeaderIO      `json:"headers,omitempty"`
	QueryParams   []QueryParamIO  `json:"queryParams,omitempty"`
	Body          *BodyIO         `json:"body,omitempty"`
	Auth          *AuthIO         `json:"auth,omitempty"`
	TimeoutMs     int64           `json:"timeoutMs,omitempty"`
	// FilePath is the on-disk path of the loaded file; not persisted to YAML.
	FilePath      string          `json:"filePath,omitempty"`
	// TargetDir overrides the save directory for new (unsaved) requests.
	TargetDir     string          `json:"targetDir,omitempty"`
}

// ---- WorkspaceService ----

type WorkspaceService struct {
	ctx           context.Context
	workspacePath string
	activeEnvPath string
	activeEnv     *domain.Environment
}

func NewWorkspaceService() *WorkspaceService {
	cfg := loadConfig()
	ws := &WorkspaceService{
		ctx:           context.Background(),
		workspacePath: cfg.WorkspacePath,
		activeEnvPath: cfg.ActiveEnvPath,
	}
	if cfg.ActiveEnvPath != "" {
		if env, err := ws.loadEnvFile(cfg.ActiveEnvPath); err == nil {
			ws.activeEnv = env
		}
	}
	return ws
}

func (w *WorkspaceService) setContext(ctx context.Context) {
	w.ctx = ctx
}

func (w *WorkspaceService) ActiveEnvironment() *domain.Environment {
	return w.activeEnv
}

// openWorkspaceDirect sets the workspace path directly without a dialog.
// It also auto-selects the environments/local.yaml if present.
// Does not persist to config (temp demo workspace should not pollute saved state).
func (w *WorkspaceService) openWorkspaceDirect(path string) {
	w.workspacePath = path
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
	w.persistConfig()
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
}

// ListRequests returns the workspace file tree limited to request files (.yaml/.yml/.json).
func (w *WorkspaceService) ListRequests() []FileNodeIO {
	if w.workspacePath == "" {
		return nil
	}
	nodes, err := buildFileTree(w.workspacePath, w.workspacePath, true)
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
// The target path is derived from req.Name: {dir}/{slug}.yaml.
// If req.FilePath is set (file already exists), the directory is preserved and
// the file is renamed when the name slug changes. Returns the saved path.
func (w *WorkspaceService) SaveRequest(req RequestFileIO) (string, error) {
	name := strings.TrimSpace(req.Name)
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
	if targetPath != req.FilePath {
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
	if req.FilePath != "" && req.FilePath != targetPath {
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
	req.Name = newName

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
		Title:           "Open Request",
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
		WorkspacePath: w.workspacePath,
		ActiveEnvPath: w.activeEnvPath,
	})
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

func buildFileTree(root, dir string, skipSpecialDirs bool) ([]FileNodeIO, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var nodes []FileNodeIO
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files/dirs and the managed directories at root level
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		if e.IsDir() {
			if skipSpecialDirs && (path == filepath.Join(root, "environments") || path == filepath.Join(root, "scenarios")) {
				continue
			}
			children, err := buildFileTree(root, path, false)
			if err != nil {
				continue
			}
			nodes = append(nodes, FileNodeIO{Name: name, Path: path, IsDir: true, Children: children})
		} else if isRequestFile(path) {
			nodes = append(nodes, FileNodeIO{Name: name, Path: path, IsDir: false})
		}
	}
	return nodes, nil
}

func domainToRequestFileIO(req *domain.Request) *RequestFileIO {
	out := &RequestFileIO{
		SchemaVersion: req.SchemaVersion,
		Name:          req.Name,
		Description:   req.Description,
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
	if req.Auth != nil {
		af := &AuthIO{Type: string(req.Auth.Type)}
		if req.Auth.Bearer != nil {
			af.Token = req.Auth.Bearer.Token
		}
		if req.Auth.Basic != nil {
			af.Username = req.Auth.Basic.Username
			af.Password = req.Auth.Basic.Password
		}
		if req.Auth.APIKey != nil {
			af.Key = req.Auth.APIKey.Key
			af.Value = req.Auth.APIKey.Value
			af.Location = string(req.Auth.APIKey.Location)
		}
		out.Auth = af
	}
	if req.Timeout > 0 {
		out.TimeoutMs = req.Timeout.Milliseconds()
	}
	return out
}

func requestFileIOToDomain(r RequestFileIO) *domain.Request {
	req := &domain.Request{
		Name:        r.Name,
		Description: r.Description,
		Method:      domain.HTTPMethod(r.Method),
		URL:         r.URL,
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
name: Health Check
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
	if err := writeFile(filepath.Join(dir, ".gitignore"), ".apideskcache/\n"); err != nil {
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
	w.persistConfig()
	return dir, nil
}

// ---- Response cache ----

// ResponseCacheIO holds the last response for a request file, persisted to .apideskcache/.
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
	return filepath.Join(w.workspacePath, ".apideskcache", base+".json"), nil
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
