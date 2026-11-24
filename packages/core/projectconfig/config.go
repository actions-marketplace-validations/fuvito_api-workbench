// Package projectconfig reads clrsig.yaml (or .yml, .json): the defaults of a project for the flags
// that repeat from one command to the next. The file holds no secrets and no variables; it is
// meant to be committed. Commands apply it only to flags the user did not give.
package projectconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fuvito/api-workbench/packages/core/checks"
	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// SchemaVersion is the version of the file format this package reads and the only one it knows.
const SchemaVersion = "1"

// FileNames are the names looked for in a folder, in this order. A folder may hold only one.
var FileNames = []string{"clrsig.yaml", "clrsig.yml", "clrsig.json"}

// Config is the content of a project file. Every key is optional. Pointers tell "not set" from
// false or zero, so a file that says readOnly: false overrides nothing by accident.
type Config struct {
	SchemaVersion string   `json:"schemaVersion"`
	Spec          string   `json:"spec,omitempty"`
	Environment   string   `json:"environment,omitempty"`
	BaseURL       string   `json:"baseUrl,omitempty"`
	Conform       *Conform `json:"conform,omitempty"`
	Run           *Run     `json:"run,omitempty"`
	Reports       []string `json:"reports,omitempty"`

	path string
	dir  string
}

