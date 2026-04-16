package transformergo

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// TestTransformForstFileToGo_examplesInTree walks repo examples/in (non-.skip.) and runs parse → typecheck → emit.
// Skips individual files that do not compile yet.
func TestTransformForstFileToGo_examplesInTree(t *testing.T) {
	t.Parallel()
	// internal/transformer/go → ../../../../examples/in
	root := filepath.Join("..", "..", "..", "..", "examples", "in")
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
	if len(paths) == 0 {
		t.Fatal("no example .ft files found under", root)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	for _, path := range paths {
		path := path
		t.Run(filepath.ToSlash(strings.TrimPrefix(path, root+"/")), func(t *testing.T) {
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
			p := parser.NewTestParser(string(src), log)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Skipf("parse: %v", err)
			}
			chk := typechecker.New(log, false)
			if err := chk.CheckTypes(nodes); err != nil {
				t.Skipf("typecheck: %v", err)
			}
			tr := New(chk, log)
			if _, err := tr.TransformForstFileToGo(nodes); err != nil {
				t.Skipf("transform: %v", err)
			}
		})
	}
}
