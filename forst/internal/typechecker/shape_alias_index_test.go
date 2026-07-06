package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestShapeAliasIndex_resolvesStructuralHash(t *testing.T) {
	t.Parallel()
	tc := testTypeChecker(t)
	userShape := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
		"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
	}}
	h, err := tc.Hasher.HashNode(userShape)
	if err != nil {
		t.Fatal(err)
	}
	hashIdent := h.ToTypeIdent()
	tc.registerType(ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	})
	tc.Defs[hashIdent] = ast.TypeDefNode{
		Ident: hashIdent,
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}
	alias, ok := tc.lookupShapeAliasForHashType(ast.TypeNode{Ident: hashIdent, TypeKind: ast.TypeKindHashBased})
	if !ok || alias != "User" {
		t.Fatalf("lookupShapeAliasForHashType(%s) = %q, %v; want User, true", hashIdent, alias, ok)
	}
	resolved := tc.resolveAliasedType(ast.TypeNode{Ident: hashIdent, TypeKind: ast.TypeKindHashBased})
	if resolved.Ident != "User" {
		t.Fatalf("resolveAliasedType(%s) = %s, want User", hashIdent, resolved.Ident)
	}
}

func TestShapeAliasIndex_stableTieBreakForDuplicateShapeHash(t *testing.T) {
	t.Parallel()
	tc := testTypeChecker(t)
	shape := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
	}}
	tc.registerType(ast.TypeDefNode{Ident: "Zebra", Expr: ast.TypeDefShapeExpr{Shape: shape}})
	tc.registerType(ast.TypeDefNode{Ident: "Alpha", Expr: ast.TypeDefShapeExpr{Shape: shape}})
	idx := tc.shapeAliasIndexOrBuild()
	if len(idx.byShapeHash) != 1 {
		t.Fatalf("expected one shape hash entry, got %d", len(idx.byShapeHash))
	}
	for _, alias := range idx.byShapeHash {
		if alias != "Alpha" {
			t.Fatalf("stable winner = %q, want Alpha", alias)
		}
	}
}

func TestShapeAliasIndex_resolvesAssertionRefinementHash(t *testing.T) {
	t.Parallel()
	tc := testTypeChecker(t)
	tc.registerType(ast.TypeDefNode{
		Ident: "LoggedIn",
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: ptrTypeIdent("User")},
		},
	})
	bt := ast.TypeIdent("LoggedIn")
	h, err := tc.Hasher.HashNode(ast.AssertionNode{BaseType: &bt})
	if err != nil {
		t.Fatal(err)
	}
	hashIdent := h.ToTypeIdent()
	alias, ok := tc.lookupAssertionAliasForHashIdent(hashIdent)
	if !ok || alias != "LoggedIn" {
		t.Fatalf("lookupAssertionAliasForHashIdent(%s) = %q, %v; want LoggedIn, true", hashIdent, alias, ok)
	}
}

func ptrTypeIdent(id ast.TypeIdent) *ast.TypeIdent {
	return &id
}
