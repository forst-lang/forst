package transformergo

import (
	"go/ast"
	"testing"

	forstast "forst/internal/ast"
)

func TestVariablePathSegments(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want []string
	}{
		{"x", []string{"x"}},
		{"w.r", []string{"w", "r"}},
		{"a.b.c", []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		vn := forstast.VariableNode{Ident: forstast.Ident{ID: forstast.Identifier(tc.id)}}
		got := variablePathSegments(vn)
		if len(got) != len(tc.want) {
			t.Fatalf("%q: got %v want %v", tc.id, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%q: got %v want %v", tc.id, got, tc.want)
			}
		}
	}
}

func TestIsDotQualifiedVariable(t *testing.T) {
	t.Parallel()
	if !isDotQualifiedVariable(forstast.VariableNode{Ident: forstast.Ident{ID: "a.b"}}) {
		t.Fatal("expected dot-qualified")
	}
	if isDotQualifiedVariable(forstast.VariableNode{Ident: forstast.Ident{ID: "x"}}) {
		t.Fatal("expected simple name")
	}
}

func TestGoLoweredResultSelectors(t *testing.T) {
	t.Parallel()
	base := ast.NewIdent("w")
	v := goLoweredResultValueSelector(base)
	if v.Sel.Name != loweredResultValueFieldName {
		t.Fatalf("V sel: got %q", v.Sel.Name)
	}
	e := goLoweredResultErrSelector(base)
	if e.Sel.Name != loweredResultErrFieldName {
		t.Fatalf("Err sel: got %q", e.Sel.Name)
	}
}
