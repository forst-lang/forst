package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func shapeTypeWithField(field string, fieldType ast.TypeNode) ast.TypeNode {
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
							field: {Type: &fieldType},
						},
					},
				}},
			}},
		},
	}
}

func TestInstantiateGenericCall_structuralUnify(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)

	tests := []struct {
		name        string
		paramType   ast.TypeNode
		argType     ast.TypeNode
		expectError string
		wantIdent   ast.TypeIdent
	}{
		{
			name:      "pointer",
			paramType: ast.NewPointerType(ast.NewTypeParamType("T")),
			argType:   ast.NewPointerType(ast.NewBuiltinType(ast.TypeInt)),
			wantIdent: ast.TypeInt,
		},
		{
			name: "map",
			paramType: ast.NewMapType(
				ast.NewTypeParamType("K"),
				ast.NewTypeParamType("V"),
			),
			argType: ast.NewMapType(
				ast.NewBuiltinType(ast.TypeString),
				ast.NewBuiltinType(ast.TypeInt),
			),
			wantIdent: ast.TypeString, // first unbound check - we'll verify both below
		},
		{
			name:      "result",
			paramType: ast.NewResultType(ast.NewTypeParamType("T"), ast.NewBuiltinType(ast.TypeError)),
			argType:   ast.NewResultType(ast.NewBuiltinType(ast.TypeInt), ast.NewBuiltinType(ast.TypeError)),
			wantIdent: ast.TypeInt,
		},
		{
			name:      "shapeField",
			paramType: shapeTypeWithField("value", ast.NewTypeParamType("T")),
			argType:   shapeTypeWithField("value", ast.NewBuiltinType(ast.TypeInt)),
			wantIdent: ast.TypeInt,
		},
		{
			name:        "fixedArrayLengthMismatch",
			paramType:   ast.NewFixedArrayType(ast.NewTypeParamType("T"), 2),
			argType:     ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 3),
			expectError: "cannot unify",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			typeParams := []ast.TypeParamDecl{{Name: "T"}}
			if tt.name == "map" {
				typeParams = []ast.TypeParamDecl{{Name: "K"}, {Name: "V"}}
			}
			sig := normalizeGenericSignature(ast.FunctionNode{
				Ident:      ast.Ident{ID: "f"},
				TypeParams: typeParams,
				Params: []ast.ParamNode{ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  tt.paramType,
				}},
				ReturnTypes: []ast.TypeNode{ast.NewBuiltinType(ast.TypeBool)},
			})
			inst, err := tc.instantiateGenericCall(sig, [][]ast.TypeNode{{tt.argType}}, ast.FakeSpan())
			if tt.expectError != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.expectError) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.expectError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.name == "map" {
				k := inst.Parameters[0].Type.TypeParams[0].Ident
				v := inst.Parameters[0].Type.TypeParams[1].Ident
				if k != ast.TypeString || v != ast.TypeInt {
					t.Fatalf("map types = %s, %s", k, v)
				}
				return
			}
			got := inst.Parameters[0].Type
			switch tt.paramType.Ident {
			case ast.TypePointer, ast.TypeResult:
				got = got.TypeParams[0]
			case ast.TypeShape:
				fields, ok := tc.ShapeFieldsFromParamType(got)
				if !ok {
					t.Fatal("expected shape fields")
				}
				ft, ok := ShapeFieldTypeNode(fields["value"])
				if !ok {
					t.Fatal("missing value field")
				}
				got = ft
			}
			if got.Ident != tt.wantIdent {
				t.Fatalf("got %s, want %s", got.Ident, tt.wantIdent)
			}
		})
	}
}
