package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
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
			}
		})
	}
}
