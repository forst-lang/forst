package hasher

import (
	"testing"

	"forst/internal/ast"
)

// Exercises HashNode branches for typedef and auxiliary AST kinds used by navigation and lowering.
func TestStructuralHasher_HashNode_typeDefExprsAndCommonKinds(t *testing.T) {
	t.Parallel()
	h := New()
	base := ast.TypeIdent("U")
	left := ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}
	right := ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}
	typeDefBin := ast.TypeDefBinaryExpr{Left: left, Op: ast.TokenBitwiseOr, Right: right}
	typeDefAssert := ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &base}}
	typeDefErr := ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}

	cases := []struct {
		name string
		node ast.Node
	}{
		{"TypeDefBinaryExpr", typeDefBin},
		{"TypeDefAssertionExpr", typeDefAssert},
		{"TypeDefAssertionExpr_ptr", &typeDefAssert},
		{"TypeDefShapeExpr", left},
		{"TypeDefErrorExpr", typeDefErr},
		{"NilLiteral", ast.NilLiteralNode{}},
		{"Comment", ast.CommentNode{Text: "note"}},
		{"Dereference", ast.DereferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}}},
		{"OkExpr", ast.OkExprNode{Value: ast.IntLiteralNode{Value: 1}}},
		{"ErrExpr", ast.ErrExprNode{Value: ast.StringLiteralNode{Value: "e"}}},
		{"ArrayLiteral", ast.ArrayLiteralNode{Type: ast.TypeNode{Ident: ast.TypeInt}, Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}}},
		{"IndexExpression", ast.IndexExpressionNode{Target: ast.VariableNode{Ident: ast.Ident{ID: "a"}}, Index: ast.IntLiteralNode{Value: 0}}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.HashNode(tc.node)
			if err != nil {
				t.Fatalf("HashNode(%s): %v", tc.name, err)
			}
		})
	}
}
