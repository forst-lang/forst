package hasher

import (
	"sync"
	"testing"

	"forst/internal/ast"
)

// concurrentHashNodes mirrors the diverse node set from structural_hashnode_smoke_test.go.
func concurrentHashNodes() []ast.Node {
	base := ast.TypeIdent("U")
	left := ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}
	right := ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}
	typeDefBin := ast.TypeDefBinaryExpr{Left: left, Op: ast.TokenBitwiseOr, Right: right}
	typeDefAssert := ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &base}}
	typeDefErr := ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}

	return []ast.Node{
		typeDefBin,
		typeDefAssert,
		&typeDefAssert,
		left,
		typeDefErr,
		ast.NilLiteralNode{},
		ast.CommentNode{Text: "note"},
		ast.DereferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}},
		ast.OkExprNode{Value: ast.IntLiteralNode{Value: 1}},
		ast.ErrExprNode{Value: ast.StringLiteralNode{Value: "e"}},
		ast.ArrayLiteralNode{Type: ast.TypeNode{Ident: ast.TypeInt}, Value: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}},
		ast.IndexExpressionNode{Target: ast.VariableNode{Ident: ast.Ident{ID: "a"}}, Index: ast.IntLiteralNode{Value: 0}},
		ast.ImportNode{Path: "fmt"},
		ast.MapLiteralNode{Type: ast.TypeNode{Ident: ast.TypeIdent("String")}, Entries: []ast.MapEntryNode{{Key: ast.StringLiteralNode{Value: "k"}, Value: ast.IntLiteralNode{Value: 1}}}},
		ast.FunctionNode{
			Ident: ast.Ident{ID: "f"},
			Body: []ast.Node{
				ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}},
			},
		},
	}
}

// TestHashNode_concurrentSharedHasher guards against regressions where HashNode shares
// mutable walk state across goroutines (the old global nodeHashCache panicked under -race).
func TestHashNode_concurrentSharedHasher(t *testing.T) {
	t.Parallel()

	nodes := concurrentHashNodes()
	h := New()

	expected := make([]NodeHash, len(nodes))
	for i, node := range nodes {
		hash, err := h.HashNode(node)
		if err != nil {
			t.Fatalf("baseline HashNode(%T): %v", node, err)
		}
		expected[i] = hash
	}

	const goroutines = 32
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for iter := 0; iter < iterations; iter++ {
				i := (g + iter) % len(nodes)
				got, err := h.HashNode(nodes[i])
				if err != nil {
					t.Errorf("HashNode(%T): %v", nodes[i], err)
					return
				}
				if got != expected[i] {
					t.Errorf("HashNode(%T): got %x want %x", nodes[i], got, expected[i])
					return
				}
			}
		}(g)
	}
	wg.Wait()
}
