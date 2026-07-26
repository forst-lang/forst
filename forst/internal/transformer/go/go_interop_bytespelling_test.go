package transformergo

import (
	"testing"

	forstast "forst/internal/ast"
)

func TestTransformTypeIdent_byteRuneComplexSpellings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   forstast.TypeIdent
		want string
	}{
		{forstast.TypeIdent("byte"), "byte"},
		{forstast.TypeIdent("rune"), "rune"},
		{forstast.TypeComplex64, "complex64"},
		{forstast.TypeComplex128, "complex128"},
	}
	for _, tc := range cases {
		got, err := transformTypeIdent(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if got.Name != tc.want {
			t.Fatalf("%s: want %q, got %q", tc.in, tc.want, got.Name)
		}
	}
}
