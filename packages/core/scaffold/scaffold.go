// Package scaffold builds the files clrsig init creates: the project file, the starter environment
// files and the GitHub workflow. It only produces text and writes files that do not exist yet; it
// never overwrites, never deletes and never writes outside the folder it is given.
package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Entry is one file or folder init would create. Path uses forward slashes and is relative to the
// project folder. A folder entry creates the folder with a .gitkeep file, and only when the folder
// does not exist yet.
type Entry struct {
	Path    string
	Content string
	Dir     bool
	// Alternatives are other names that already count as this file: if one exists, the entry is
	// kept as it is (a clrsig.yml next to a new clrsig.yaml would be two project files).
	Alternatives []string
}

// ProjectOptions are the values init writes into the project file.
type ProjectOptions struct {
	// Spec is the spec as it should be written in clrsig.yaml: a path relative to the project
	// folder, or a URL. Empty leaves a commented example.
	Spec string
	// BaseURL is the server to check. Empty leaves a commented example.
	BaseURL string
}

// ProjectEntries lists what clrsig init creates, in the order it reports them.
func ProjectEntries(o ProjectOptions) []Entry {
	baseURL := o.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return []Entry{
		{Path: "clrsig.yaml", Content: projectFile(o), Alternatives: []string{"clrsig.yml", "clrsig.json"}},
		{Path: "environments/local.yaml", Content: localEnvironment(baseURL)},
		{Path: "environments/ci.yaml", Content: ciEnvironment},
		{Path: "scenarios", Dir: true},
		{Path: "requests", Dir: true},
		{Path: "specs", Dir: true},
	}
}

func projectFile(o ProjectOptions) string {
	var b strings.Builder
	b.WriteString("# Clear Signal project file. A flag on the command line always wins over a value here.\n")
	b.WriteString("# The keys are listed in the Format Spec (Help > Format Spec in the app). Keep secrets out of this\n")
	b.WriteString("# file: put values that come from the environment in an environment file as ${NAME}.\n")
	b.WriteString("schemaVersion: \"1\"\n\n")
	if o.Spec != "" {
		fmt.Fprintf(&b, "spec: %s\n", yamlScalar(o.Spec))
	} else {
		b.WriteString("# spec: specs/openapi.yaml\n")
	}
	b.WriteString("environment: environments/local.yaml\n")
	if o.BaseURL != "" {
		fmt.Fprintf(&b, "baseUrl: %s\n", yamlScalar(o.BaseURL))
	} else {
		b.WriteString("# baseUrl: http://localhost:8080   # the server to check; defaults to the first servers entry of the spec\n")
	}
	b.WriteString("\nconform:\n")
	b.WriteString("  readOnly: true        # send only GET and HEAD; set to false to also send writes\n")
	return b.String()
}

func localEnvironment(baseURL string) string {
	return "schemaVersion: \"1\"\n" +
		"name: Local\n" +
		"# Values the requests built from your spec use as {{name}}. A run reports every one it needs and\n" +
		"# does not have (for example {{token}} or {{id}}); add them here.\n" +
		"variables:\n" +
		"  - name: baseUrl\n" +
		"    value: " + yamlScalar(baseURL) + "\n"
}

const ciEnvironment = `schemaVersion: "1"
name: CI
# ${NAME} is read from the process environment when a run starts, so a secret is never written to disk.
# In GitHub Actions set it on the step (env: API_TOKEN: ${{ secrets.API_TOKEN }}).
variables:
  - name: baseUrl
    value: ${API_BASE_URL}
  - name: token
    value: ${API_TOKEN}
    secret: true
`

var plainScalar = regexp.MustCompile(`^[A-Za-z0-9_./:@+-][A-Za-z0-9_./:@+=?&%-]*$`)

// yamlScalar writes a value so it reads back as the same text: bare when that is safe, otherwise
// double-quoted.
func yamlScalar(s string) string {
	if plainScalar.MatchString(s) && !strings.HasSuffix(s, ":") && s != "true" && s != "false" && s != "null" {
		return s
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Outcome says what Apply did (or, for a dry run, would do).
type Outcome struct {
	Created []string
	Kept    []string
}

// Apply writes the entries under root. An entry whose file or folder already exists is kept, never
// changed. With dryRun nothing is written. Every path must stay inside root, including through
// symbolic links; otherwise nothing at all is written and the error names the path.
func Apply(root string, entries []Entry, dryRun bool) (Outcome, error) {
	var out Outcome
	abs, err := filepath.Abs(root)
	if err != nil {
		return out, err
	}
	real := realPath(abs)
	// check everything before writing anything
	targets := make([]string, len(entries))
	for i, e := range entries {
		t, err := target(abs, real, e.Path)
		if err != nil {
			return out, err
		}
		targets[i] = t
	}
	for i, e := range entries {
		exists := pathExists(targets[i])
		for _, alt := range e.Alternatives {
			if pathExists(filepath.Join(abs, filepath.FromSlash(alt))) {
				exists = true
			}
		}
		if exists {
			out.Kept = append(out.Kept, e.Path)
			continue
		}
		out.Created = append(out.Created, e.Path)
		if dryRun {
			continue
		}
		if e.Dir {
			if err := os.MkdirAll(targets[i], 0o755); err != nil {
				return out, err
			}
			if err := writeNew(filepath.Join(targets[i], ".gitkeep"), ""); err != nil {
				return out, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targets[i]), 0o755); err != nil {
			return out, err
		}
		if err := writeNew(targets[i], e.Content); err != nil {
			return out, err
		}
	}
	return out, nil
}

// realPath resolves symbolic links in the part of path that exists and keeps the rest as written,
// so a folder that is about to be created still has a real location to compare against.
func realPath(path string) string {
	rest := ""
	for {
		if r, err := filepath.EvalSymlinks(path); err == nil {
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return filepath.Join(path, rest)
		}
		rest = filepath.Join(filepath.Base(path), rest)
		path = parent
	}
}

// WriteNew writes one file that must not exist yet, atomically, creating its folder. It returns an
// error that wraps fs.ErrExist when the file is already there.
func WriteNew(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeNew(path, content)
}

func pathExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// target resolves a relative entry path under root and refuses anything that leaves it.
func target(abs, real, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) || strings.Contains(rel, "..") {
		return "", fmt.Errorf("refusing to write %q: it is not a path inside the project folder", rel)
	}
	full := filepath.Join(abs, filepath.FromSlash(rel))
	// where the file would really end up, after following links in the part that exists
	r := realPath(filepath.Dir(full))
	if rr, err := filepath.Rel(real, r); err != nil || rr == ".." || strings.HasPrefix(rr, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to write %q: it would end up outside the project folder (%s)", rel, r)
	}
	return full, nil
}

// writeNew writes through a temporary file and links it into place, so a reader never sees half a
// file and an existing file is never replaced.
func writeNew(path, content string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".clrsig-init-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(name, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return err
		}
		// a file system without hard links: fall back to a rename after checking again
		if pathExists(path) {
			return fs.ErrExist
		}
		return os.Rename(name, path)
	}
	return nil
}
