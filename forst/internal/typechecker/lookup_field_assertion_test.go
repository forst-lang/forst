package typechecker

import (
	"strings"
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

func TestLookupFieldPathOnMergedFields_nestedPath(t *testing.T) {
	tc := New(logger.New(), false)
	fields := map[string]ast.ShapeFieldNode{
		"user": {
			Shape: &ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	got, err := tc.lookupFieldPathOnMergedFields(fields, []string{"user", "name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeString {
		t.Fatalf("got %q", got.Ident)
	}
}

func TestLookupFieldPathOnMergedFields_missingField(t *testing.T) {
	tc := New(logger.New(), false)
	_, err := tc.lookupFieldPathOnMergedFields(
		map[string]ast.ShapeFieldNode{"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}}},
		[]string{"missing"},
	)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("got %v", err)
	}
}
