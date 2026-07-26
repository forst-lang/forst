package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseType_fixedArray(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)
	p := setupParser([]ast.Token{
		{Type: ast.TokenLBracket, Value: "["},
		{Type: ast.TokenIntLiteral, Value: "3"},
		{Type: ast.TokenRBracket, Value: "]"},
		{Type: ast.TokenInt, Value: "Int"},
		{Type: ast.TokenEOF},
	}, logger)
	typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	if !typ.IsFixedArray() || typ.ArrayLen == nil || *typ.ArrayLen != 3 {
		t.Fatalf("got %+v", typ)
	}
	if len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("elem = %+v", typ.TypeParams)
	}
}

func TestParseType_sliceDistinctFromFixedArray(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)
	slice := setupParser([]ast.Token{
		{Type: ast.TokenLBracket, Value: "["},
		{Type: ast.TokenRBracket, Value: "]"},
		{Type: ast.TokenInt, Value: "Int"},
		{Type: ast.TokenEOF},
	}, logger).parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	if !slice.IsSlice() {
		t.Fatalf("want slice, got %+v", slice)
	}
	fixed := setupParser([]ast.Token{
		{Type: ast.TokenLBracket, Value: "["},
		{Type: ast.TokenIntLiteral, Value: "3"},
		{Type: ast.TokenRBracket, Value: "]"},
		{Type: ast.TokenInt, Value: "Int"},
		{Type: ast.TokenEOF},
	}, logger).parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	if !fixed.IsFixedArray() {
		t.Fatalf("want fixed array, got %+v", fixed)
	}
}

func TestParseType_fixedArrayRejectsNegativeLength(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)
	p := setupParser([]ast.Token{
		{Type: ast.TokenLBracket, Value: "["},
		{Type: ast.TokenMinus, Value: "-"},
		{Type: ast.TokenIntLiteral, Value: "1"},
		{Type: ast.TokenRBracket, Value: "]"},
		{Type: ast.TokenInt, Value: "Int"},
		{Type: ast.TokenEOF},
	}, logger)
	defer func() {
		if recover() == nil {
			t.Fatal("expected parse error for negative array length")
		}
	}()
	p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
}

func TestParseFile_fixedArrayVar(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	var xs [3]Int
	xs[0] = 1
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := nodes[1].(ast.FunctionNode)
	varDecl := fn.Body[0].(ast.AssignmentNode)
	if len(varDecl.ExplicitTypes) != 1 || varDecl.ExplicitTypes[0] == nil || !varDecl.ExplicitTypes[0].IsFixedArray() {
		t.Fatalf("want [3]Int var type, got %+v", varDecl.ExplicitTypes)
	}
}
