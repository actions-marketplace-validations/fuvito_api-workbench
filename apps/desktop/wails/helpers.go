package main

import (
	"os"
	"path/filepath"
	"strings"
)

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// outsideRel reports whether a path made relative with filepath.Rel points outside its base: it is
// ".." or starts with "../". A name that merely begins with two dots (such as "..notes") is inside.
func outsideRel(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// pathInside reports whether p is root or inside it, after resolving symbolic links on both, so a link
// inside the workspace that points elsewhere counts as outside. A path that does not exist yet is
// judged by its cleaned absolute form.
func pathInside(root, p string) bool {
	resolve := func(x string) string {
		if abs, err := filepath.Abs(x); err == nil {
			x = abs
		}
		if real, err := filepath.EvalSymlinks(x); err == nil {
			return real
		}
		return x
	}
	rel, err := filepath.Rel(resolve(root), resolve(p))
	return err == nil && !outsideRel(rel)
}
