package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestParameterSignature_String(t *testing.T) {
	t.Parallel()
	p := ParameterSignature{
		Ident: ast.Ident{ID: "x"},
		Type:  ast.TypeNode{Ident: ast.TypeInt},
	}
	if p.String() == "" {
		t.Fatal("expected non-empty")
	}
	if p.GetIdent() != "x" {
		t.Fatalf("GetIdent: %q", p.GetIdent())
	}
}

func TestFunctionSignature_String_and_GetIdent(t *testing.T) {
	t.Parallel()
	f := FunctionSignature{
		Ident: ast.Ident{ID: "f"},
		Parameters: []ParameterSignature{
			{Ident: ast.Ident{ID: "a"}, Type: ast.TypeNode{Ident: ast.TypeInt}},
		},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
	}
	if f.String() == "" {
		t.Fatal("expected non-empty")
	}
	if f.GetIdent() != "f" {
		t.Fatalf("GetIdent: %q", f.GetIdent())
	}
}
