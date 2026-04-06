package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestForstBuiltinReceiverGoType_errorAndScalars(t *testing.T) {
	t.Parallel()
	if _, name, ok := forstBuiltinReceiverGoType(ast.TypeNode{Ident: ast.TypeError}); !ok || name != "error" {
		t.Fatalf("error: ok=%v name=%q", ok, name)
	}
	if _, name, ok := forstBuiltinReceiverGoType(ast.TypeNode{Ident: ast.TypeString}); !ok || name != "string" {
		t.Fatalf("string: ok=%v name=%q", ok, name)
	}
	if _, _, ok := forstBuiltinReceiverGoType(ast.TypeNode{Ident: ast.TypeShape}); ok {
		t.Fatal("shape should not map to predeclared receiver hover")
	}
}

func TestForstBuiltinReceiverGoType_unwrapsPointer(t *testing.T) {
	t.Parallel()
	inner := ast.TypeNode{Ident: ast.TypeError}
	ptr := ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{inner}}
	if _, name, ok := forstBuiltinReceiverGoType(ptr); !ok || name != "error" {
		t.Fatalf("pointer to error: ok=%v name=%q", ok, name)
	}
}

func TestBuiltinGoDocParagraph_errorError(t *testing.T) {
	t.Parallel()
	pkg, err := loadBuiltinGoDocPackage(logrus.New())
	if err != nil || pkg == nil {
		t.Skip("GOROOT builtin package not available:", err)
	}
	s := builtinGoDocParagraph(pkg, "error", "Error")
	if s == "" {
		t.Fatal("expected doc text from package builtin for error.Error")
	}
	if !strings.Contains(strings.ToLower(s), "error") {
		t.Fatalf("expected error-related doc, got %q", s)
	}
}
