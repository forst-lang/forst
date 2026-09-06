package parser

import (
	"forst/internal/ast"
	"forst/internal/lexer"
	"testing"
)

func TestParseTypeDef(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "type alias",
			tokens: []ast.Token{
				{Type: ast.TokenType, Value: "type", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "MyInt", Line: 1, Column: 6},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 14},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 17},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				typeDefNode := assertNodeType[ast.TypeDefNode](t, nodes[0], "ast.TypeDefNode")
				if typeDefNode.Ident != ast.TypeIdent("MyInt") {
					t.Errorf("Expected type name 'MyInt', got %s", typeDefNode.Ident)
				}
				// The Expr field contains the type definition expression
				if typeDefNode.Expr == nil {
					t.Fatal("Expected type definition expression, got nil")
				}
			},
		},
		{
			name: "type definition with struct",
			tokens: []ast.Token{
				{Type: ast.TokenType, Value: "type", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "Person", Line: 1, Column: 6},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 14},
				{Type: ast.TokenIdentifier, Value: "name", Line: 2, Column: 4},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 8},
				{Type: ast.TokenString, Value: "string", Line: 2, Column: 10},
				{Type: ast.TokenComma, Value: ",", Line: 2, Column: 16},
				{Type: ast.TokenIdentifier, Value: "age", Line: 3, Column: 4},
				{Type: ast.TokenColon, Value: ":", Line: 3, Column: 7},
				{Type: ast.TokenInt, Value: "int", Line: 3, Column: 9},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				typeDefNode := assertNodeType[ast.TypeDefNode](t, nodes[0], "ast.TypeDefNode")
				if typeDefNode.Ident != ast.TypeIdent("Person") {
					t.Errorf("Expected type name 'Person', got %s", typeDefNode.Ident)
				}
				// The Expr field contains the type definition expression
				if typeDefNode.Expr == nil {
					t.Fatal("Expected type definition expression, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ast.SetupTestLogger(nil)
			p := setupParser(tt.tokens, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}

func TestParseTypeDef_sliceAliasPreservesTypeParams(t *testing.T) {
	src := `package main
type Bytes = []Byte
`
	logger := ast.SetupTestLogger(nil)
	toks := lexer.New([]byte(src), "t.ft", logger).Lex()
	p := New(toks, "t.ft", logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var td ast.TypeDefNode
	for _, n := range nodes {
		if d, ok := n.(ast.TypeDefNode); ok && d.Ident == "Bytes" {
			td = d
			break
		}
	}
	if td.Ident == "" {
		t.Fatal("expected Bytes typedef")
	}
	ade, ok := td.Expr.(ast.TypeDefAssertionExpr)
	if !ok || ade.Assertion == nil {
		t.Fatalf("want TypeDefAssertionExpr, got %T", td.Expr)
	}
	if ade.Assertion.BaseType == nil || *ade.Assertion.BaseType != ast.TypeArray {
		t.Fatalf("want TYPE_ARRAY base, got %v", ade.Assertion.BaseType)
	}
	if len(ade.Assertion.TypeParams) != 1 || ade.Assertion.TypeParams[0].Ident != "Byte" {
		t.Fatalf("want TypeParams [Byte], got %#v", ade.Assertion.TypeParams)
	}
}
