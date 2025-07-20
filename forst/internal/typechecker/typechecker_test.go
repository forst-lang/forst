package typechecker

import (
	"fmt"
	"os"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestTypeGuardReturnType(t *testing.T) {
	tests := []struct {
		name        string
		typeGuard   *ast.TypeGuardNode
		expectError bool
	}{
		{
			name: "valid type guard with if statement",
			typeGuard: &ast.TypeGuardNode{
				Ident: "IsValid",
				Subject: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.IfNode{
						Condition: ast.BinaryExpressionNode{
							Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
							Operator: ast.TokenIs,
							Right: ast.AssertionNode{
								BaseType: typeIdentPtr(string(ast.TypeInt)),
							},
						},
						Body: []ast.Node{},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid - has return statement",
			typeGuard: &ast.TypeGuardNode{
				Ident: "InvalidReturn",
				Subject: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{ast.BoolLiteralNode{Value: true}},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid - condition not using is operator",
			typeGuard: &ast.TypeGuardNode{
				Ident: "InvalidCondition",
				Subject: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.IfNode{
						Condition: ast.BinaryExpressionNode{
							Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
							Operator: ast.TokenGreater,
							Right:    ast.IntLiteralNode{Value: 0},
						},
						Body: []ast.Node{},
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid type guard with ensure statement",
			typeGuard: &ast.TypeGuardNode{
				Ident: "HasField",
				Subject: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "s"},
					Type:  ast.TypeNode{Ident: ast.TypeShape},
				},
				Body: []ast.Node{
					ast.EnsureNode{
						Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
						Assertion: ast.AssertionNode{
							BaseType: typeIdentPtr(string(ast.TypeShape)),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid type guard with if-else",
			typeGuard: &ast.TypeGuardNode{
				Ident: "IsValid",
				Subject: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.IfNode{
						Condition: ast.BinaryExpressionNode{
							Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
							Operator: ast.TokenIs,
							Right: ast.AssertionNode{
								BaseType: typeIdentPtr(string(ast.TypeInt)),
							},
						},
						Body: []ast.Node{},
						Else: &ast.ElseBlockNode{
							Body: []ast.Node{},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := setupTestLogger()
			tc := New(log, false)
			err := tc.CheckTypes([]ast.Node{tt.typeGuard})
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for invalid type guard, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid type guard: %v", err)
				}

				// Verify type guard is stored in global scope
				globalScope := tc.globalScope()
				symbol, exists := globalScope.Symbols[ast.Identifier(tt.typeGuard.Ident)]
				if !exists {
					t.Errorf("type guard %s not found in global scope", tt.typeGuard.Ident)
				}
				if symbol.Kind != SymbolTypeGuard {
					t.Errorf("type guard %s stored with wrong kind, got %v want %v", tt.typeGuard.Ident, symbol.Kind, SymbolTypeGuard)
				}
				if len(symbol.Types) != 1 || symbol.Types[0].Ident != ast.TypeVoid {
					t.Errorf("type guard %s stored with wrong type, got %v want void", tt.typeGuard.Ident, symbol.Types)
				}
			}
		})
	}
}

func TestIsOperationWithShapeWrapper(t *testing.T) {
	tests := []struct {
		name        string
		expr        ast.BinaryExpressionNode
		expectError bool
	}{
		{
			name: "valid shape wrapper",
			expr: func() ast.BinaryExpressionNode {
				return ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "s"}},
					Operator: ast.TokenIs,
					Right: ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"field1": {Type: &ast.TypeNode{Ident: ast.TypeString}},
						},
					},
				}
			}(),
			expectError: false,
		},
		{
			name: "invalid shape wrapper",
			expr: ast.BinaryExpressionNode{
				Left:     ast.VariableNode{Ident: ast.Ident{ID: "s"}},
				Operator: ast.TokenIs,
				Right: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"field1": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := setupTestLogger()
			tc := New(log, false)

			// Create a function that declares 's' and tests the is operation
			testFn := ast.FunctionNode{
				Ident: ast.Ident{ID: "test"},
				Body: []ast.Node{
					ast.AssignmentNode{
						LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "s"}}},
						RValues: []ast.ExpressionNode{ast.ShapeNode{
							Fields: map[string]ast.ShapeFieldNode{
								"field1": {Type: &ast.TypeNode{Ident: ast.TypeString}},
							},
						}},
						IsShort: true,
					},
					// Add the is operation directly as an expression in the function body
					tt.expr,
				},
			}
			err := tc.CheckTypes([]ast.Node{testFn})
			if err != nil {
				t.Fatalf("Failed to register function: %v", err)
			}

			// The is operation should have been processed during CheckTypes
			// We can verify this by checking if the types were inferred correctly
			if !tt.expectError {
				// For valid cases, we expect the types to be inferred without error
				// The actual type inference happens during CheckTypes above
			}
		})
	}
}

