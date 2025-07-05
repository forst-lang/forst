package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "empty file",
			tokens: []ast.Token{
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 1},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 0 {
					t.Fatalf("Expected 0 nodes for empty file, got %d", len(nodes))
				}
			},
		},
		{
			name: "file with comments only",
			tokens: []ast.Token{
				{Type: ast.TokenComment, Value: "// This is a comment", Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 2, Column: 1},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 0 {
					t.Fatalf("Expected 0 nodes for file with comments only, got %d", len(nodes))
				}
			},
		},
		{
			name: "file with package and function",
			tokens: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
				{Type: ast.TokenFunc, Value: "func", Line: 2, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 2, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 3, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 3, Column: 11},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(nodes))
				}
				packageNode := assertNodeType[ast.PackageNode](t, nodes[0], "ast.PackageNode")
				if packageNode.Ident.ID != "main" {
					t.Errorf("Expected package name 'main', got %s", packageNode.Ident.ID)
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
				if functionNode.Ident.ID != "main" {
					t.Errorf("Expected function name 'main', got %s", functionNode.Ident.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}

func TestParseFile_WithUnexpectedToken(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "unexpected", Line: 1, Column: 1},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 11},
	}

	p := setupParser(tokens)
	_, err := p.ParseFile()
	if err == nil {
		t.Fatal("Expected parse error for unexpected token, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected ParseError, got %T", err)
	}

	if parseErr.Token.Value != "unexpected" {
		t.Errorf("Expected error token 'unexpected', got %s", parseErr.Token.Value)
	}
}
