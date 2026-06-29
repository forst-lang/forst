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

func TestGetZeroValue_additionalBranches(t *testing.T) {
	t.Parallel()
	if lit, ok := getZeroValue(goast.NewIdent("int")).(*goast.BasicLit); !ok || lit.Value != "0" {
		t.Fatalf("int zero: %#v", getZeroValue(goast.NewIdent("int")))
	}
	if ident, ok := getZeroValue(&goast.ArrayType{Elt: goast.NewIdent("int")}).(*goast.Ident); !ok || ident.Name != "nil" {
		t.Fatalf("array zero: %#v", getZeroValue(&goast.ArrayType{Elt: goast.NewIdent("int")}))
	}
	if ident, ok := getZeroValue(&goast.InterfaceType{}).(*goast.Ident); !ok || ident.Name != "nil" {
		t.Fatalf("interface zero: %#v", getZeroValue(&goast.InterfaceType{}))
	}
	if ident, ok := getZeroValue(goast.NewIdent("CustomType")).(*goast.Ident); !ok || ident.Name != "nil" {
		t.Fatalf("custom ident zero: %#v", getZeroValue(goast.NewIdent("CustomType")))
	}
}

func TestBuildCompositeLiteralForReturn_multiField(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Pair"] = ast.TypeDefNode{
		Ident: "Pair",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"a": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					"b": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	lit := tr.buildCompositeLiteralForReturn(&ast.TypeNode{Ident: "Pair"}, "v")
	comp, ok := lit.(*goast.CompositeLit)
	if !ok || len(comp.Elts) != 2 {
		t.Fatalf("expected 2-field composite, got %#v", lit)
	}
}

func TestWrapVariableInNamedStruct_mapsVariableToField(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Wrap"] = ast.TypeDefNode{
		Ident: "Wrap",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"msg": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					"n":   {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	expr, err := tr.wrapVariableInNamedStruct(
		&ast.TypeNode{Ident: "Wrap"},
		ast.VariableNode{Ident: ast.Ident{ID: "msg"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*goast.CompositeLit); !ok {
		t.Fatalf("expected composite literal, got %T", expr)
	}
}

func TestBuildZeroCompositeLiteral_nestedUserDefinedField(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Inner"] = ast.TypeDefNode{
		Ident: "Inner",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	tc.Defs["Outer"] = ast.TypeDefNode{
		Ident: "Outer",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"inner": {Type: &ast.TypeNode{Ident: "Inner", TypeKind: ast.TypeKindUserDefined}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	lit := tr.buildZeroCompositeLiteral(&ast.TypeNode{Ident: "Outer", TypeKind: ast.TypeKindUserDefined})
	if _, ok := lit.(*goast.CompositeLit); !ok {
		t.Fatalf("expected composite literal, got %T", lit)
	}
}
