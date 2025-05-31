package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	"go/token"
	"strings"
	"testing"
)

func newValueNode(v ast.ValueNode) *ast.ValueNode {
	return &v
}

func TestAssertionTransformer(t *testing.T) {
	tc := typechecker.New()
	transformer := New(tc)
	at := NewAssertionTransformer(transformer)

	tests := []struct {
		name        string
		ensure      ast.EnsureNode
		baseType    ast.TypeNode
		wantErr     bool
		errContains string
		validate    func(t *testing.T, expr goast.Expr)
	}{
		{
			name: "string min length",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "test"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: MinConstraint,
							Args: []ast.ConstraintArgumentNode{
								{Value: newValueNode(ast.IntLiteralNode{Value: 5, Type: ast.TypeNode{Ident: ast.TypeInt}})},
							},
						},
					},
				},
			},
			baseType: ast.TypeNode{Ident: ast.TypeString},
			validate: func(t *testing.T, expr goast.Expr) {
				binExpr, ok := expr.(*goast.BinaryExpr)
				if !ok {
					t.Errorf("expected BinaryExpr, got %T", expr)
					return
				}
				if binExpr.Op != token.LSS {
					t.Errorf("expected LSS operator, got %v", binExpr.Op)
				}
				callExpr, ok := binExpr.X.(*goast.CallExpr)
				if !ok {
					t.Errorf("expected CallExpr for len(), got %T", binExpr.X)
					return
				}
				if len(callExpr.Args) != 1 {
					t.Errorf("expected 1 argument to len(), got %d", len(callExpr.Args))
				}
			},
		},
		{
			name: "int less than",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "speed"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: LessThanConstraint,
							Args: []ast.ConstraintArgumentNode{
								{Value: newValueNode(ast.IntLiteralNode{Value: 20, Type: ast.TypeNode{Ident: ast.TypeInt}})},
							},
						},
					},
				},
			},
			baseType: ast.TypeNode{Ident: ast.TypeInt},
			validate: func(t *testing.T, expr goast.Expr) {
				binExpr, ok := expr.(*goast.BinaryExpr)
				if !ok {
					t.Errorf("expected BinaryExpr, got %T", expr)
					return
				}
				if binExpr.Op != token.GEQ {
					t.Errorf("expected GEQ operator, got %v", binExpr.Op)
				}
			},
		},
		{
			name: "float greater than",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "price"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: GreaterThanConstraint,
							Args: []ast.ConstraintArgumentNode{
								{Value: newValueNode(ast.FloatLiteralNode{Value: 5.0, Type: ast.TypeNode{Ident: ast.TypeFloat}})},
							},
						},
					},
				},
			},
			baseType: ast.TypeNode{Ident: ast.TypeFloat},
			validate: func(t *testing.T, expr goast.Expr) {
				binExpr, ok := expr.(*goast.BinaryExpr)
				if !ok {
					t.Errorf("expected BinaryExpr, got %T", expr)
					return
				}
				if binExpr.Op != token.LEQ {
					t.Errorf("expected LEQ operator, got %v", binExpr.Op)
				}
			},
		},
		{
			name: "bool true",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "isValid"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: TrueConstraint,
						},
					},
				},
			},
			baseType: ast.TypeNode{Ident: ast.TypeBool},
			validate: func(t *testing.T, expr goast.Expr) {
				unaryExpr, ok := expr.(*goast.UnaryExpr)
				if !ok {
					t.Errorf("expected UnaryExpr, got %T", expr)
					return
				}
				if unaryExpr.Op != token.NOT {
					t.Errorf("expected NOT operator, got %v", unaryExpr.Op)
				}
				ident, ok := unaryExpr.X.(*goast.Ident)
				if !ok {
					t.Errorf("expected Ident for variable, got %T", unaryExpr.X)
					return
				}
				if ident.Name != "isValid" {
					t.Errorf("expected variable name 'isValid', got %s", ident.Name)
				}
			},
		},
		{
			name: "bool false",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "isInvalid"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: FalseConstraint,
						},
					},
				},
			},
			baseType: ast.TypeNode{Ident: ast.TypeBool},
			validate: func(t *testing.T, expr goast.Expr) {
				ident, ok := expr.(*goast.Ident)
				if !ok {
					t.Errorf("expected Ident, got %T", expr)
					return
				}
				if ident.Name != "isInvalid" {
					t.Errorf("expected variable name 'isInvalid', got %s", ident.Name)
				}
			},
		},
		{
			name: "error nil",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "err"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: NilConstraint,
						},
					},
				},
			},
			baseType: ast.TypeNode{Ident: ast.TypeError},
			validate: func(t *testing.T, expr goast.Expr) {
				binExpr, ok := expr.(*goast.BinaryExpr)
				if !ok {
					t.Errorf("expected BinaryExpr, got %T", expr)
					return
				}
				if binExpr.Op != token.NEQ {
					t.Errorf("expected NEQ operator, got %v", binExpr.Op)
				}
				ident, ok := binExpr.Y.(*goast.Ident)
				if !ok {
					t.Errorf("expected Ident for nil, got %T", binExpr.Y)
					return
				}
				if ident.Name != NilConstant {
					t.Errorf("expected %s, got %s", NilConstant, ident.Name)
				}
			},
		},
		{
			name: "invalid constraint",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "value"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: "InvalidConstraint",
						},
					},
				},
			},
			baseType:    ast.TypeNode{Ident: ast.TypeInt},
			wantErr:     true,
			errContains: "unknown Int constraint",
		},
		{
			name: "invalid argument type",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "value"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: LessThanConstraint,
							Args: []ast.ConstraintArgumentNode{
								{Value: newValueNode(ast.StringLiteralNode{Value: "20", Type: ast.TypeNode{Ident: ast.TypeString}})},
							},
						},
					},
				},
			},
			baseType:    ast.TypeNode{Ident: ast.TypeInt},
			wantErr:     true,
			errContains: "expected value to be a number literal",
		},
		{
			name: "missing argument",
			ensure: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "value"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: LessThanConstraint,
						},
					},
				},
			},
			baseType:    ast.TypeNode{Ident: ast.TypeInt},
			wantErr:     true,
			errContains: "LessThan constraint requires 1 argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the base type for the test
			tt.ensure.Assertion.BaseType = &tt.baseType.Ident

			expr, err := at.transformEnsure(tt.ensure)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if expr == nil {
				t.Error("expected non-nil expression")
				return
			}

			if tt.validate != nil {
				tt.validate(t, expr)
			}
		})
	}
}
