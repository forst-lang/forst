package transformergo

import (
	"testing"

	"forst/internal/ast"
)

func TestTransformTypeIdent_builtins_and_user(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ident ast.TypeIdent
		want  string
	}{
		{ast.TypeString, "string"},
		{ast.TypeInt, "int"},
		{ast.TypeFloat, "float64"},
		{ast.TypeBool, "bool"},
		{ast.TypeVoid, "void"},
		{ast.TypeError, "error"},
		{ast.TypeIdent("UserShape"), "UserShape"},
	}
	for _, tc := range cases {
		id, err := transformTypeIdent(tc.ident)
		if err != nil {
			t.Fatalf("%q: %v", tc.ident, err)
		}
		if id.Name != tc.want {
			t.Fatalf("%q: got %q want %q", tc.ident, id.Name, tc.want)
		}
	}
}

func TestTransformTypeIdent_invalid_placeholders_return_error(t *testing.T) {
	t.Parallel()
	for _, ident := range []ast.TypeIdent{ast.TypeObject, ast.TypeAssertion, ast.TypeImplicit} {
		_, err := transformTypeIdent(ident)
		if err == nil {
			t.Fatalf("%q: expected error", ident)
		}
	}
}