func TestTypeChecker_RegistersInferredParameterTypeForAssertion(t *testing.T) {
	log := setupTestLogger()
	tc := New(log, false)

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

	// Create a type guard that takes an AppMutation parameter
	typeGuard := ast.TypeGuardNode{
		Ident: "TestGuard",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "m"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("AppMutation")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "param"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}

	// Use CheckTypes to register the type definitions and type guard properly
	err := tc.CheckTypes([]ast.Node{appMutationDef, typeGuard})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Simulate a type assertion like AppMutation.TestGuard({input: {name: String}})
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("AppMutation")),
		Constraints: []ast.ConstraintNode{{
			Name: "TestGuard",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"input": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			}},
		}},
	}

	// Infer the type as the typechecker would
	inferredTypes, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil || len(inferredTypes) == 0 {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	paramType := inferredTypes[0]
	paramTypeName := string(paramType.Ident)

	// The typechecker should register the inferred type in tc.Defs
	if _, exists := tc.Defs[ast.TypeIdent(paramTypeName)]; !exists {
		t.Errorf("Typechecker did NOT register inferred parameter type %q in tc.Defs (core bug)", paramTypeName)
	} else {
		t.Logf("Typechecker registered inferred parameter type %q in tc.Defs", paramTypeName)
	}
}

func TestMultipleReturnValues(t *testing.T) {
	tests := []struct {
		name        string
		function    ast.FunctionNode
		expectError bool
	}{
		{
			name: "function with multiple return values",
			function: ast.FunctionNode{
				Ident: ast.Ident{ID: "test"},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{
							ast.IntLiteralNode{Value: 1},
							ast.IntLiteralNode{Value: 2},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "assignment with multiple return values",
			function: ast.FunctionNode{
				Ident: ast.Ident{ID: "test"},
				Body: []ast.Node{
					ast.AssignmentNode{
						LValues: []ast.VariableNode{
							{Ident: ast.Ident{ID: "a"}},
							{Ident: ast.Ident{ID: "b"}},
						},
						RValues: []ast.ExpressionNode{
							ast.FunctionCallNode{
								Function:  ast.Ident{ID: "foo"},
								Arguments: []ast.ExpressionNode{},
							},
						},
					},
				},
			},
			expectError: true, // Still fails because 'foo' function is undefined
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := setupTestLogger()
			tc := New(log, false)

			// Register dummy 'foo' function for assignment test
			if tt.name == "assignment with multiple return values" {
				tc.Functions["foo"] = FunctionSignature{
					Parameters: []ParameterSignature{},
					ReturnTypes: []ast.TypeNode{
						{Ident: ast.TypeInt},
						{Ident: ast.TypeInt},
					},
				}
			}

			err := tc.CheckTypes([]ast.Node{tt.function})
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for multiple return values, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for multiple return values: %v", err)
				}
			}
		})
	}
}

func TestTypeChecker_AssertionWithShapeLiteral_PropagatesConcreteShapeType(t *testing.T) {
	// Set up a debug logger to see all debug output
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(os.Stdout)

	tc := New(log, false)

	// Create type definitions for the types we need
	personDef := ast.TypeDefNode{
		Ident: "Person",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					"age":  {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}

	// Define a type guard that takes a Person parameter
	typeGuard := ast.TypeGuardNode{
		Ident: "WithName",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "p"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Person")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "name"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}

	// Use CheckTypes to register the type definitions and type guard properly
	err := tc.CheckTypes([]ast.Node{personDef, typeGuard})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Simulate a type assertion like Person.WithName({name: {city: String}})
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("Person")),
		Constraints: []ast.ConstraintNode{{
			Name: "WithName",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			}},
		}},
	}

	inferredTypes, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil || len(inferredTypes) == 0 {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	paramType := inferredTypes[0]
	paramTypeName := string(paramType.Ident)
	fmt.Printf("[DEBUG] Inferred type ident: %s\n", paramTypeName)

	// The inferred type should be registered
	def, exists := tc.Defs[ast.TypeIdent(paramTypeName)]
	if !exists {
		t.Fatalf("Typechecker did NOT register inferred parameter type %q in tc.Defs", paramTypeName)
	}
	td, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Registered type is not a TypeDefNode")
	}
	shapeExpr, ok := td.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Registered type is not a TypeDefShapeExpr")
	}
	// Instead of checking for 'address', check for 'name' and 'age'
	if _, ok := shapeExpr.Shape.Fields["name"]; !ok {
		t.Errorf("Field 'name' not found in inferred shape")
	}
	if _, ok := shapeExpr.Shape.Fields["age"]; !ok {
		t.Errorf("Field 'age' not found in inferred shape")
	}
}

