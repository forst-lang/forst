package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestNormalizeTypeForStorage_branches(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)

	t.Run("type param unchanged", func(t *testing.T) {
		in := ast.NewTypeParamType("T")
		got := tc.normalizeTypeForStorage(in)
		if !got.IsTypeParam() {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("hash based unchanged", func(t *testing.T) {
		in := ast.NewHashBasedType("T_abc")
		got := tc.normalizeTypeForStorage(in)
		if !got.IsHashBased() {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("builtin unchanged", func(t *testing.T) {
		in := ast.NewBuiltinType(ast.TypeInt)
		got := tc.normalizeTypeForStorage(in)
		if !got.IsGoBuiltin() {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("named user type marked user defined", func(t *testing.T) {
		in := ast.TypeNode{Ident: "AppContext"}
		got := tc.normalizeTypeForStorage(in)
		if !got.IsUserDefined() {
			t.Fatalf("got %+v", got)
		}
	})
}

func TestNormalizeTypesForStorage_emptySlice(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	if got := tc.normalizeTypesForStorage(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
