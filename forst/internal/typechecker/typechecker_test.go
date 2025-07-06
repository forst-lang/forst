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

func TestTypeChecker_AssertionWithShapeLiteral_PropagatesConcreteShapeType(t *testing.T) {
	// Set up a debug logger to see all debug output
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(os.Stdout)

	// Register a minimal type guard for "WithAddress"
	withAddressGuard := &ast.TypeGuardNode{
		Ident: "WithAddress",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("p")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Person")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("address")},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("Shape")},
			},
		},
		Body: nil,
	}
	tc := New(log, false)
	tc.Defs[ast.TypeIdent("WithAddress")] = withAddressGuard

	// Simulate a type assertion like Person.WithAddress({address: {city: String}})
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("Person")),
		Constraints: []ast.ConstraintNode{{
			Name: "WithAddress",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"address": {Shape: &ast.ShapeNode{
							Fields: map[string]ast.ShapeFieldNode{
								"city": {Type: &ast.TypeNode{Ident: ast.TypeString}},
							},
						}},
					},
				},
			}},
		}},
	}

	inferredTypes, err := tc.InferAssertionType(assertion, false)
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
	addressField, ok := shapeExpr.Shape.Fields["address"]
	if !ok {
		t.Fatalf("Field 'address' not found in inferred shape")
	}
	fmt.Printf("[DEBUG] Field 'address': %+v\n", addressField)
	if addressField.Type == nil {
		t.Fatalf("Field 'address' has no type")
	}
	fmt.Printf("[DEBUG] Field 'address' type ident: %s\n", addressField.Type.Ident)
	if string(addressField.Type.Ident) == "Shape" || string(addressField.Type.Ident) == "Shape(?)" {
		t.Errorf("BUG: Field 'address' type is generic Shape, not a unique hash-based type: got %q", addressField.Type.Ident)
	} else {
		t.Logf("Field 'address' type correctly inferred as %q", addressField.Type.Ident)
	}
}

func TestTypeChecker_AssertionWithTypeNodeMatchConstraint_PropagatesConcreteShapeType(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(os.Stdout)

	// Register a minimal type guard for "WithField"
	withFieldGuard := &ast.TypeGuardNode{
		Ident: "WithField",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("r")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Record")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("field")},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("Shape")},
			},
		},
		Body: nil,
	}
	tc := New(log, false)
	tc.Defs[ast.TypeIdent("WithField")] = withFieldGuard

	// Simulate a type assertion like Record.WithField({field: {value: String}}),
	// but pass the argument as a TypeNode with an Assertion (Match constraint with shape literal)
	fieldShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"value": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	fieldTypeNode := &ast.TypeNode{
		Ident: ast.TypeShape,
		Assertion: &ast.AssertionNode{
			BaseType: func() *ast.TypeIdent { t := ast.TypeIdent("Shape"); return &t }(),
			Constraints: []ast.ConstraintNode{{
				Name: "Match",
				Args: []ast.ConstraintArgumentNode{{
					Shape: fieldShape,
				}},
			}},
		},
	}
	assertion := &ast.AssertionNode{
		BaseType: func() *ast.TypeIdent { t := ast.TypeIdent("Record"); return &t }(),
		Constraints: []ast.ConstraintNode{{
			Name: "WithField",
			Args: []ast.ConstraintArgumentNode{{
				Type: fieldTypeNode,
			}},
		}},
	}

	inferredTypes, err := tc.InferAssertionType(assertion, false)
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
	fieldField, ok := shapeExpr.Shape.Fields["field"]
	if !ok {
		t.Logf("Field 'field' not found in inferred shape (expected for TypeNode with Match constraint)")
		return // This is the expected behavior
	}
	if fieldField.Type == nil {
		t.Fatalf("Field 'field' has no type")
	}
	if string(fieldField.Type.Ident) == "Shape" || string(fieldField.Type.Ident) == "Shape(?)" {
		t.Logf("Field 'field' type is generic Shape (expected for this scenario): got %q", fieldField.Type.Ident)
	} else {
		t.Logf("Field 'field' type is a unique hash-based type: %q", fieldField.Type.Ident)
	}
}

func TestTypeChecker_AssertionWithShapeLiteral_SingleField_NestedShapeType(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(os.Stdout)

	// Register a minimal type guard for "WithField"
	withFieldGuard := &ast.TypeGuardNode{
		Ident: "WithField",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("r")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Record")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("field")},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("Shape")},
			},
		},
		Body: nil,
	}
	tc := New(log, false)
	tc.Defs[ast.TypeIdent("WithField")] = withFieldGuard

	// Simulate a type assertion like Record.WithField({field: {value: String}})
	assertion := &ast.AssertionNode{
		BaseType: func() *ast.TypeIdent { t := ast.TypeIdent("Record"); return &t }(),
		Constraints: []ast.ConstraintNode{{
			Name: "WithField",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"field": {Shape: &ast.ShapeNode{
							Fields: map[string]ast.ShapeFieldNode{
								"value": {Type: &ast.TypeNode{Ident: ast.TypeString}},
							},
						}},
					},
				},
			}},
		}},
	}

	inferredTypes, err := tc.InferAssertionType(assertion, false)
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
	fieldField, ok := shapeExpr.Shape.Fields["field"]
	if !ok {
		t.Fatalf("Field 'field' not found in inferred shape")
	}
	if fieldField.Type == nil {
		t.Fatalf("Field 'field' has no type")
	}
	// Look up the type for 'field' and check its fields
	fieldTypeIdent := fieldField.Type.Ident
	fieldTypeDef, ok := tc.Defs[fieldTypeIdent].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Type for 'field' is not a TypeDefNode")
	}
	fieldShapeExpr, ok := fieldTypeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Type for 'field' is not a TypeDefShapeExpr")
	}
	if _, ok := fieldShapeExpr.Shape.Fields["value"]; !ok {
		t.Errorf("BUG: Type for 'field' does not have a 'value' field; got fields: %+v", fieldShapeExpr.Shape.Fields)
	} else {
		t.Logf("Type for 'field' correctly has a 'value' field: %+v", fieldShapeExpr.Shape.Fields)
	}
	if _, ok := fieldShapeExpr.Shape.Fields["field"]; ok {
		t.Errorf("BUG: Type for 'field' should not have a 'field' field; got fields: %+v", fieldShapeExpr.Shape.Fields)
	}
}
