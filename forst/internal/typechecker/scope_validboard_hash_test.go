package typechecker

import (
	"io"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/hasher"

	"github.com/sirupsen/logrus"
)

func validBoardFirstEnsure(t *testing.T, merged []ast.Node) ast.EnsureNode {
	t.Helper()
	for _, node := range merged {
		switch g := node.(type) {
		case ast.TypeGuardNode:
			if g.Ident == ast.Identifier("ValidBoard") {
				return g.Body[0].(ast.EnsureNode)
			}
		case *ast.TypeGuardNode:
			if g.Ident == ast.Identifier("ValidBoard") {
				return g.Body[0].(ast.EnsureNode)
			}
		}
	}
	t.Fatal("ValidBoard ensure not found in merged AST")
	return ast.EnsureNode{}
}

func TestValidBoardEnsureHash_stableBeforeAndAfterCollect(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	log := logrus.New()
	log.SetOutput(io.Discard)
	merged, _, err := forstpkg.ParseAndMergePackage(log, []string{filepath.Join(root, "main", "engine.ft")})
	if err != nil {
		t.Fatal(err)
	}
	ensure := validBoardFirstEnsure(t, merged)

	h := hasher.New()
	for i := 0; i < 200; i++ {
		h1, err := h.HashNode(ensure)
		if err != nil {
			t.Fatal(err)
		}
		h2, err := h.HashNode(ensure)
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Fatalf("fresh hasher iter %d unstable: %x vs %x", i, h1, h2)
		}
	}

	tc := New(log, false)
	if err := tc.CollectTypes(merged); err != nil {
		t.Fatal(err)
	}
	stored := tc.Defs[ast.TypeIdent("ValidBoard")].(*ast.TypeGuardNode)
	ensureStored := stored.Body[0].(ast.EnsureNode)
	for i := 0; i < 200; i++ {
		h1, err := tc.Hasher.HashNode(ensureStored)
		if err != nil {
			t.Fatal(err)
		}
		h2, err := tc.Hasher.HashNode(ensureStored)
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Fatalf("after collect iter %d unstable: %x vs %x", i, h1, h2)
		}
	}
}

func TestValidBoardEnsureScopes_afterCollectOnly(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	log := logrus.New()
	log.SetOutput(io.Discard)

	for iter := 0; iter < 500; iter++ {
		merged, _, err := forstpkg.ParseAndMergePackage(log, []string{filepath.Join(root, "main", "engine.ft")})
		if err != nil {
			t.Fatal(err)
		}
		tc := New(log, false)
		if err := tc.CollectTypes(merged); err != nil {
			t.Fatalf("iter %d collect: %v", iter, err)
		}
		def := tc.Defs[ast.TypeIdent("ValidBoard")].(*ast.TypeGuardNode)
		for _, bodyNode := range def.Body {
			ensure, ok := bodyNode.(ast.EnsureNode)
			if !ok {
				continue
			}
			hash, err := tc.Hasher.HashNode(ensure)
			if err != nil {
				t.Fatal(err)
			}
			hash2, err := tc.Hasher.HashNode(ensure)
			if err != nil {
				t.Fatal(err)
			}
			if hash != hash2 {
				t.Fatalf("iter %d unstable ensure hash: %x vs %x", iter, hash, hash2)
			}
			if _, ok := tc.scopeStack.scopes[hash]; !ok {
				t.Fatalf("iter %d after collect only: scope missing for %s hash=%x", iter, ensure, hash)
			}
		}
	}
}

func TestValidBoardEnsureScopes_afterFullCheck(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	log := logrus.New()
	log.SetOutput(io.Discard)

	for iter := 0; iter < 500; iter++ {
		merged, _, err := forstpkg.ParseAndMergePackage(log, []string{filepath.Join(root, "main", "engine.ft")})
		if err != nil {
			t.Fatal(err)
		}
		tc := New(log, false)
		if err := tc.CheckTypes(merged); err != nil {
			t.Fatalf("iter %d check: %v", iter, err)
		}
		def := tc.Defs[ast.TypeIdent("ValidBoard")].(*ast.TypeGuardNode)
		for _, bodyNode := range def.Body {
			ensure, ok := bodyNode.(ast.EnsureNode)
			if !ok {
				continue
			}
			hash, err := tc.Hasher.HashNode(ensure)
			if err != nil {
				t.Fatal(err)
			}
			hash2, err := tc.Hasher.HashNode(ensure)
			if err != nil {
				t.Fatal(err)
			}
			if hash != hash2 {
				t.Fatalf("iter %d unstable ensure hash: %x vs %x", iter, hash, hash2)
			}
			if _, ok := tc.scopeStack.scopes[hash]; !ok {
				t.Fatalf("iter %d after full check: scope missing for %s hash=%x", iter, ensure, hash)
			}
		}
	}
}
