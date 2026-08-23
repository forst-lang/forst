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
			if out.String() != tt.want {
				t.Fatalf("got %s, want %s", out.String(), tt.want)
			}
		})
	}
}

func ptrTypeNode(t ast.TypeNode) *ast.TypeNode {
	return &t
}
