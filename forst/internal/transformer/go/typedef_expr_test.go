package transformergo

import (
	"forst/internal/ast"
	"forst/internal/hasher"
	"forst/internal/typechecker"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestIsHashBasedTypeName(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"T_abcdefghij", true},
		{"T_1234567890", true},
		{"T_a1B2c3D4e5", true},
		{"T_abc", false}, // too short
		{"AppContext", false},
		{"T-abcdefghij", false},
		{"T_abc$efghij", false},
		{"T_abcdefghijX", true}, // allow longer
	}
	for _, c := range cases {
		if got := isHashBasedTypeName(c.name); got != c.expected {
			t.Errorf("isHashBasedTypeName(%q) = %v, want %v", c.name, got, c.expected)
		}
	}
}

func TestGetTypeAliasNameForTypeNode_RejectsOriginalName(t *testing.T) {
	tc := &typechecker.TypeChecker{}
	tc.Hasher = hasher.New()
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel) // silence output
	tf := &Transformer{TypeChecker: tc, log: logger}
	// Built-in types should pass
	for _, builtin := range []ast.TypeIdent{"string", "int", "float64", "bool", "void", "error"} {
		_, err := tf.getTypeAliasNameForTypeNode(ast.TypeNode{Ident: builtin})
		if err != nil {
			t.Errorf("builtin: %s, unexpected error: %v", builtin, err)
		}
	}
	// Hash-based names should pass
	_, err := tf.getTypeAliasNameForTypeNode(ast.TypeNode{Ident: "T_abcdefghij"})
	if err != nil {
		t.Errorf("hash-based name: unexpected error: %v", err)
	}
	// Original names should fail
	_, err = tf.getTypeAliasNameForTypeNode(ast.TypeNode{Ident: "AppContext"})
	if err == nil || err.Error() == "" || !contains(err.Error(), "BUG: attempted to use original Forst type name") {
		t.Errorf("expected error for original name, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
