package transformergo

import (
	"testing"

	"forst/internal/ast"
)

func TestEmitReferencedTypesHelpers_andTransformBlock(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	tc.Defs["Inner"] = ast.TypeDefNode{
		Ident: "Inner",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	tc.Defs["Outer"] = ast.TypeDefNode{
		Ident: "Outer",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"inner": {Type: &ast.TypeNode{Ident: "Inner"}},
				},
			},
		},
	}
	processed := map[ast.TypeIdent]bool{}
	if err := tr.emitReferencedTypes(tc.Defs["Outer"].(ast.TypeDefNode), processed); err != nil {
		t.Fatalf("emitReferencedTypes: %v", err)
	}
	assertion := &ast.AssertionNode{
		BaseType: ptrTypeIdent("Inner"),
	}
	if err := tr.emitReferencedTypesFromAssertion(assertion, processed); err != nil {
		t.Fatalf("emitReferencedTypesFromAssertion: %v", err)
	}

	blk := tr.transformBlock([]ast.Node{
		ast.CommentNode{Text: "x"},
		ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}},
	})
	if blk == nil || len(blk.List) == 0 {
		t.Fatal("expected transformed block statements")
	}
}

func ptrTypeIdent(id ast.TypeIdent) *ast.TypeIdent {
	v := id
	return &v
}

