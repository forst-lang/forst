package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
)

func TestParseFile_arrayLiteralTypeSuffixNotAfterNewline(t *testing.T) {
	// Regression: `]` at end of line must not consume the next statement's identifier as `]T`.
	src := `package main

func main() {
	ys := [0, 0, 0]
	xs := [1, 2, 3]
	println(len(xs))
}
`
	logger := ast.SetupTestLogger(nil)
	toks := lexer.New([]byte(src), "t.ft", logger).Lex()
	p := New(toks, "t.ft", logger)
	_, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
}

func TestParseFile_arrayLiteralOptionalTypeSameLine(t *testing.T) {
	// `]T` on the same line as `]` is still a valid element-type suffix (user-defined name).
	src := `package main

func main() {
	x := [1, 2]T
	println(len(x))
}
`
	logger := ast.SetupTestLogger(nil)
	toks := lexer.New([]byte(src), "t.ft", logger).Lex()
	p := New(toks, "t.ft", logger)
	_, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
}

func TestParseFile_typedCompositeArrayLiteral(t *testing.T) {
	t.Parallel()
	src := `package main

func f(): []String {
	return []String{"bun"}
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	ret := fn.Body[0].(ast.ReturnNode)
	arr := ret.Values[0].(ast.ArrayLiteralNode)
	if arr.Type.Ident != ast.TypeString {
		t.Fatalf("want String elem type, got %v", arr.Type.Ident)
	}
	if len(arr.Value) != 1 {
		t.Fatalf("want 1 element, got %d", len(arr.Value))
	}
}
