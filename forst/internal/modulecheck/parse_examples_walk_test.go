package modulecheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/forstpkg"
)

func TestParseExamplesIn_noPanicsOnWalk(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full examples/in walk in -short mode")
	}
	root := filepath.Join("..", "..", "..", "examples", "in")
	var panics []string
	_ = walkExampleFt(root, root, func(path string) {
		defer func() {
			if r := recover(); r != nil {
				panics = append(panics, path+": "+strings.TrimSpace(fmtRecover(r)))
			}
		}()
		_, err := forstpkg.ParseForstFile(nil, path)
		if err != nil {
			return
		}
	})
	if len(panics) > 0 {
		t.Fatalf("parse panics:\n%s", strings.Join(panics, "\n"))
	}
}

func walkExampleFt(root, dir string, visit func(path string)) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root {
				if _, statErr := os.Stat(filepath.Join(path, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(path, ".ft") {
			visit(path)
		}
		return nil
	})
}

func fmtRecover(r any) string {
	if s, ok := r.(string); ok {
		return s
	}
	if e, ok := r.(error); ok {
		return e.Error()
	}
	return "unknown panic"
}
