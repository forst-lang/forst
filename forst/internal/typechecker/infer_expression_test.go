package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFunctionCallWithShapeLiteralArgument(t *testing.T) {
	// Create a typechecker
	tc := New(logrus.New(), false)

	// Create type definitions for the types we need
	appMutationDef := ast.TypeDefNode{
		Ident: "AppMutation",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}

	// Use CheckTypes to register the type definitions properly
	err := tc.CheckTypes([]ast.Node{appMutationDef})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Add a dummy function definition for createMutation that accepts AppMutation
	createMutationFn := ast.FunctionNode{
		Ident: ast.Ident{ID: "createMutation"},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("AppMutation")},
			},
		},
		ReturnTypes: []ast.TypeNode{},
		Body:        []ast.Node{},
	}

	// Create a shape literal argument matching AppMutation
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}

	// Create a function call with the shape literal as an argument
	functionCall := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "createMutation"},
		Arguments: []ast.ExpressionNode{shapeLiteral},
	}

	// Wrap the function call in a function node
	testFn := ast.FunctionNode{
		Ident: ast.Ident{ID: "test"},
		Body:  []ast.Node{functionCall},
	}

	// Use CheckTypes to register the type definitions and infer the shape literal type
	err = tc.CheckTypes([]ast.Node{appMutationDef, createMutationFn, testFn})
	if err != nil {
		t.Fatalf("Failed to register type definitions and infer types: %v", err)
	}

	// Optionally, check that the argument is compatible with AppMutation
	appMutationType := ast.TypeNode{Ident: ast.TypeIdent("AppMutation")}
	argTypes, err := tc.inferExpressionType(shapeLiteral)
	if err != nil {
		t.Fatalf("Failed to infer type for shape literal: %v", err)
	}
	compatible := tc.IsTypeCompatible(argTypes[0], appMutationType)
	if !compatible {
		t.Errorf("Expected shape literal to be compatible with AppMutation, got %v", argTypes[0])
	}
}
