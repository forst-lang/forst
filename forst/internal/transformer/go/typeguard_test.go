package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

// typeIdentPtr creates a pointer to a TypeIdent
func typeIdentPtr(s string) *ast.TypeIdent {
	ti := ast.TypeIdent(s)
	return &ti
}

func TestTransformEnsureConstraintWithIsConstraint(t *testing.T) {
	// Test the core issue: the "is" constraint is not implemented
	// This reproduces the issue from the shape_guard.ft example

	// Create a simple ensure node with an "is" constraint
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier("m")},
		},
		Assertion: ast.AssertionNode{
			BaseType: typeIdentPtr("MutationArg"),
			Constraints: []ast.ConstraintNode{
				ast.ConstraintNode{
					Name: "is",
					Args: []ast.ConstraintArgumentNode{
						{
							Shape: &ast.ShapeNode{
								Fields: map[string]ast.ShapeFieldNode{
									"ctx": {
										Type: &ast.TypeNode{Ident: ast.TypeIdent("Shape")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a minimal type checker
	tc := typechecker.New(nil, false)

	// Create a transformer
	transformer := New(tc, nil)

	// Try to transform the ensure constraint
	// This should fail because "is" is not a built-in constraint
	_, err := transformer.transformEnsureConstraint(ensure, ensure.Assertion.Constraints[0], ast.TypeNode{Ident: ast.TypeIdent("MutationArg")})

	// This should fail with "no valid transformation found for constraint: is"
	if err == nil {
		t.Fatal("Expected error 'no valid transformation found for constraint: is', but got nil")
	}

	expectedError := "no valid transformation found for constraint: is"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestTransformEnsureConstraintWithValidConstraint(t *testing.T) {
	// Test that valid constraints work correctly

	// Create a simple ensure node with a valid constraint
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier("ctx")},
		},
		Assertion: ast.AssertionNode{
			BaseType: typeIdentPtr("TYPE_POINTER"),
			Constraints: []ast.ConstraintNode{
				ast.ConstraintNode{
					Name: "NotNil",
					Args: []ast.ConstraintArgumentNode{},
				},
			},
		},
	}

	// Create a minimal type checker
	tc := typechecker.New(nil, false)

	// Create a transformer
	transformer := New(tc, nil)

	// Try to transform the ensure constraint
	// This should succeed because NotNil is a valid built-in constraint for Pointer types
	_, err := transformer.transformEnsureConstraint(ensure, ensure.Assertion.Constraints[0], ast.TypeNode{Ident: ast.TypePointer})

	// This should succeed because NotNil is a valid built-in constraint
	if err != nil {
		t.Errorf("Expected no error, but got: %s", err.Error())
	}
}
