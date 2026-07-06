package hasher

import (
	"testing"

	"forst/internal/ast"
)

func TestNodeIdentityKey_sameInterfaceSlot(t *testing.T) {
	t.Parallel()
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	var n ast.Node = fn
	k1, ok1 := NodeIdentityKey(n)
	k2, ok2 := NodeIdentityKey(n)
	if !ok1 || !ok2 || k1 != k2 {
		t.Fatalf("identity keys: %+v %v %+v %v", k1, ok1, k2, ok2)
	}
}

func TestNodeIdentityKey_distinctLiteralTypes(t *testing.T) {
	t.Parallel()
	var a ast.Node = ast.BoolLiteralNode{Value: true}
	var b ast.Node = ast.IntLiteralNode{Value: 1}
	ka, okA := NodeIdentityKey(a)
	kb, okB := NodeIdentityKey(b)
	if !okA || !okB || ka == kb {
		t.Fatalf("bool and int literals must not share identity: %+v vs %+v", ka, kb)
	}
}

func TestHashNode_memoizesSameNodePointer(t *testing.T) {
	t.Parallel()
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "benchFn"},
		Body: []ast.Node{
			ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}},
		},
	}
	var node ast.Node = fn
	h := New()
	first, err := h.HashNode(node)
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.HashNode(node)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("hash mismatch: %x vs %x", first, second)
	}
	if len(h.nodeHashCache) < 1 {
		t.Fatalf("expected cache entries, got %d", len(h.nodeHashCache))
	}
}

func TestHashNode_structurallyIdenticalDistinctNodes(t *testing.T) {
	t.Parallel()
	a := ast.IntLiteralNode{Value: 7}
	b := ast.IntLiteralNode{Value: 7}
	h := New()
	ha, err := h.HashNode(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := h.HashNode(b)
	if err != nil {
		t.Fatal(err)
	}
	if ha != hb {
		t.Fatalf("structural hash mismatch for identical literals: %x vs %x", ha, hb)
	}
}
