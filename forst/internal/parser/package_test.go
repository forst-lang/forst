package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParsePackage(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, packageNode ast.PackageNode)
	}{
		{
			name: "simple package declaration",
			tokens: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			validate: func(t *testing.T, packageNode ast.PackageNode) {
				if packageNode.Ident.ID != "main" {
					t.Errorf("Expected package name 'main', got %s", packageNode.Ident.ID)
				}
			},
		},
		{
			name: "package with complex name",
			tokens: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "github_com_example_pkg", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 33},
			},
			validate: func(t *testing.T, packageNode ast.PackageNode) {
				if packageNode.Ident.ID != "github_com_example_pkg" {
					t.Errorf("Expected package name 'github_com_example_pkg', got %s", packageNode.Ident.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens)
			packageNode := p.parsePackage()
			tt.validate(t, packageNode)
		})
	}
}

func TestParsePackage_SetsContextPackage(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "test", Line: 1, Column: 9},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
	}

	p := setupParser(tokens)
	packageNode := p.parsePackage()

	// Check that the package was set in the context
	if p.context.Package == nil {
		t.Fatal("Expected package to be set in context, got nil")
	}
	if p.context.Package.Ident.ID != "test" {
		t.Errorf("Expected context package name 'test', got %s", p.context.Package.Ident.ID)
	}

	// Verify the returned package node matches
	if packageNode.Ident.ID != "test" {
		t.Errorf("Expected returned package name 'test', got %s", packageNode.Ident.ID)
	}
}
