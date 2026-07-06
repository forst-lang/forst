package typechecker

import (
	"io"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"

	"github.com/sirupsen/logrus"
)

func TestRestoreScope_tictactoeValidBoardEnsures(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	paths := []string{
		filepath.Join(root, "engine.ft"),
		filepath.Join(root, "server.ft"),
	}
	log := logrus.New()
	log.SetOutput(io.Discard)

	for i := 0; i < 200; i++ {
		merged, _, err := forstpkg.ParseAndMergePackage(log, paths)
		if err != nil {
			t.Fatal(err)
		}
		tc := New(log, false)
		if err := tc.CheckTypes(merged); err != nil {
			t.Fatalf("iteration %d CheckTypes: %v", i, err)
		}
		for _, node := range merged {
			guard, ok := node.(ast.TypeGuardNode)
			if !ok || guard.Ident != "ValidBoard" {
				continue
			}
			for _, bodyNode := range guard.Body {
				ensure, ok := bodyNode.(ast.EnsureNode)
				if !ok {
					continue
				}
				if err := tc.RestoreScope(ensure); err != nil {
					hash, _ := tc.Hasher.HashNode(ensure)
					_, inScopes := tc.scopeStack.scopes[hash]
					t.Fatalf("iteration %d RestoreScope %s: %v (hash=%x inScopes=%v)", i, ensure, err, hash, inScopes)
				}
			}
		}
	}
}
