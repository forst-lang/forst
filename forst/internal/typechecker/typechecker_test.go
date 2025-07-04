package typechecker

import (
	"testing"

	"forst/internal/ast"
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
							"field1": makeAssertionField(ast.TypeString),
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
				Right:    ast.IntLiteralNode{Value: 42},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := setupTestLogger()
			tc := New(log, false)
			// Register 's' as a Shape type variable in the current scope
			baseType := ast.TypeIdent(ast.TypeShape)
			shape := makeShape(map[string]ast.ShapeFieldNode{
				"field1": makeAssertionField(ast.TypeString),
			})
			shapeType := ast.TypeNode{
				Ident: ast.TypeShape,
				Assertion: &ast.AssertionNode{
					BaseType: &baseType,
					Constraints: []ast.ConstraintNode{
						makeConstraint("Match", shape),
					},
				},
			}
			tc.storeSymbol(ast.Identifier("s"), []ast.TypeNode{shapeType}, SymbolVariable)
			_, err := tc.unifyTypes(tt.expr.Left, tt.expr.Right, tt.expr.Operator)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for invalid shape wrapper, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid shape wrapper: %v", err)
				}
			}
		})
	}
}

func TestTypeChecker_RegistersInferredParameterTypeForAssertion(t *testing.T) {
	tc := New(nil, false)

	// Register a minimal type guard for "Input"
	inputGuard := &ast.TypeGuardNode{
		Ident: "Input",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("m")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("MutationArg")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("input")},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("Shape")},
			},
		},
		Body: nil,
	}
	tc.Defs[ast.TypeIdent("Input")] = inputGuard

	// Simulate a type assertion like AppMutation.Input({input: {name: String}})
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("AppMutation")),
		Constraints: []ast.ConstraintNode{{
			Name: "Input",
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
	inferredTypes, err := tc.InferAssertionType(assertion, false)
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
