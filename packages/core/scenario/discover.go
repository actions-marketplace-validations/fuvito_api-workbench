package scenario

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Discovery is the outcome of expanding command-line paths into scenario files.
type Discovery struct {
	// Files are the scenario files to run, de-duplicated and sorted by path.
	Files []string
	// Skipped are files found while scanning a folder that are not scenarios (for example
	// environment, request or spec files). They are reported, never run.
	Skipped []string
}

var scenarioExts = map[string]bool{".yaml": true, ".yml": true, ".json": true}

// Discover expands paths into scenario files. A file is taken as given. A folder contributes
// every .yaml, .yml or .json file below it that is a scenario, searching sub-folders when
// recursive is true; hidden folders and node_modules are skipped.
//
// A file found by scanning counts as a scenario when it has a top-level "steps" key. A file
// that cannot be parsed is treated as a scenario if it mentions "steps", so a broken scenario
// is reported as an error later instead of being skipped silently.
func Discover(paths []string, recursive bool) (Discovery, error) {
	var d Discovery
	var errs []error
	seen := map[string]bool{}
	seenSkipped := map[string]bool{}
	add := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if !seen[abs] {
			seen[abs] = true
			d.Files = append(d.Files, p)
		}
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: not found", p))
			continue
		}
		if !info.IsDir() {
			add(p)
			continue
		}

		found := 0
		walkErr := filepath.WalkDir(p, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if path == p {
					return nil
				}
				if name := entry.Name(); strings.HasPrefix(name, ".") || name == "node_modules" || !recursive {
					return fs.SkipDir
				}
				return nil
			}
			if !scenarioExts[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			if looksLikeScenario(path) {
				found++
				add(path)
			} else if !seenSkipped[path] {
				seenSkipped[path] = true
				d.Skipped = append(d.Skipped, path)
			}
			return nil
		})
		if walkErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, walkErr))
			continue
		}
		if found == 0 {
			errs = append(errs, fmt.Errorf("%s: no scenarios found", p))
		}
	}

	sort.Strings(d.Files)
	sort.Strings(d.Skipped)
	return d, errors.Join(errs...)
}

func looksLikeScenario(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true // let the caller report the read error
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return bytes.Contains(data, []byte("steps"))
	}
	_, ok := doc["steps"]
	return ok
}
