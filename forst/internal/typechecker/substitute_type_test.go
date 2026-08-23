package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
)

func TestSubstituteType(t *testing.T) {
	t.Parallel()
	tc := New(testutil.TestLogger(t, nil), false)
	bindings := map[ast.TypeIdent]ast.TypeNode{"T": ast.NewBuiltinType(ast.TypeInt)}

	tests := []struct {
		name string
		in   ast.TypeNode
		want string
	}{
		{
			name: "nestedArray",
			in:   ast.NewArrayType(ast.NewTypeParamType("T")),
			want: "Array(Int)",
		},
		{
			name: "shapeConstraintField",
			in: func() ast.TypeNode {
				baseType := ast.TypeIdent(ast.TypeShape)
				return ast.TypeNode{
					Ident: ast.TypeShape,
					Assertion: &ast.AssertionNode{
						BaseType: &baseType,
						Constraints: []ast.ConstraintNode{{
							Name: ConstraintMatch,
							Args: []ast.ConstraintArgumentNode{{
								Shape: &ast.ShapeNode{
									Fields: map[string]ast.ShapeFieldNode{
										"value": {Type: ptrTypeNode(ast.NewTypeParamType("T"))},
									},
								},
							}},
						}},
					},
				}
			}(),
			want: "Int",
		},
		{
			name: "assertionConstraintArg",
			in: ast.TypeNode{
				Ident: ast.TypeIdent("Payload"),
				Assertion: &ast.AssertionNode{
					BaseType: func() *ast.TypeIdent { id := ast.TypeIdent("T"); return &id }(),
					Constraints: []ast.ConstraintNode{{
						Name: "Ref",
						Args: []ast.ConstraintArgumentNode{{
							Type: ptrTypeNode(ast.NewTypeParamType("T")),
						}},
					}},
				},
			},
			want: "Payload(Int.Ref(Int))",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := tc.substituteTypeBindings(tt.in, bindings)
			if tt.name == "shapeConstraintField" {
				fields, ok := tc.ShapeFieldsFromParamType(out)
				if !ok {
					t.Fatal("expected shape fields")
				}
				ft, ok := ShapeFieldTypeNode(fields["value"])
				if !ok || ft.Ident != ast.TypeInt {
					t.Fatalf("value field = %+v, want Int", ft)
				}
				return
			}
			if out.String() != tt.want {
				t.Fatalf("got %s, want %s", out.String(), tt.want)
			}
		})
	}
}

func ptrTypeNode(t ast.TypeNode) *ast.TypeNode {
	return &t
}
