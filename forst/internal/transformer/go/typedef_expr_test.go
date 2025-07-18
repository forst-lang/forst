package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
	"testing"
)

func TestGetTypeAliasNameForTypeNode_RejectsOriginalName(t *testing.T) {
	logger := ast.SetupTestLogger()
	tc := typechecker.New(logger, false)
	tf := New(tc, logger)

	// Test built-in types
	for _, builtin := range []ast.TypeIdent{"string", "int", "float64", "bool", "void", "error"} {
		typeNode := ast.TypeNode{Ident: builtin, TypeKind: ast.TypeKindBuiltin}
		_, err := tf.TypeChecker.GetAliasedTypeName(typeNode)
		if err != nil {
			t.Errorf("builtin: %s, unexpected error: %v", builtin, err)
		}
	}

	// Test hash-based types
	hashTypeNode := ast.TypeNode{Ident: "T_abcdefghij", TypeKind: ast.TypeKindHashBased}
	_, err := tf.TypeChecker.GetAliasedTypeName(hashTypeNode)
	if err != nil {
		t.Errorf("hash-based name: unexpected error: %v", err)
	}

	// Test user-defined types
	userTypeNode := ast.TypeNode{Ident: "AppContext", TypeKind: ast.TypeKindUserDefined}
	_, err = tf.TypeChecker.GetAliasedTypeName(userTypeNode)
	if err != nil {
		t.Errorf("user-defined type: unexpected error: %v", err)
	}
}
