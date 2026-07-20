package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func TestTransformPackageVarDecl_untypedLiteral(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	decl, err := tr.transformPackageVarDecl(ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "Version"}}},
		RValues:        []ast.ExpressionNode{ast.StringLiteralNode{Value: "1.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decl.Tok != token.VAR {
		t.Fatalf("tok = %v", decl.Tok)
	}
	spec := decl.Specs[0].(*goast.ValueSpec)
	if spec.Names[0].Name != "Version" {
		t.Fatalf("name = %q", spec.Names[0].Name)
	}
	if lit, ok := spec.Values[0].(*goast.BasicLit); !ok || lit.Value != `"1.0"` {
		t.Fatalf("value = %#v", spec.Values[0])
	}
}

func TestTransformPackageVarDecl_explicitType(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	intType := ast.TypeNode{Ident: ast.TypeInt}
	decl, err := tr.transformPackageVarDecl(ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "Count"}}},
		RValues:        []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		ExplicitTypes:  []*ast.TypeNode{&intType},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec := decl.Specs[0].(*goast.ValueSpec)
	if spec.Type == nil {
		t.Fatal("expected explicit type on package var")
	}
}

func TestTransformPackageVarDecl_rejectsMultiName(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	_, err := tr.transformPackageVarDecl(ast.AssignmentNode{
		IsPackageLevel: true,
		LValues: []ast.ExpressionNode{
			ast.VariableNode{Ident: ast.Ident{ID: "A"}},
			ast.VariableNode{Ident: ast.Ident{ID: "B"}},
		},
		RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}, ast.IntLiteralNode{Value: 2}},
	})
	if err == nil || !strings.Contains(err.Error(), "one name and one value") {
		t.Fatalf("got %v", err)
	}
}

func TestTransformPackageVarDecl_rejectsNonIdentifierLHS(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	_, err := tr.transformPackageVarDecl(ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		RValues:        []ast.ExpressionNode{ast.IntLiteralNode{Value: 2}},
	})
	if err == nil || !strings.Contains(err.Error(), "simple identifier") {
		t.Fatalf("got %v", err)
	}
}