// Conform holds the defaults of clrsig conform.
type Conform struct {
	ReadOnly           *bool    `json:"readOnly,omitempty"`
	Negative           *bool    `json:"negative,omitempty"`
	FailOn             string   `json:"failOn,omitempty"`
	RequestTimeoutMs   *int     `json:"requestTimeoutMs,omitempty"`
	MaxResponseTimeMs  *int     `json:"maxResponseTimeMs,omitempty"`
	MaxResponseTimeFor []string `json:"maxResponseTimeFor,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	Paths              []string `json:"paths,omitempty"`
	Baseline           string   `json:"baseline,omitempty"`
}

// Run holds the defaults of clrsig run, validate and impact.
type Run struct {
	Scenarios []string `json:"scenarios,omitempty"`
	FailFast  *bool    `json:"failFast,omitempty"`
}

// Path is the file the configuration was read from.
func (c *Config) Path() string { return c.path }

// Dir is the folder of that file; relative paths in it are relative to this folder.
func (c *Config) Dir() string { return c.dir }

// ForCommandLine turns a path written in the file into the one to put on a command line: a URL or
// an absolute path stays as it is, any other path is resolved against the folder of the file and
// then written relative to cwd when that stays below it (so reports show short, portable paths),
// otherwise as an absolute path.
func (c *Config) ForCommandLine(value, cwd string) string {
	if value == "" || strings.Contains(value, "://") || filepath.IsAbs(value) {
		return value
	}
	abs := filepath.Join(c.dir, filepath.FromSlash(value))
	if cwd != "" {
		if rel, err := filepath.Rel(cwd, abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return abs
}

// Discover looks for a project file, starting in startDir and walking up. It stops at the first
// folder that has one. A folder that contains .git, and the home folder, are the last ones
// looked at. It returns "" when there is none, and an error when one folder holds several files.
func Discover(startDir, homeDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	home := ""
	if homeDir != "" {
		home, _ = filepath.Abs(homeDir)
	}
	for i := 0; i < 40; i++ {
		var found []string
		for _, name := range FileNames {
			if st, err := os.Stat(filepath.Join(dir, name)); err == nil && !st.IsDir() {
				found = append(found, filepath.Join(dir, name))
			}
		}
		switch {
		case len(found) > 1:
			names := make([]string, len(found))
			for i, f := range found {
				names[i] = filepath.Base(f)
			}
			return "", fmt.Errorf("%s has more than one project file (%s): keep one, or choose with --config", dir, strings.Join(names, ", "))
		case len(found) == 1:
			return found[0], nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", nil
		}
		if home != "" && sameDir(dir, home) {
			return "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
	return "", nil
}

func sameDir(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }

// Load reads and validates a project file. Every error says which file and which key.
func Load(path string) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("project file %s does not exist", path)
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	cfg, err := Parse(data, abs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(abs), err)
	}
	cfg.path, cfg.dir = abs, filepath.Dir(abs)
	return cfg, nil
}

// Parse reads the content of a project file; filename decides YAML or JSON by its extension.
func Parse(data []byte, filename string) (*Config, error) {
	var raw map[string]any
	var err error
	if strings.EqualFold(filepath.Ext(filename), ".json") {
		err = json.Unmarshal(data, &raw)
	} else {
		err = yaml.Unmarshal(data, &raw)
	}
	if err != nil {
		return nil, fmt.Errorf("it is not valid %s: %w", formatName(filename), err)
	}
	if raw == nil {
		return nil, errors.New("it is empty; the smallest valid file is: schemaVersion: \"1\"")
	}
	if err := checkVersion(raw); err != nil {
		return nil, err
	}
	stripExtensions(raw)
	for _, section := range []string{"conform", "run"} {
		if m, ok := raw[section].(map[string]any); ok {
			stripExtensions(m)
		}
	}
	if err := checkKeys(raw); err != nil {
		return nil, err
	}
	// one model, two serializations: both formats are normalised to JSON and decoded the same way
	normal, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("it holds a value that is not plain data: %w", err)
	}
	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(normal))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, describeDecodeError(err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Keys lists every setting a project file can hold, written as in the file (conform.readOnly,
// run.scenarios, ...). Commands map each one to a flag or argument; a test keeps the two lists equal.
func Keys() []string {
	var out []string
	for _, k := range sectionKeys[""] {
		if k == "schemaVersion" || k == "conform" || k == "run" {
			continue
		}
		out = append(out, k)
	}
	for _, section := range []string{"conform", "run"} {
		for _, k := range sectionKeys[section] {
			out = append(out, section+"."+k)
		}
	}
	return out
}

var sectionKeys = map[string][]string{
	"":        {"schemaVersion", "spec", "environment", "baseUrl", "conform", "run", "reports"},
	"conform": {"readOnly", "negative", "failOn", "requestTimeoutMs", "maxResponseTimeMs", "maxResponseTimeFor", "tags", "paths", "baseline"},
	"run":     {"scenarios", "failFast"},
}

// checkKeys rejects a key that is not exactly one of the known ones. The JSON decoder accepts a
// different letter case, which would let one project file mean two things in two tools.
func checkKeys(raw map[string]any) error {
	for _, section := range []string{"", "conform", "run"} {
		m := raw
		if section != "" {
			sub, ok := raw[section].(map[string]any)
			if !ok {
				continue
			}
			m = sub
		}
		keys := append([]string(nil), mapKeys(m)...)
		sort.Strings(keys)
		for _, k := range keys {
			if contains(sectionKeys[section], k) {
				continue
			}
			where := ""
			if section != "" {
				where = " under " + section
			}
			hint := ""
			if s := nearest(k); s != "" {
				hint = fmt.Sprintf("; did you mean %q?", s)
			}
			return fmt.Errorf("unknown key %q%s%s Keys that start with x- are kept for your own use", k, where, hint)
		}
	}
	return nil
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func formatName(filename string) string {
	if strings.EqualFold(filepath.Ext(filename), ".json") {
		return "JSON"
	}
	return "YAML"
}

func checkVersion(raw map[string]any) error {
	v, ok := raw["schemaVersion"]
	if !ok {
		return errors.New(`schemaVersion is missing: add schemaVersion: "1" (a quoted string)`)
	}
	s, isString := v.(string)
	if !isString {
		return fmt.Errorf(`schemaVersion must be the quoted string "1", not %v`, v)
	}
	if s != SchemaVersion {
		return fmt.Errorf("schemaVersion %q is not supported (this version of clrsig reads %q; a newer file needs a newer clrsig)", s, SchemaVersion)
	}
	return nil
}

func stripExtensions(m map[string]any) {
	for k := range m {
		if strings.HasPrefix(k, "x-") {
			delete(m, k)
		}
	}
}

var knownKeys = []string{
	"schemaVersion", "spec", "environment", "baseUrl", "conform", "run", "reports",
	"readOnly", "negative", "failOn", "requestTimeoutMs", "maxResponseTimeMs", "maxResponseTimeFor", "tags", "paths", "baseline",
	"scenarios", "failFast",
}

func describeDecodeError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "the file"
		}
		return fmt.Errorf("%s has the wrong type: expected %s, found %s", field, friendlyType(typeErr.Type.String()), typeErr.Value)
	}
	msg := err.Error()
	if rest, ok := strings.CutPrefix(msg, "json: unknown field "); ok {
		name := strings.Trim(rest, `"`)
		hint := ""
		if s := nearest(name); s != "" {
			hint = fmt.Sprintf("; did you mean %q?", s)
		}
		return fmt.Errorf("unknown key %q%s Keys that start with x- are kept for your own use", name, hint)
	}
	return fmt.Errorf("%s", strings.TrimPrefix(msg, "json: "))
}

