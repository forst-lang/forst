package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile_WithBinaryExpressionInFunction(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "passwordStrength", Line: 1, Column: 6},
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 14},
		{Type: ast.TokenIdentifier, Value: "password", Line: 1, Column: 15},
		{Type: ast.TokenIdentifier, Value: "Password", Line: 1, Column: 25},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 32},
		{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 33},
		{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
		{Type: ast.TokenIdentifier, Value: "len", Line: 2, Column: 11},
		{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 14},
		{Type: ast.TokenIdentifier, Value: "password", Line: 2, Column: 15},
		{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 22},
		{Type: ast.TokenPlus, Value: "+", Line: 2, Column: 24},
		{Type: ast.TokenIntLiteral, Value: "12", Line: 2, Column: 27},
		{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
		{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
	}

	p := setupParser(tokens)
	nodes, err := p.ParseFile()
	if err == nil {
		if len(nodes) != 1 {
			t.Fatalf("Expected 1 node, got %d", len(nodes))
		}
		functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
		if functionNode.GetIdent() != "passwordStrength" {
			t.Errorf("Expected function name 'isStrong', got %s", functionNode.GetIdent())
		}
		if len(functionNode.Body) != 1 {
			t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
		}
		returnNode := assertNodeType[ast.ReturnNode](t, functionNode.Body[0], "ast.ReturnNode")
		// Optionally, check that the return value is a binary expression
		if len(returnNode.Values) != 1 {
			t.Fatal("Expected exactly one return value")
		}
		if _, ok := returnNode.Values[0].(ast.BinaryExpressionNode); !ok {
			t.Errorf("Expected return value to be a BinaryExpressionNode, got %T", returnNode.Values[0])
		}
	} else {
		t.Fatalf("ParseFile failed: %v", err)
	}
}

func TestParseFile_WithReferences(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "reference to variable",
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
				{Type: ast.TokenVar, Value: "var", Line: 3, Column: 4},
				{Type: ast.TokenIdentifier, Value: "p", Line: 3, Column: 8},
				{Type: ast.TokenColon, Value: ":", Line: 3, Column: 9},
				{Type: ast.TokenInt, Value: "int", Line: 3, Column: 11},
				{Type: ast.TokenEquals, Value: "=", Line: 3, Column: 15},
				{Type: ast.TokenIdentifier, Value: "x", Line: 3, Column: 17},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 2 {
					t.Fatalf("Expected 2 statements in function body, got %d", len(functionNode.Body))
				}
				assignNode := assertNodeType[ast.AssignmentNode](t, functionNode.Body[1], "ast.AssignmentNode")
				if len(assignNode.LValues) != 1 {
					t.Fatalf("Expected 1 left value, got %d", len(assignNode.LValues))
				}
				if assignNode.LValues[0].Ident.ID != "p" {
					t.Errorf("Expected variable name 'p', got %s", assignNode.LValues[0].Ident.ID)
				}
				varNode := assertNodeType[ast.VariableNode](t, assignNode.RValues[0], "ast.VariableNode")
				if varNode.Ident.ID != "x" {
					t.Errorf("Expected variable 'x', got %s", varNode.Ident.ID)
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
