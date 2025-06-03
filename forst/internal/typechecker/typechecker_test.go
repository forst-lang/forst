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
			name: "valid boolean return",
			typeGuard: &ast.TypeGuardNode{
				Ident: "IsValid",
				SubjectParam: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Value: ast.BoolLiteralNode{Value: true},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid int return",
			typeGuard: &ast.TypeGuardNode{
				Ident: "InvalidReturn",
				SubjectParam: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Value: ast.IntLiteralNode{Value: 42},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid string return",
			typeGuard: &ast.TypeGuardNode{
				Ident: "InvalidReturn",
				SubjectParam: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Value: ast.StringLiteralNode{Value: "invalid"},
					},
				},
			},
			expectError: true,
		},
		{
			name: "no return statement",
			typeGuard: &ast.TypeGuardNode{
				Ident: "NoReturn",
				SubjectParam: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{},
			},
			expectError: true,
		},
		{
			name: "valid boolean expression return",
			typeGuard: &ast.TypeGuardNode{
				Ident: "IsValid",
				SubjectParam: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Value: ast.BinaryExpressionNode{
							Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
							Operator: ast.TokenGreater,
							Right:    ast.IntLiteralNode{Value: 0},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := New()
			err := tc.CheckTypes([]ast.Node{tt.typeGuard})
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for invalid type guard return type, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid type guard return type: %v", err)
				}

				// Verify type guard is stored in global scope
				globalScope := tc.GlobalScope()
				symbol, exists := globalScope.Symbols[ast.Identifier(tt.typeGuard.Ident)]
				if !exists {
					t.Errorf("type guard %s not found in global scope", tt.typeGuard.Ident)
				}
				if symbol.Kind != SymbolFunction {
					t.Errorf("type guard %s stored with wrong kind, got %v want %v", tt.typeGuard.Ident, symbol.Kind, SymbolFunction)
				}
				if len(symbol.Types) != 1 || symbol.Types[0].Ident != ast.TypeBool {
					t.Errorf("type guard %s stored with wrong type, got %v want bool", tt.typeGuard.Ident, symbol.Types)
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
				baseType := ast.TypeString
				return ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "s"}},
					Operator: ast.TokenIs,
					Right: ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"field1": {
								Assertion: &ast.AssertionNode{
									BaseType: &baseType,
								},
							},
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
			tc := New()
			// Register 's' as a Shape type variable in the current scope
			tc.storeSymbol(ast.Identifier("s"), []ast.TypeNode{{Ident: ast.TypeShape}}, SymbolVariable)
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
