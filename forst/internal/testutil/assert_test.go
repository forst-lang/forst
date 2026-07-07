package testutil

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestAssertSingleType(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		AssertSingleTypeString(t, []ast.TypeNode{{Ident: "Int"}}, "Int")
	})
	t.Run("wrongCount", func(t *testing.T) {
		msg := stubFailf(t, func() {
			AssertSingleType(t, nil, "Int")
		})
		if !strings.Contains(msg, "expected exactly one type") {
			t.Fatalf("msg = %q", msg)
		}
	})
	t.Run("tooMany", func(t *testing.T) {
		msg := stubFailf(t, func() {
			AssertSingleType(t, []ast.TypeNode{{Ident: "Int"}, {Ident: "String"}}, "Int")
		})
		if !strings.Contains(msg, "expected exactly one type") {
			t.Fatalf("msg = %q", msg)
		}
	})
	t.Run("wrongIdent", func(t *testing.T) {
		msg := stubFailf(t, func() {
			AssertSingleType(t, []ast.TypeNode{{Ident: "String"}}, "Int")
		})
		if !strings.Contains(msg, `expected type "Int"`) {
			t.Fatalf("msg = %q", msg)
		}
	})
}

func TestFormatTypes(t *testing.T) {
	if got := FormatTypes(nil); got != "[]" {
		t.Fatalf("FormatTypes(nil) = %q", got)
	}
	got := FormatTypes([]ast.TypeNode{{Ident: "Int"}, {Ident: "String"}})
	if !strings.Contains(got, "Int") || !strings.Contains(got, "String") {
		t.Fatalf("FormatTypes = %q", got)
	}
}