func friendlyType(t string) string {
	switch {
	case strings.HasPrefix(t, "[]"):
		return "a list"
	case t == "bool" || t == "*bool":
		return "true or false"
	case strings.HasPrefix(t, "int") || strings.HasPrefix(t, "*int"):
		return "a whole number"
	case t == "string":
		return "text"
	}
	return t
}

// nearest suggests the known key closest to a misspelled one.
func nearest(name string) string {
	best, bestD := "", 3
	for _, k := range knownKeys {
		if d := distance(strings.ToLower(name), strings.ToLower(k)); d < bestD {
			best, bestD = k, d
		}
	}
	return best
}

func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

func (c *Config) validate() error {
	// ${NAME} belongs in environment files, where it is expanded when a run starts. A project file
	// is committed, so it must not be a place for values that look like secrets.
	var bad []string
	check := func(key, value string) {
		if strings.Contains(value, "${") {
			bad = append(bad, key)
		}
	}
	check("spec", c.Spec)
	check("environment", c.Environment)
	check("baseUrl", c.BaseURL)
	for _, r := range c.Reports {
		check("reports", r)
	}
	if c.Conform != nil {
		check("conform.baseline", c.Conform.Baseline)
		for _, v := range c.Conform.Tags {
			check("conform.tags", v)
		}
		for _, v := range c.Conform.Paths {
			check("conform.paths", v)
		}
		for _, v := range c.Conform.MaxResponseTimeFor {
			check("conform.maxResponseTimeFor", v)
		}
	}
	if c.Run != nil {
		for _, v := range c.Run.Scenarios {
			check("run.scenarios", v)
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("%s contains ${...}, which is not expanded in a project file: put values that come from the environment in an environment file (environments/*.yaml) and name that file under environment", strings.Join(dedupe(bad), ", "))
	}
	if c.Conform != nil {
		if _, err := checks.ParseFailOn(c.Conform.FailOn); err != nil {
			return fmt.Errorf("conform.failOn %q is not valid: use failure (the default) or warning", c.Conform.FailOn)
		}
		for name, v := range map[string]*int{"conform.requestTimeoutMs": c.Conform.RequestTimeoutMs, "conform.maxResponseTimeMs": c.Conform.MaxResponseTimeMs} {
			if v != nil && *v < 0 {
				return fmt.Errorf("%s must not be negative, found %d", name, *v)
			}
		}
		if _, err := conformance.ParsePathLimits(c.Conform.MaxResponseTimeFor); err != nil {
			return fmt.Errorf("conform.maxResponseTimeFor: %w", err)
		}
	}
	return nil
}

func dedupe(s []string) []string {
	var out []string
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}
