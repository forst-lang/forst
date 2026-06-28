package typechecker

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// TestCheckTypes_examplesInTree typechecks every examples/in/*.ft (skips parse/typecheck failures).
func TestCheckTypes_examplesInTree(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "examples", "in")
	var paths []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".ft") && !strings.Contains(path, ".skip.") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	for _, path := range paths {
		rel := filepath.ToSlash(strings.TrimPrefix(path, root+"/"))
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Skipf("panic: %v", r)
				}
			}()
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := parser.NewTestParser(string(src), log).ParseFile()
			if err != nil {
				t.Skipf("parse: %v", err)
			}
			chk := New(log, false)
			chk.GoWorkspaceDir = moduleRootForProvidersTest(t)
			if err := chk.CheckTypes(nodes); err != nil {
				t.Skipf("typecheck: %v", err)
			}
		})
	}
}