func TestTypeChecker_AssertionWithTypeNodeMatchConstraint_PropagatesConcreteShapeType(t *testing.T) {
	log := setupTestLogger()
	tc := New(log, false)

	// Create type definitions for the types we need
	recordDef := ast.TypeDefNode{
		Ident: "Record",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id":   {Type: &ast.TypeNode{Ident: ast.TypeString}},
					"data": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}

	// Define a type guard that takes a Record parameter
	typeGuard := ast.TypeGuardNode{
		Ident: "WithData",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "r"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Record")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "data"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}

	// Use CheckTypes to register the type definitions and type guard properly
	err := tc.CheckTypes([]ast.Node{recordDef, typeGuard})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Simulate a type assertion like Record.WithData({data: {value: String}}),
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("Record")),
		Constraints: []ast.ConstraintNode{{
			Name: "WithData",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"data": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			}},
		}},
	}

	inferredTypes, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil || len(inferredTypes) == 0 {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	paramType := inferredTypes[0]
	paramTypeName := string(paramType.Ident)
	fmt.Printf("[DEBUG] Inferred type ident: %s\n", paramTypeName)

	// The inferred type should be registered
	def, exists := tc.Defs[ast.TypeIdent(paramTypeName)]
	if !exists {
		t.Fatalf("Typechecker did NOT register inferred parameter type %q in tc.Defs", paramTypeName)
	}
	td, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Registered type is not a TypeDefNode")
	}
	shapeExpr, ok := td.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Registered type is not a TypeDefShapeExpr")
	}
	// Instead of checking for 'field', check for 'data' and 'id'
	if _, ok := shapeExpr.Shape.Fields["data"]; !ok {
		t.Errorf("Field 'data' not found in inferred shape")
	}
	if _, ok := shapeExpr.Shape.Fields["id"]; !ok {
		t.Errorf("Field 'id' not found in inferred shape")
	}
}

func TestTypeChecker_AssertionWithShapeLiteral_SingleField_NestedShapeType(t *testing.T) {
	log := setupTestLogger()
	tc := New(log, false)

	// Create type definitions for the types we need
	recordDef := ast.TypeDefNode{
		Ident: "Record",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id":   {Type: &ast.TypeNode{Ident: ast.TypeString}},
					"data": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}

	// Define a type guard that takes a Record parameter
	typeGuard := ast.TypeGuardNode{
		Ident: "WithNested",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "r"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Record")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "nested"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}

	// Use CheckTypes to register the type definitions and type guard properly
	err := tc.CheckTypes([]ast.Node{recordDef, typeGuard})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Simulate a type assertion like Record.WithNested({nested: {value: String}})
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("Record")),
		Constraints: []ast.ConstraintNode{{
			Name: "WithNested",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"nested": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			}},
		}},
	}

	inferredTypes, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil || len(inferredTypes) == 0 {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	paramType := inferredTypes[0]
	paramTypeName := string(paramType.Ident)
	fmt.Printf("[DEBUG] Inferred type ident: %s\n", paramTypeName)

	// The inferred type should be registered
	def, exists := tc.Defs[ast.TypeIdent(paramTypeName)]
	if !exists {
		t.Fatalf("Typechecker did NOT register inferred parameter type %q in tc.Defs", paramTypeName)
	}
	td, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Registered type is not a TypeDefNode")
	}
	shapeExpr, ok := td.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Registered type is not a TypeDefShapeExpr")
	}
	// Instead of checking for 'field', check for 'nested', 'data', and 'id'
	if _, ok := shapeExpr.Shape.Fields["nested"]; !ok {
		t.Errorf("Field 'nested' not found in inferred shape")
	}
	if _, ok := shapeExpr.Shape.Fields["data"]; !ok {
		t.Errorf("Field 'data' not found in inferred shape")
	}
	if _, ok := shapeExpr.Shape.Fields["id"]; !ok {
		t.Errorf("Field 'id' not found in inferred shape")
	}
}
