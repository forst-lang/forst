package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// TDD: RFC 02 nominal errors assign to built-in Error; ordinary shape types do not.
func TestIsTypeCompatible_errorNominalToBuiltinError(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "NotPositive",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"field": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	if !tc.IsTypeCompatible(ast.TypeNode{Ident: "NotPositive"}, ast.TypeNode{Ident: ast.TypeError}) {
		t.Fatal("want NotPositive <: Error for nominal error type")
	}
}

func TestCheckTypes_errorNominalDeclaration_registersTypeDefErrorExpr(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

error NotPositive {
	field: String
}

func main() {
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	def, ok := tc.Defs["NotPositive"].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Defs[NotPositive]: missing")
	}
	if _, ok := def.Expr.(ast.TypeDefErrorExpr); !ok {
		t.Fatalf("want TypeDefErrorExpr, got %T", def.Expr)
	}
}

func TestIsTypeCompatible_plainShapeTypeNotImplicitError(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "Person",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	if tc.IsTypeCompatible(ast.TypeNode{Ident: "Person"}, ast.TypeNode{Ident: ast.TypeError}) {
		t.Fatal("non-error nominal shape must not be assignable to Error")
	}
}
