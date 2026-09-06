package forstdep

import (
	"os"
	"path/filepath"
	"strings"
)

// GenGoStubOverlay builds a go/packages Overlay that empties committed *.gen.go
// files so they do not collide with symbols inferred from .ft sources.
func GenGoStubOverlay(dir, pkgName string) (map[string][]byte, error) {
	if dir == "" || pkgName == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	stub := []byte("package " + pkgName + "\n")
	out := make(map[string][]byte)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".gen.go") {
			continue
		}
		abs := filepath.Join(dir, name)
		out[abs] = stub
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
