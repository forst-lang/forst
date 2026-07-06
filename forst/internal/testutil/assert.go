package testutil

import (
	"fmt"
	"testing"

	"forst/internal/ast"
)

// AssertSingleType fails when types does not contain exactly one node with ident want.
func AssertSingleType(tb testing.TB, types []ast.TypeNode, want ast.TypeIdent) {
	tb.Helper()
	if len(types) != 1 {
		tb.Fatalf("expected exactly one type, got %d: %v", len(types), types)
	}
	if types[0].Ident != want {
		tb.Fatalf("expected type %q, got %q", want, types[0].Ident)
	}
}

// AssertSingleTypeString is like AssertSingleType with a string ident.
func AssertSingleTypeString(tb testing.TB, types []ast.TypeNode, want string) {
	tb.Helper()
	AssertSingleType(tb, types, ast.TypeIdent(want))
}

// FormatTypes returns a compact string for test failure messages.
func FormatTypes(types []ast.TypeNode) string {
	if len(types) == 0 {
		return "[]"
	}
	parts := make([]string, len(types))
	for i, ty := range types {
		parts[i] = string(ty.Ident)
	}
	return fmt.Sprintf("[%s]", fmt.Sprint(parts))
}
