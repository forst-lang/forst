// Package testmod provides helpers for writing test go.mod fixtures at a consistent Go version.
package testmod

import (
	"fmt"
	"os"
	"path/filepath"
)

// GoVersion matches forst/go.mod.
const GoVersion = "1.26.0"

// GoModContent returns a minimal go.mod body for module.
func GoModContent(module string) string {
	return fmt.Sprintf("module %s\n\ngo %s\n", module, GoVersion)
}

type testFatal interface {
	Helper()
	Fatal(...interface{})
}

// WriteGoMod writes go.mod into dir for module.
func WriteGoMod(t testFatal, dir, module string) {
	t.Helper()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(GoModContent(module)), 0o644); err != nil {
		t.Fatal(err)
	}
}
