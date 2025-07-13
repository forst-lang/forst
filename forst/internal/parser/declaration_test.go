package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile_WithPackageDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "package declaration",
			tokens: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				packageNode := assertNodeType[ast.PackageNode](t, nodes[0], "ast.PackageNode")
				if packageNode.GetIdent() != "main" {
					t.Errorf("Expected package name 'main', got '%s'", packageNode.GetIdent())
				}
			},
		},
		{
			name: "import declaration",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: "fmt", Line: 1, Column: 8},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				importNode := assertNodeType[ast.ImportNode](t, nodes[0], "ast.ImportNode")
				if importNode.Path != "fmt" {
					t.Errorf("Expected import path 'fmt', got %s", importNode.Path)
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

func TestParseFile_WithTypeDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic assertion type definition",
			tokens: []ast.Token{
				{Type: ast.TokenType, Value: "type", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "Username", Line: 1, Column: 6},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
				{Type: ast.TokenString, Value: "String", Line: 1, Column: 14},
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 20},
				{Type: ast.TokenIdentifier, Value: "Min", Line: 1, Column: 21},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 24},
				{Type: ast.TokenIntLiteral, Value: "3", Line: 1, Column: 25},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 26},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 20},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				typeNode := assertNodeType[ast.TypeDefNode](t, nodes[0], "ast.TypeDefNode")
				if typeNode.Ident != "Username" {
					t.Errorf("Expected type name 'Username', got %s", typeNode.Ident)
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
