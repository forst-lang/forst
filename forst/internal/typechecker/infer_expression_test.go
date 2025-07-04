package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFunctionCallWithShapeLiteralArgument(t *testing.T) {
	// Create a typechecker
	tc := New(logrus.New(), false)
	tc.VariableTypes = map[ast.Identifier][]ast.TypeNode{
		"sessionId": {{Ident: ast.TypeString}},
	}

	// Create a shape literal that will be passed as a function argument
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {
				Assertion: &ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{
						Name: "Value",
						Args: []ast.ConstraintArgumentNode{{
							Value: func() *ast.ValueNode {
								var v ast.ValueNode = ast.ReferenceNode{
									Value: ast.VariableNode{
										Ident: ast.Ident{ID: ast.Identifier("sessionId")},
									},
								}
								return &v
							}(),
						}},
					}},
				},
			},
		},
	}

	// Create a function call with the shape literal as an argument
	functionCall := ast.FunctionCallNode{
		Function: ast.Ident{ID: ast.Identifier("createUser")},
		Arguments: []ast.ExpressionNode{
			shapeLiteral,
		},
	}

	// Register a function signature for createUser
	tc.Functions[ast.Identifier("createUser")] = FunctionSignature{
		Parameters: []ParameterSignature{
			{
				Ident: ast.Ident{ID: ast.Identifier("op")},
				Type: ast.TypeNode{
					Ident: ast.TypeIdent("AppMutation"),
				},
			},
		},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
	}

	// Test that the shape literal argument is properly type-checked
	// This should infer the type for the shape literal and store it
	_, err := tc.inferExpressionType(functionCall)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the shape literal has an inferred type stored
	shapeHash, err := tc.Hasher.HashNode(shapeLiteral)
	if err != nil {
		t.Fatalf("Failed to hash shape literal: %v", err)
	}
	shapeTypeIdent := shapeHash.ToTypeIdent()

	// Check if the shape type was registered
	if _, exists := tc.Defs[shapeTypeIdent]; !exists {
		t.Errorf("Expected shape type %s to be registered in Defs", shapeTypeIdent)
	}

	// Check if the inferred type is stored for the shape literal
	if inferredTypes, exists := tc.Types[shapeHash]; !exists {
		t.Errorf("Expected inferred types to be stored for shape literal")
	} else if len(inferredTypes) == 0 {
		t.Errorf("Expected non-empty inferred types for shape literal")
	} else if inferredTypes[0].Ident != shapeTypeIdent {
		t.Errorf("Expected inferred type %s, got %s", shapeTypeIdent, inferredTypes[0].Ident)
	}
}
