package typechecker

import (
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
	log.SetOutput(nil)
	merged, _, err := forstpkg.ParseAndMergePackage(log, []string{typesPath, apiPath})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	tc := New(log, false)
	if err := tc.CheckTypes(merged); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestIsTypeCompatible_simpleTypeAliasToString(t *testing.T) {
	log := logrus.New()
	log.SetOutput(nil)
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
