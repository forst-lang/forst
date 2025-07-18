package parser

import (
	"forst/internal/ast"
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
			logger := ast.SetupTestLogger()
			p := setupParser(tt.tokens, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}
