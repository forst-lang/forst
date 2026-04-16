package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestRangeTypesForTwoVars_arrayMapString(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)

	key, val, err := tc.rangeTypesForTwoVars(ast.TypeNode{
		Ident:      ast.TypeArray,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeString}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if key.Ident != ast.TypeInt || val.Ident != ast.TypeString {
		t.Fatalf("array: got key=%s val=%s", key.Ident, val.Ident)
	}

	key, val, err = tc.rangeTypesForTwoVars(ast.TypeNode{
		Ident: ast.TypeMap,
		TypeParams: []ast.TypeNode{
			{Ident: ast.TypeString},
			{Ident: ast.TypeInt},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if key.Ident != ast.TypeString || val.Ident != ast.TypeInt {
		t.Fatalf("map: got key=%s val=%s", key.Ident, val.Ident)
	}

	key, val, err = tc.rangeTypesForTwoVars(ast.TypeNode{Ident: ast.TypeString})
	if err != nil {
		t.Fatal(err)
	}
	if key.Ident != ast.TypeInt || val.Ident != ast.TypeInt {
		t.Fatalf("string two-var: got key=%s val=%s", key.Ident, val.Ident)
	}
}

func TestRangeTypesForTwoVars_unsupported(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	_, _, err := tc.rangeTypesForTwoVars(ast.TypeNode{Ident: ast.TypeFloat})
	if err == nil {
		t.Fatal("expected error for unsupported two-var range type")
	}
}
