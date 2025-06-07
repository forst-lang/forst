package ast

import (
	"testing"
)

func typeIdentPtr(s string) *TypeIdent {
	ti := TypeIdent(s)
	return &ti
}

func TestValidateShapeGuard(t *testing.T) {
	tests := []struct {
		name    string
		node    TypeGuardNode
		wantErr bool
	}{
		{
			name: "valid shape guard",
			node: TypeGuardNode{
				Ident: "HasField",
				Subject: SimpleParamNode{
					Ident: Ident{ID: "s"},
					Type:  TypeNode{Ident: "Shape"},
				},
				Body: []Node{
					ReturnNode{
						Value: BinaryExpressionNode{
							Left:     VariableNode{Ident: Ident{ID: "s"}},
							Operator: TokenIs,
							Right: ShapeNode{
								Fields: map[string]ShapeFieldNode{
									"field": {
										Assertion: &AssertionNode{
											BaseType: typeIdentPtr("String"),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid receiver type",
			node: TypeGuardNode{
				Ident: "HasField",
				Subject: SimpleParamNode{
					Ident: Ident{ID: "s"},
					Type:  TypeNode{Ident: "Int"},
				},
				Body: []Node{
					ReturnNode{
						Value: BinaryExpressionNode{
							Left:     VariableNode{Ident: Ident{ID: "s"}},
							Operator: TokenIs,
							Right: ShapeNode{
								Fields: map[string]ShapeFieldNode{
									"field": {
										Assertion: &AssertionNode{
											BaseType: typeIdentPtr("String"),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid body - no return",
			node: TypeGuardNode{
				Ident: "HasField",
				Subject: SimpleParamNode{
					Ident: Ident{ID: "s"},
					Type:  TypeNode{Ident: "Shape"},
				},
				Body: []Node{},
			},
			wantErr: true,
		},
		{
			name: "invalid body - not a shape refinement",
			node: TypeGuardNode{
				Ident: "HasField",
				Subject: SimpleParamNode{
					Ident: Ident{ID: "s"},
					Type:  TypeNode{Ident: "Shape"},
				},
				Body: []Node{
					ReturnNode{
						Value: BinaryExpressionNode{
							Left:     VariableNode{Ident: Ident{ID: "s"}},
							Operator: TokenIs,
							Right: AssertionNode{
								BaseType: typeIdentPtr("Int"),
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShapeGuard(tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateShapeGuard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsShapeRefinement(t *testing.T) {
	tests := []struct {
		name string
		node Node
		want bool
	}{
		{
			name: "valid shape refinement",
			node: BinaryExpressionNode{
				Left:     VariableNode{Ident: Ident{ID: "s"}},
				Operator: TokenIs,
				Right: ShapeNode{
					Fields: map[string]ShapeFieldNode{
						"field": {
							Assertion: &AssertionNode{
								BaseType: typeIdentPtr("String"),
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "invalid - not a binary expression",
			node: VariableNode{Ident: Ident{ID: "s"}},
			want: false,
		},
		{
			name: "invalid - wrong operator",
			node: BinaryExpressionNode{
				Left:     VariableNode{Ident: Ident{ID: "s"}},
				Operator: TokenEquals,
				Right: ShapeNode{
					Fields: map[string]ShapeFieldNode{
						"field": {
							Assertion: &AssertionNode{
								BaseType: typeIdentPtr("String"),
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "invalid - not a shape",
			node: BinaryExpressionNode{
				Left:     VariableNode{Ident: Ident{ID: "s"}},
				Operator: TokenIs,
				Right: AssertionNode{
					BaseType: typeIdentPtr("Int"),
				},
			},
			want: false,
		},
		{
			name: "invalid - empty shape",
			node: BinaryExpressionNode{
				Left:     VariableNode{Ident: Ident{ID: "s"}},
				Operator: TokenIs,
				Right: ShapeNode{
					Fields: map[string]ShapeFieldNode{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isShapeRefinement(tt.node); got != tt.want {
				t.Errorf("isShapeRefinement() = %v, want %v", got, tt.want)
			}
		})
	}
}
