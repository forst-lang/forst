package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestBuildFieldsForExpectedType_preservesShapeFieldOrder(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs[ast.TypeIdent("Token")] = ast.TypeDefNode{
		Ident: "Token",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id":        {Type: &ast.TypeNode{Ident: ast.TypeString}},
					"expiresAt": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	shape := &ast.ShapeNode{
		FieldOrder: []string{"id", "expiresAt"},
		Fields: map[string]ast.ShapeFieldNode{
			"id":        ast.MakeStructField(ast.StringLiteralNode{Value: "t1"}),
			"expiresAt": ast.MakeStructField(ast.IntLiteralNode{Value: 500}),
		},
	}
	fields, err := tr.buildFieldsForExpectedType(shape, &ast.TypeNode{Ident: "Token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("len %d", len(fields))
	}
	if fields[0].Key.(*goast.Ident).Name != "id" || fields[1].Key.(*goast.Ident).Name != "expiresAt" {
		t.Fatalf("field order: %s, %s", fields[0].Key, fields[1].Key)
	}
}

func TestBuildFieldsForShape_intLiteralField(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"n": ast.MakeStructField(ast.IntLiteralNode{Value: 99}),
		},
	}
	fields, err := tr.buildFieldsForShape(shape)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 {
		t.Fatalf("len %d", len(fields))
	}
	if fields[0].Key.(*goast.Ident).Name != "n" {
		t.Fatalf("key %v", fields[0].Key)
	}
	// Value should be integer literal 99 in Go AST
	s := goExprString(t, fields[0].Value)
	if !strings.Contains(s, "99") {
		t.Fatalf("value: %s", s)
	}
}

func TestFindBestNamedTypeForStructLiteral_prefersExpectedNamedType(t *testing.T) {
	t.Parallel()
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	exp := ast.TypeNode{Ident: ast.TypeString, TypeKind: ast.TypeKindBuiltin}
	hash := ast.TypeNode{Ident: "T_xyz", TypeKind: ast.TypeKindHashBased}
	got := tr.findBestNamedTypeForStructLiteral(hash, &exp)
	if got.Ident != exp.Ident {
		t.Fatalf("got %v want %v", got.Ident, exp.Ident)
	}
}

func TestFindBestNamedTypeForStructLiteral_returnsNamedInferred(t *testing.T) {
	t.Parallel()
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	named := ast.TypeNode{Ident: "User", TypeKind: ast.TypeKindBuiltin}
	got := tr.findBestNamedTypeForStructLiteral(named, nil)
	if got.Ident != named.Ident {
		t.Fatalf("got %v", got.Ident)
	}
}
