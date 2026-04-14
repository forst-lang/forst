package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/logger"
)

func TestLookupFieldInAssertion_noAssertion(t *testing.T) {
	tc := &TypeChecker{log: logger.New()}
	_, err := tc.lookupFieldInAssertion(
		ast.TypeNode{},
		ast.Ident{ID: ast.Identifier("x")},
		nil,
	)
	if err == nil {
		t.Fatal("expected no assertion found error")
	}
}

func TestLookupFieldPathOnMergedFields_shapeSingleSegmentReturnsShapeType(t *testing.T) {
	tc := &TypeChecker{log: logger.New()}
	fields := map[string]ast.ShapeFieldNode{
		"profile": {
			Shape: &ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	got, err := tc.lookupFieldPathOnMergedFields(fields, []string{"profile"})
	if err != nil {
		t.Fatalf("lookupFieldPathOnMergedFields: %v", err)
	}
	if got.Ident != ast.TypeShape {
		t.Fatalf("expected TYPE_SHAPE, got %q", got.Ident)
	}
}

func TestLookupFieldPathOnMergedFields_emptyPath(t *testing.T) {
	tc := &TypeChecker{log: logger.New()}
	_, err := tc.lookupFieldPathOnMergedFields(map[string]ast.ShapeFieldNode{}, []string{})
	if err == nil {
		t.Fatal("expected empty field path error")
	}
}
