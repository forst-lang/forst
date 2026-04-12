package typechecker

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"

	"github.com/sirupsen/logrus"
)

func TestCheckTypes_mergedPackage_aliasReturnCompatibleWithStringLiteral(t *testing.T) {
	dir := t.TempDir()
	typesPath := filepath.Join(dir, "types.ft")
	apiPath := filepath.Join(dir, "api.ft")
	if err := os.WriteFile(typesPath, []byte(`package demo

type Greeting = String
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiPath, []byte(`package demo

func Hello(): Greeting {
	return "hi"
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	merged, _, err := forstpkg.ParseAndMergePackage(log, []string{typesPath, apiPath})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	tc := New(log, false)
	if err := tc.CheckTypes(merged); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

// Regression: inferNodeType must not replace `type Greeting = String` with an empty TypeDefShapeExpr.
// underlyingBuiltinTypeOfAliasAssertion(String) returns "" because String is not in Defs as a typedef.
func TestInferTypeDef_directBuiltinAliasPreservesAssertionExpr(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	str := ast.TypeString
	td := ast.TypeDefNode{
		Ident: "Greeting",
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &str},
		},
	}
	tc.registerType(td)
	if _, err := tc.inferNodeType(td); err != nil {
		t.Fatalf("infer: %v", err)
	}
	def, ok := tc.Defs[ast.TypeIdent("Greeting")].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("expected TypeDefNode for Greeting")
	}
	if _, ok := def.Expr.(ast.TypeDefAssertionExpr); !ok {
		t.Fatalf("expected TypeDefAssertionExpr after infer, got %T", def.Expr)
	}
}

func TestIsTypeCompatible_simpleTypeAliasToString(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	tc.Defs[ast.TypeIdent("Greeting")] = ast.TypeDefNode{
		Ident: ast.TypeIdent("Greeting"),
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: ptrTypeIdent(ast.TypeString),
			},
		},
	}
	if !tc.IsTypeCompatible(ast.TypeNode{Ident: ast.TypeString}, ast.TypeNode{Ident: ast.TypeIdent("Greeting")}) {
		t.Fatal("expected String compatible with Greeting alias")
	}
	if !tc.IsTypeCompatible(ast.TypeNode{Ident: ast.TypeIdent("Greeting")}, ast.TypeNode{Ident: ast.TypeString}) {
		t.Fatal("expected Greeting alias compatible with String")
	}
}

func ptrTypeIdent(id ast.TypeIdent) *ast.TypeIdent {
	return &id
}
