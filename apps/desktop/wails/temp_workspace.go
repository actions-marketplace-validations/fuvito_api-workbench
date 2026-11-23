package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The demo and conformance servers each open a temporary workspace. Several copies of the app can run
// at once, so cleanup must not delete a folder another copy has open. Every running copy keeps a small
// owner file in its temporary workspace fresh; a folder is stale only when nobody has refreshed it.
const (
	tempOwnerFile       = ".clrsig-owner"
	tempOwnerBeat       = 30 * time.Second
	tempOwnerStaleAfter = 3 * time.Minute
	// A folder from a version that wrote no owner file is judged by its own modification time.
	tempLegacyStaleAfter = time.Hour
)

// claimTempWorkspace marks dir as in use by this copy of the app until the folder is removed.
func claimTempWorkspace(dir string) {
	path := filepath.Join(dir, tempOwnerFile)
	touch := func() bool {
		if _, err := os.Stat(dir); err != nil {
			return false
		}
		return os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0o600) == nil
	}
	if !touch() {
		return
	}
	go func() {
		for {
			time.Sleep(tempOwnerBeat)
			if !touch() {
				return
			}
		}
	}()
}

// isStaleTempWorkspace reports whether nothing is keeping dir alive.
func isStaleTempWorkspace(dir string, now time.Time) bool {
	if info, err := os.Stat(filepath.Join(dir, tempOwnerFile)); err == nil {
		return now.Sub(info.ModTime()) > tempOwnerStaleAfter
	}
	info, err := os.Stat(dir)
	return err == nil && now.Sub(info.ModTime()) > tempLegacyStaleAfter
}

// removeStaleTempWorkspaces deletes the temporary workspaces under base whose name starts with one of
// prefixes and that no running copy of the app keeps alive. skip is never touched.
func removeStaleTempWorkspaces(base, skip string, prefixes ...string) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	now := time.Now()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		matches := false
		for _, p := range prefixes {
			matches = matches || strings.HasPrefix(e.Name(), p)
		}
		full := filepath.Join(base, e.Name())
		if !matches || full == skip || !isStaleTempWorkspace(full, now) {
			continue
		}
		os.RemoveAll(full) //nolint:errcheck
	}
}
