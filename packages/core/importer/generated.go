package importer

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// FileState says what writing a generated file would do to what is on disk.
type FileState string

const (
	FileNew       FileState = "new"       // nothing is there yet
	FileUnchanged FileState = "unchanged" // the same text is there: nothing needs writing
	FileDiffers   FileState = "differs"   // something else is there: writing would lose it
)

// StateOf compares data with the file at path. Line endings alone do not make a difference: a
// checkout on Windows may have changed them.
func StateOf(path string, data []byte) (FileState, error) {
	have, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return FileNew, nil
	}
	if err != nil {
		return "", err
	}
	plain := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
	if bytes.Equal(plain(have), plain(data)) {
		return FileUnchanged, nil
	}
	return FileDiffers, nil
}

// UniqueName returns path, or path with -2, -3 … when an earlier file of the same run took it, so
// that one generated file never replaces another. It does not look at the disk.
func UniqueName(path string, taken map[string]bool) string {
	name := path
	for i := 2; taken[name]; i++ {
		name = numbered(path, i)
	}
	taken[name] = true
	return name
}

// WriteGenerated writes a file atomically, making its folder if needed.
func WriteGenerated(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWrite(path, data)
}
