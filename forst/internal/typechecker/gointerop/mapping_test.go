package gointerop_test

import (
	"go/types"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

func TestTypeToForstType_complex64(t *testing.T) {
	t.Parallel()
	got, ok := gointerop.TypeToForstType(types.Typ[types.Complex64])
	if !ok || got.Ident != ast.TypeComplex64 {
		t.Fatalf("want Complex64, got ok=%v %#v", ok, got)
	}
}

func TestTypeToForstType_complex128(t *testing.T) {
	t.Parallel()
	got, ok := gointerop.TypeToForstType(types.Typ[types.Complex128])
	if !ok || got.Ident != ast.TypeComplex128 {
		t.Fatalf("want Complex128, got ok=%v %#v", ok, got)
	}
}

func TestTypeToForstType_complex64And128AreDistinct(t *testing.T) {
	t.Parallel()
	c64, ok := gointerop.TypeToForstType(types.Typ[types.Complex64])
	if !ok {
		t.Fatal("complex64 should map")
	}
	c128, ok := gointerop.TypeToForstType(types.Typ[types.Complex128])
	if !ok {
		t.Fatal("complex128 should map")
	}
	if c64.Ident == c128.Ident {
		t.Fatal("complex64 and complex128 must map to distinct Forst types")
	}
}

func TestTypeToForstType_byteAndRuneSpellings(t *testing.T) {
	t.Parallel()
	byteT, ok := gointerop.TypeToForstType(types.Typ[types.Byte])
	if !ok || byteT.Ident != ast.TypeIdent("byte") {
		t.Fatalf("want byte ident, got ok=%v %#v", ok, byteT)
	}
	runeT, ok := gointerop.TypeToForstType(types.Typ[types.Rune])
	if !ok || runeT.Ident != ast.TypeIdent("rune") {
		t.Fatalf("want rune ident, got ok=%v %#v", ok, runeT)
	}
}
