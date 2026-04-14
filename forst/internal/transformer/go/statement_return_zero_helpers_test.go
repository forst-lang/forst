package transformergo

import (
	goast "go/ast"
	"testing"

	"forst/internal/ast"
)

func TestGetZeroValue_commonBranches(t *testing.T) {
	if lit, ok := getZeroValue(goast.NewIdent("string")).(*goast.BasicLit); !ok || lit.Value != "\"\"" {
		t.Fatalf("string zero value mismatch: %#v", getZeroValue(goast.NewIdent("string")))
	}
	if ident, ok := getZeroValue(goast.NewIdent("bool")).(*goast.Ident); !ok || ident.Name != "false" {
		t.Fatalf("bool zero value mismatch: %#v", getZeroValue(goast.NewIdent("bool")))
	}
	if ident, ok := getZeroValue(goast.NewIdent("error")).(*goast.Ident); !ok || ident.Name != "nil" {
		t.Fatalf("error zero value mismatch: %#v", getZeroValue(goast.NewIdent("error")))
	}
	if ident, ok := getZeroValue(&goast.StarExpr{X: goast.NewIdent("T")}).(*goast.Ident); !ok || ident.Name != "nil" {
		t.Fatalf("pointer zero value mismatch: %#v", getZeroValue(&goast.StarExpr{X: goast.NewIdent("T")}))
	}
}

func TestGetShapeNode_extractsValueAndPointer(t *testing.T) {
	valueShape := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"a": {Type: &ast.TypeNode{Ident: ast.TypeString}}}}
	ptrShape := &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"b": {Type: &ast.TypeNode{Ident: ast.TypeInt}}}}

	got1, ok1 := getShapeNode(valueShape)
	if !ok1 || got1 == nil || got1.Fields["a"].Type == nil {
		t.Fatalf("failed to extract value shape: ok=%v got=%+v", ok1, got1)
	}

	got2, ok2 := getShapeNode(ptrShape)
	if !ok2 || got2 == nil || got2.Fields["b"].Type == nil {
		t.Fatalf("failed to extract pointer shape: ok=%v got=%+v", ok2, got2)
	}

	if _, ok := getShapeNode(ast.IntLiteralNode{Value: 1}); ok {
		t.Fatal("unexpected shape extraction for non-shape node")
	}
}

func TestZeroValueExprForASTType_builtinAndNamedShape(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	expr, err := tr.zeroValueExprForASTType(ast.TypeNode{Ident: ast.TypeString})
	if err != nil {
		t.Fatalf("builtin zero value: %v", err)
	}
	if lit, ok := expr.(*goast.BasicLit); !ok || lit.Value != "\"\"" {
		t.Fatalf("expected string basic literal zero value, got %#v", expr)
	}

	tc.Defs["MyShape"] = ast.TypeDefNode{
		Ident: "MyShape",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"name": {Type: &ast.TypeNode{Ident: ast.TypeString}}}},
		},
	}
	expr, err = tr.zeroValueExprForASTType(ast.TypeNode{Ident: "MyShape", TypeKind: ast.TypeKindUserDefined})
	if err != nil {
		t.Fatalf("named shape zero value: %v", err)
	}
	if _, ok := expr.(*goast.CompositeLit); !ok {
		t.Fatalf("expected composite literal for named shape zero value, got %T", expr)
	}
}
