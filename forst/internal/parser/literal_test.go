package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile_WithMapLiterals(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic map literal",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenVar, Value: "var", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "m", Line: 2, Column: 8},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 9},
				{Type: ast.TokenMap, Value: "map", Line: 2, Column: 11},
				{Type: ast.TokenLBracket, Value: "[", Line: 2, Column: 14},
				{Type: ast.TokenString, Value: "string", Line: 2, Column: 15},
				{Type: ast.TokenRBracket, Value: "]", Line: 2, Column: 21},
				{Type: ast.TokenInt, Value: "int", Line: 2, Column: 23},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 26},
				{Type: ast.TokenMap, Value: "map", Line: 2, Column: 28},
				{Type: ast.TokenLBracket, Value: "[", Line: 2, Column: 31},
				{Type: ast.TokenString, Value: "string", Line: 2, Column: 32},
				{Type: ast.TokenRBracket, Value: "]", Line: 2, Column: 38},
				{Type: ast.TokenInt, Value: "int", Line: 2, Column: 40},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 43},
				{Type: ast.TokenStringLiteral, Value: "\"key\"", Line: 2, Column: 44},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 49},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 51},
				{Type: ast.TokenRBrace, Value: "}", Line: 2, Column: 53},
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
				assignNode := assertNodeType[ast.AssignmentNode](t, functionNode.Body[0], "ast.AssignmentNode")
				if len(assignNode.LValues) != 1 {
					t.Fatalf("Expected 1 left value, got %d", len(assignNode.LValues))
				}
				if assignNode.LValues[0].Ident.ID != "m" {
					t.Errorf("Expected variable name 'm', got %s", assignNode.LValues[0].Ident.ID)
				}
				mapNode := assertNodeType[ast.MapLiteralNode](t, assignNode.RValues[0], "ast.MapLiteralNode")
				if len(mapNode.Entries) != 1 {
					t.Fatalf("Expected 1 map entry, got %d", len(mapNode.Entries))
				}
				entry := mapNode.Entries[0]
				key := assertNodeType[ast.StringLiteralNode](t, entry.Key, "ast.StringLiteralNode")
				if key.Value != "key" {
					t.Errorf("Expected key 'key', got %s", key.Value)
				}
				value := assertNodeType[ast.IntLiteralNode](t, entry.Value, "ast.IntLiteralNode")
				if value.Value != 42 {
					t.Errorf("Expected value 42, got %d", value.Value)
				}
			},
		},
		{
			name: "empty map literal",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenVar, Value: "var", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "m", Line: 2, Column: 8},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 9},
				{Type: ast.TokenMap, Value: "Map", Line: 2, Column: 11},
				{Type: ast.TokenLBracket, Value: "[", Line: 2, Column: 14},
				{Type: ast.TokenString, Value: "String", Line: 2, Column: 15},
				{Type: ast.TokenRBracket, Value: "]", Line: 2, Column: 21},
				{Type: ast.TokenInt, Value: "Int", Line: 2, Column: 23},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 26},
				{Type: ast.TokenMap, Value: "Map", Line: 2, Column: 28},
				{Type: ast.TokenLBracket, Value: "[", Line: 2, Column: 31},
				{Type: ast.TokenString, Value: "String", Line: 2, Column: 32},
				{Type: ast.TokenRBracket, Value: "]", Line: 2, Column: 38},
				{Type: ast.TokenInt, Value: "Int", Line: 2, Column: 40},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 43},
				{Type: ast.TokenRBrace, Value: "}", Line: 2, Column: 44},
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
				assignNode := assertNodeType[ast.AssignmentNode](t, functionNode.Body[0], "ast.AssignmentNode")
				if len(assignNode.LValues) != 1 {
					t.Fatalf("Expected 1 left value, got %d", len(assignNode.LValues))
				}
				mapNode := assertNodeType[ast.MapLiteralNode](t, assignNode.RValues[0], "ast.MapLiteralNode")
				if len(mapNode.Entries) != 0 {
					t.Fatalf("Expected 0 map entries, got %d", len(mapNode.Entries))
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
