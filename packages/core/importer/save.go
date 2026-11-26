package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// SanitizeDirName turns a collection or folder name into a safe folder name: lower case,
// letters, digits and dashes, with spaces and underscores becoming dashes.
func SanitizeDirName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// SanitizeFilename turns a request title into a safe file name stem, keeping case.
func SanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// Saved is one request file written, or in a dry run, one that would be written.
type Saved struct {
	Path   string
	Title  string
	Method string
	URL    string
}

// SaveResult is the outcome of Save.
type SaveResult struct {
	Collection string
	Saved      []Saved
	Skipped    int // requests that could not be written
}

// Save writes imported requests as YAML files under requestsDir/<collection>/<folders>/.
// A name that already exists, or that an earlier request in this import already took, gets a
// -2, -3 … suffix; nothing existing is overwritten. With dryRun it only reports the paths.
func Save(requestsDir, collectionName string, requests []ImportedRequest, dryRun bool) SaveResult {
	dirName := SanitizeDirName(collectionName)
	if dirName == "" {
		dirName = "imported"
	}
	result := SaveResult{Collection: collectionName}
	planned := map[string]bool{}

	for _, ir := range requests {
		destDir := filepath.Join(requestsDir, dirName)
		if ir.FolderPath != "" {
			for _, seg := range strings.Split(ir.FolderPath, "/") {
				destDir = filepath.Join(destDir, SanitizeDirName(seg))
			}
		}

		yamlData, err := serialization.WriteRequest(ir.Request, "yaml")
		if err != nil {
			result.Skipped++
			continue
		}

		filename := SanitizeFilename(ir.Request.Title)
		if filename == "" {
			filename = "request"
		}
		savePath := freePath(filepath.Join(destDir, filename+".yaml"), planned)

		if !dryRun {
			if err := os.MkdirAll(destDir, 0o755); err != nil {
				result.Skipped++
				continue
			}
			if err := atomicWrite(savePath, yamlData); err != nil {
				result.Skipped++
				continue
			}
		}
		planned[savePath] = true
		result.Saved = append(result.Saved, Saved{
			Path: savePath, Title: ir.Request.Title, Method: string(ir.Request.Method), URL: ir.Request.URL,
		})
	}
	return result
}

// freePath returns path, or path with a -2, -3 … suffix when a file already exists there or an
// earlier request in the same import already took it.
func freePath(path string, planned map[string]bool) string {
	taken := func(p string) bool {
		if planned[p] {
			return true
		}
		_, err := os.Stat(p)
		return err == nil
	}
	if !taken(path) {
		return path
	}
	for i := 2; ; i++ {
		if candidate := numbered(path, i); !taken(candidate) {
			return candidate
		}
	}
}

// numbered is path with a number before its extension: users.yaml -> users-2.yaml.
func numbered(path string, i int) string {
	ext := filepath.Ext(path)
	return fmt.Sprintf("%s-%d%s", strings.TrimSuffix(path, ext), i, ext)
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
