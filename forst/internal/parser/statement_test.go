package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseBlockStatement(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "var statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenVar, Value: "var", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 8},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 9},
				{Type: ast.TokenInt, Value: "int", Line: 2, Column: 11},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 15},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 17},
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
				if assignNode.LValues[0].Ident.ID != "x" {
					t.Errorf("Expected variable name 'x', got %s", assignNode.LValues[0].Ident.ID)
				}
			},
		},
		{
			name: "return statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 11},
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
				if len(returnNode.Values) != 1 {
					t.Fatal("Expected exactly one return value")
				}
				if returnNode.Values[0] == nil {
					t.Fatal("Expected return value, got nil")
				}
				value := assertNodeType[ast.IntLiteralNode](t, returnNode.Values[0], "ast.IntLiteralNode")
				if value.Value != 42 {
					t.Errorf("Expected return value 42, got %d", value.Value)
				}
			},
		},
		{
			name: "expression statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIdentifier, Value: "fmt", Line: 2, Column: 4},
				{Type: ast.TokenDot, Value: ".", Line: 2, Column: 7},
				{Type: ast.TokenIdentifier, Value: "Println", Line: 2, Column: 8},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 15},
				{Type: ast.TokenStringLiteral, Value: "Hello", Line: 2, Column: 16},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 22},
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
				// Should be a function call expression
				if _, ok := functionNode.Body[0].(ast.FunctionCallNode); !ok {
					t.Errorf("Expected FunctionCallNode, got %T", functionNode.Body[0])
				}
			},
		},
		{
			name: "if statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIf, Value: "if", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 7},
				{Type: ast.TokenGreater, Value: ">", Line: 2, Column: 9},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 2, Column: 11},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 13},
				{Type: ast.TokenReturn, Value: "return", Line: 3, Column: 8},
				{Type: ast.TokenIdentifier, Value: "x", Line: 3, Column: 15},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 5, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 5, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				ifNode := assertNodeType[*ast.IfNode](t, functionNode.Body[0], "*ast.IfNode")
				if ifNode.Condition == nil {
					t.Fatal("Expected if condition, got nil")
				}
			},
		},
		{
			name: "multiple return values",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "1", Line: 2, Column: 11},
				{Type: ast.TokenComma, Value: ",", Line: 2, Column: 12},
				{Type: ast.TokenIntLiteral, Value: "2", Line: 2, Column: 14},
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
				// This will fail until multiple return values are supported
				if len(returnNode.Values) != 2 {
					t.Fatal("Expected exactly two return values")
				}
				if returnNode.Values[0] == nil || returnNode.Values[1] == nil {
					t.Fatal("Expected return values, got nil")
				}
				// Should be a tuple or slice of values in the future
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
