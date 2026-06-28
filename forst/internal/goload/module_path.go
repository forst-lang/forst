package goload

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ModulePath reads the module path from go.mod in moduleRoot, or "" if unavailable.
func ModulePath(moduleRoot string) string {
	path := filepath.Join(moduleRoot, "go.mod")
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
