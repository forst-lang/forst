package transformergo

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// Full examples/in matrix lives in cmd/forst TestExamples.
var examplesTransformSmokeFiles = []string{
	"basic.ft",
	"ensure.ft",
	"nominal_error.ft",
	"rfc/guard/shape_guard.ft",
	"rfc/providers/providers.ft",
}

// TestTransformForstFileToGo_examplesSmoke runs parse → typecheck → emit on a small representative set.
func TestTransformForstFileToGo_examplesSmoke(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "..", "examples", "in")
	log := logrus.New()
	log.SetOutput(io.Discard)
	for _, rel := range examplesTransformSmokeFiles {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(root, filepath.FromSlash(rel))
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			p := parser.NewTestParser(string(src), log)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("parse %s: %v", rel, err)
			}
			chk := typechecker.New(log, false)
			if err := chk.CheckTypes(nodes); err != nil {
				t.Fatalf("typecheck %s: %v", rel, err)
			}
			tr := New(chk, log)
			if _, err := tr.TransformForstFileToGo(nodes); err != nil {
				t.Fatalf("transform %s: %v", rel, err)
			}
		})
	}
}
