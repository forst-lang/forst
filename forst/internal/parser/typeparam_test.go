package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFunctionDefinition_typeParams(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Type: ast.TokenFunc, Value: "func"},
		{Type: ast.TokenIdentifier, Value: "identity"},
		{Type: ast.TokenLBracket, Value: "["},
		{Type: ast.TokenIdentifier, Value: "T"},
		{Type: ast.TokenIdentifier, Value: "any"},
		{Type: ast.TokenRBracket, Value: "]"},
		{Type: ast.TokenLParen, Value: "("},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenIdentifier, Value: "T"},
		{Type: ast.TokenRParen, Value: ")"},
		{Type: ast.TokenColon, Value: ":"},
		{Type: ast.TokenIdentifier, Value: "T"},
		{Type: ast.TokenLBrace, Value: "{"},
		{Type: ast.TokenReturn, Value: "return"},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenRBrace, Value: "}"},
		{Type: ast.TokenEOF},
	}
	p := New(tokens, "test.ft", nil)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := nodes[0].(ast.FunctionNode)
	if !ok {
		t.Fatalf("expected FunctionNode, got %T", nodes[0])
	}
	if len(fn.TypeParams) != 1 || fn.TypeParams[0].Name != "T" {
		t.Fatalf("TypeParams: %+v", fn.TypeParams)
	}
}
