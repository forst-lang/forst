package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "variable reference",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 11},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, functionNode.Body[0], "ast.ReturnNode")
				varNode := assertNodeType[ast.VariableNode](t, returnNode.Value, "ast.VariableNode")
				if varNode.Ident.ID != "x" {
					t.Errorf("Expected variable 'x', got %s", varNode.Ident.ID)
				}
			},
		},
		{
			name: "reference to variable",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenBitwiseAnd, Value: "&", Line: 2, Column: 11},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 12},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, functionNode.Body[0], "ast.ReturnNode")
				refNode := assertNodeType[ast.ReferenceNode](t, returnNode.Value, "ast.ReferenceNode")
				varNode := assertNodeType[ast.VariableNode](t, refNode.Value, "ast.VariableNode")
				if varNode.Ident.ID != "x" {
					t.Errorf("Expected referenced variable 'x', got %s", varNode.Ident.ID)
				}
			},
		},
		{
			name: "dereference",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenStar, Value: "*", Line: 2, Column: 11},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 12},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, functionNode.Body[0], "ast.ReturnNode")
				derefNode := assertNodeType[ast.DereferenceNode](t, returnNode.Value, "ast.DereferenceNode")
				varNode := assertNodeType[ast.VariableNode](t, derefNode.Value, "ast.VariableNode")
				if varNode.Ident.ID != "x" {
					t.Errorf("Expected dereferenced variable 'x', got %s", varNode.Ident.ID)
				}
			},
		},
		{
			name: "qualified identifier",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "fmt", Line: 2, Column: 11},
				{Type: ast.TokenDot, Value: ".", Line: 2, Column: 14},
				{Type: ast.TokenIdentifier, Value: "Println", Line: 2, Column: 15},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, functionNode.Body[0], "ast.ReturnNode")
				varNode := assertNodeType[ast.VariableNode](t, returnNode.Value, "ast.VariableNode")
				if varNode.Ident.ID != "fmt.Println" {
					t.Errorf("Expected qualified identifier 'fmt.Println', got %s", varNode.Ident.ID)
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

func TestParseIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		expected ast.Identifier
	}{
		{
			name: "simple identifier",
			tokens: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 2},
			},
			expected: "x",
		},
		{
			name: "qualified identifier",
			tokens: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "fmt", Line: 1, Column: 1},
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 4},
				{Type: ast.TokenIdentifier, Value: "Println", Line: 1, Column: 5},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 12},
			},
			expected: "fmt.Println",
		},
		{
			name: "deeply qualified identifier",
			tokens: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "pkg", Line: 1, Column: 1},
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 4},
				{Type: ast.TokenIdentifier, Value: "sub", Line: 1, Column: 5},
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 8},
				{Type: ast.TokenIdentifier, Value: "func", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			expected: "pkg.sub.func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens)
			identifier := p.parseIdentifier()
			if identifier != tt.expected {
				t.Errorf("Expected identifier '%s', got '%s'", tt.expected, identifier)
			}
		})
	}
}
