package astwalk

import (
	"testing"

	"forst/internal/ast"
)

func TestWalkStmtsContaining_collectsNestedWithChain(t *testing.T) {
	with := ast.WithNode{
		Span: ast.SourceSpan{StartLine: 2, StartCol: 1, EndLine: 4, EndCol: 2},
		Body: []ast.Node{
			ast.WithNode{
				Span: ast.SourceSpan{StartLine: 3, StartCol: 2, EndLine: 3, EndCol: 20},
			},
		},
	}
	fn := ast.FunctionNode{
		Body: []ast.Node{with},
	}
	var chain []ast.WithNode
	WalkStmtsContaining(fn.Body, 3, 5, StmtVisitor{
		OnWith: func(w ast.WithNode) bool {
			chain = append(chain, w)
			return true
		},
	})
	if len(chain) != 2 {
		t.Fatalf("chain len = %d, want 2 nested with", len(chain))
	}
}
