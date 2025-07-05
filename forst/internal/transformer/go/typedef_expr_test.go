package transformergo

import (
	"forst/internal/ast"
	"forst/internal/hasher"
	"forst/internal/typechecker"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetTypeAliasNameForTypeNode_RejectsOriginalName(t *testing.T) {
	tc := &typechecker.TypeChecker{}
	tc.Hasher = hasher.New()
	tc.Defs = make(map[ast.TypeIdent]ast.Node)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel) // silence output
	tf := &Transformer{TypeChecker: tc, log: logger}

	// Test built-in types
	for _, builtin := range []ast.TypeIdent{"string", "int", "float64", "bool", "void", "error"} {
		typeNode := ast.TypeNode{Ident: builtin, TypeKind: ast.TypeKindBuiltin}
		_, err := tf.getTypeAliasNameForTypeNode(typeNode)
		if err != nil {
			t.Errorf("builtin: %s, unexpected error: %v", builtin, err)
		}
	}

	// Test hash-based types
	hashTypeNode := ast.TypeNode{Ident: "T_abcdefghij", TypeKind: ast.TypeKindHashBased}
	_, err := tf.getTypeAliasNameForTypeNode(hashTypeNode)
	if err != nil {
		t.Errorf("hash-based name: unexpected error: %v", err)
	}

	// Test user-defined types
	userTypeNode := ast.TypeNode{Ident: "AppContext", TypeKind: ast.TypeKindUserDefined}
	_, err = tf.getTypeAliasNameForTypeNode(userTypeNode)
	if err != nil {
		t.Errorf("user-defined type: unexpected error: %v", err)
	}
}
