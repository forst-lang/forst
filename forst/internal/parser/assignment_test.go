package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseAssignment(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "simple assignment",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 4},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 6},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 8},
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
				if assignNode.IsShort {
					t.Error("Expected regular assignment, got short assignment")
				}
			},
		},
		{
			name: "assignment with type annotation",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 4},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 5},
				{Type: ast.TokenInt, Value: "int", Line: 2, Column: 7},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 11},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 13},
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
				if assignNode.LValues[0].ExplicitType.Ident != ast.TypeInt {
					t.Errorf("Expected type 'int', got %s", assignNode.LValues[0].ExplicitType.Ident)
				}
			},
		},
		{
			name: "short assignment",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 4},
				{Type: ast.TokenColonEquals, Value: ":=", Line: 2, Column: 6},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 9},
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
				if !assignNode.IsShort {
					t.Error("Expected short assignment, got regular assignment")
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

func TestParseMultipleAssignment(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "multiple assignment",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 4},
				{Type: ast.TokenComma, Value: ",", Line: 2, Column: 5},
				{Type: ast.TokenIdentifier, Value: "y", Line: 2, Column: 7},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 9},
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
				assignNode := assertNodeType[ast.AssignmentNode](t, functionNode.Body[0], "ast.AssignmentNode")
				if len(assignNode.LValues) != 2 {
					t.Fatalf("Expected 2 left values, got %d", len(assignNode.LValues))
				}
				if assignNode.LValues[0].Ident.ID != "x" {
					t.Errorf("Expected first variable name 'x', got %s", assignNode.LValues[0].Ident.ID)
				}
				if assignNode.LValues[1].Ident.ID != "y" {
					t.Errorf("Expected second variable name 'y', got %s", assignNode.LValues[1].Ident.ID)
				}
				if len(assignNode.RValues) != 2 {
					t.Fatalf("Expected 2 right values, got %d", len(assignNode.RValues))
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
