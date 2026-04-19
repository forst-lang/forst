package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestLookupFieldPath_pointerStripsForOpaqueField(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	ptr := ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeImplicit}},
	}
	got, err := tc.lookupFieldPath(ptr, []string{"Field"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeImplicit {
		t.Fatalf("want implicit, got %s", got.Ident)
	}
}

func TestLookupFieldPath_implicitFieldIsOpaque(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	base := ast.TypeNode{Ident: ast.TypeImplicit}
	got, err := tc.lookupFieldPath(base, []string{"AnyName"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeImplicit {
		t.Fatalf("want implicit, got %s", got.Ident)
	}
}

func TestLookupFieldPath_implicitNestedPathStaysOpaque(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	base := ast.TypeNode{Ident: ast.TypeImplicit}
	got, err := tc.lookupFieldPath(base, []string{"A", "B"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeImplicit {
		t.Fatalf("want implicit, got %s", got.Ident)
	}
}
