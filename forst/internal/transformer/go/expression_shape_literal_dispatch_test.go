package transformergo

import (
	"go/token"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestTransformAssertionValue_referenceInPointerContextUsesAddressOf(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	var valueNode ast.ValueNode = ast.ReferenceNode{
		Value: ast.VariableNode{Ident: ast.Ident{ID: "item"}},
	}
	assertion := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &valueNode}},
		}},
	}
	expectedType := &ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}

	expr, err := transformer.transformAssertionValue(assertion, expectedType)
	if err != nil {
		t.Fatalf("transformAssertionValue: %v", err)
	}
	unaryExpr, ok := expr.(*goast.UnaryExpr)
	if !ok || unaryExpr.Op != token.AND {
		t.Fatalf("expected address-of unary expr, got %T (%#v)", expr, expr)
	}
}

func TestBuildTypeValue_pointerAndPrimitiveAliases(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	pointerValue, err := transformer.buildTypeValue(&ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeString}},
	})
	if err != nil {
		t.Fatalf("buildTypeValue(pointer): %v", err)
	}
	if identifier, ok := pointerValue.(*goast.Ident); !ok || identifier.Name != "nil" {
		t.Fatalf("expected nil for pointer type, got %#v", pointerValue)
	}

	intValue, err := transformer.buildTypeValue(&ast.TypeNode{Ident: ast.TypeInt})
	if err != nil {
		t.Fatalf("buildTypeValue(int): %v", err)
	}
	intLiteral, ok := intValue.(*goast.BasicLit)
	if !ok || intLiteral.Kind != token.INT || intLiteral.Value != "0" {
		t.Fatalf("expected int zero literal, got %#v", intValue)
	}
}

func TestDetermineStructType_prefersNamedExpectedTypeIncludingPointer(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	shape := &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}

	namedExpr, err := transformer.determineStructType(shape, &ast.TypeNode{
		Ident:    "User",
		TypeKind: ast.TypeKindBuiltin,
	})
	if err != nil {
		t.Fatalf("determineStructType(named): %v", err)
	}
	namedIdent, ok := namedExpr.(*goast.Ident)
	if !ok || namedIdent.Name != "User" {
		t.Fatalf("expected named ident User, got %#v", namedExpr)
	}

	pointerExpr, err := transformer.determineStructType(shape, &ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeKind:   ast.TypeKindBuiltin,
		TypeParams: []ast.TypeNode{{Ident: "AppContext", TypeKind: ast.TypeKindBuiltin}},
	})
	if err != nil {
		t.Fatalf("determineStructType(pointer): %v", err)
	}
	pointerIdent, ok := pointerExpr.(*goast.Ident)
	if !ok || pointerIdent.Name != "*AppContext" {
		t.Fatalf("expected pointer ident *AppContext, got %#v", pointerExpr)
	}
}
