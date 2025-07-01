package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseImport(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, importNode ast.ImportNode)
	}{
		{
			name: "simple import",
			tokens: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 6},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "fmt" {
					t.Errorf("Expected path 'fmt', got %s", importNode.Path)
				}
				if importNode.Alias != nil {
					t.Errorf("Expected no alias, got %s", importNode.Alias.ID)
				}
			},
		},
		{
			name: "import with alias",
			tokens: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "f", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 3},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 8},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "fmt" {
					t.Errorf("Expected path 'fmt', got %s", importNode.Path)
				}
				if importNode.Alias == nil {
					t.Fatal("Expected alias, got nil")
				}
				if importNode.Alias.ID != "f" {
					t.Errorf("Expected alias 'f', got %s", importNode.Alias.ID)
				}
			},
		},
		{
			name: "import with complex path",
			tokens: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"github.com/example/pkg"`, Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 26},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "github.com/example/pkg" {
					t.Errorf("Expected path 'github.com/example/pkg', got %s", importNode.Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens)
			importNode := p.parseImport()
			tt.validate(t, importNode)
		})
	}
}

func TestParseImportGroup(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 2},
		{Type: ast.TokenIdentifier, Value: "f", Line: 1, Column: 7},
		{Type: ast.TokenStringLiteral, Value: `"os"`, Line: 1, Column: 9},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 13},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 14},
	}

	p := setupParser(tokens)
	importGroup := p.parseImportGroup()

	if len(importGroup.Imports) != 2 {
		t.Fatalf("Expected 2 imports, got %d", len(importGroup.Imports))
	}

	// First import (fmt)
	if importGroup.Imports[0].Path != "fmt" {
		t.Errorf("Expected first import path 'fmt', got %s", importGroup.Imports[0].Path)
	}
	if importGroup.Imports[0].Alias != nil {
		t.Errorf("Expected first import to have no alias, got %s", importGroup.Imports[0].Alias.ID)
	}

	// Second import (os with alias f)
	if importGroup.Imports[1].Path != "os" {
		t.Errorf("Expected second import path 'os', got %s", importGroup.Imports[1].Path)
	}
	if importGroup.Imports[1].Alias == nil {
		t.Fatal("Expected second import to have alias")
	}
	if importGroup.Imports[1].Alias.ID != "f" {
		t.Errorf("Expected second import alias 'f', got %s", importGroup.Imports[1].Alias.ID)
	}
}

func TestParseImports(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "single import",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 8},
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
		{
			name: "import group",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 8},
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: `"os"`, Line: 1, Column: 14},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 18},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 19},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				importGroup := assertNodeType[ast.ImportGroupNode](t, nodes[0], "ast.ImportGroupNode")
				if len(importGroup.Imports) != 2 {
					t.Fatalf("Expected 2 imports in group, got %d", len(importGroup.Imports))
				}
				if importGroup.Imports[0].Path != "fmt" {
					t.Errorf("Expected first import path 'fmt', got %s", importGroup.Imports[0].Path)
				}
				if importGroup.Imports[1].Path != "os" {
					t.Errorf("Expected second import path 'os', got %s", importGroup.Imports[1].Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens)
			nodes := p.parseImports()
			tt.validate(t, nodes)
		})
	}
}
