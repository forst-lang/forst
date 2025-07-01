package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile_WithControlFlow(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "if statement with init",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenIf, Value: "if", Line: 2, Column: 4},
				{Type: ast.TokenVar, Value: "var", Line: 2, Column: 7},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 11},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 12},
				{Type: ast.TokenInt, Value: "int", Line: 2, Column: 14},
				{Type: ast.TokenEquals, Value: "=", Line: 2, Column: 18},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 2, Column: 20},
				{Type: ast.TokenSemicolon, Value: ";", Line: 2, Column: 22},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 24},
				{Type: ast.TokenGreater, Value: ">", Line: 2, Column: 26},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 2, Column: 28},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 30},
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
				if ifNode.Init == nil {
					t.Fatal("Expected initialization statement")
				}
				initNode := assertNodeType[ast.AssignmentNode](t, ifNode.Init, "ast.AssignmentNode")
				if len(initNode.LValues) != 1 {
					t.Fatalf("Expected 1 left value, got %d", len(initNode.LValues))
				}
				if initNode.LValues[0].Ident.ID != "x" {
					t.Errorf("Expected init variable 'x', got %s", initNode.LValues[0].Ident.ID)
				}
				condNode := assertNodeType[ast.BinaryExpressionNode](t, ifNode.Condition, "ast.BinaryExpressionNode")
				if condNode.Operator != ast.TokenGreater {
					t.Errorf("Expected operator '>', got %s", condNode.Operator)
				}
				if len(ifNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in if body, got %d", len(ifNode.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, ifNode.Body[0], "ast.ReturnNode")
				varNode := assertNodeType[ast.VariableNode](t, returnNode.Value, "ast.VariableNode")
				if varNode.Ident.ID != "x" {
					t.Errorf("Expected return value 'x', got %s", varNode.Ident.ID)
				}
			},
		},
		{
			name: "if statement with else-if and else",
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
				{Type: ast.TokenIdentifier, Value: "1", Line: 3, Column: 15},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 4},
				{Type: ast.TokenElse, Value: "else", Line: 4, Column: 6},
				{Type: ast.TokenIf, Value: "if", Line: 4, Column: 11},
				{Type: ast.TokenIdentifier, Value: "x", Line: 4, Column: 14},
				{Type: ast.TokenLess, Value: "<", Line: 4, Column: 16},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 4, Column: 18},
				{Type: ast.TokenLBrace, Value: "{", Line: 4, Column: 20},
				{Type: ast.TokenReturn, Value: "return", Line: 5, Column: 8},
				{Type: ast.TokenIntLiteral, Value: "-1", Line: 5, Column: 15},
				{Type: ast.TokenRBrace, Value: "}", Line: 6, Column: 4},
				{Type: ast.TokenElse, Value: "else", Line: 6, Column: 6},
				{Type: ast.TokenLBrace, Value: "{", Line: 6, Column: 11},
				{Type: ast.TokenReturn, Value: "return", Line: 7, Column: 8},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 7, Column: 15},
				{Type: ast.TokenRBrace, Value: "}", Line: 8, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 9, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 9, Column: 2},
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
				if len(ifNode.ElseIfs) != 1 {
					t.Fatalf("Expected 1 else-if block, got %d", len(ifNode.ElseIfs))
				}
				if ifNode.Else == nil {
					t.Fatal("Expected else block")
				}
				// Check main if condition
				condNode := assertNodeType[ast.BinaryExpressionNode](t, ifNode.Condition, "ast.BinaryExpressionNode")
				if condNode.Operator != ast.TokenGreater {
					t.Errorf("Expected operator '>', got %s", condNode.Operator)
				}
				// Check else-if condition
				elseIfNode := ifNode.ElseIfs[0]
				condNode = assertNodeType[ast.BinaryExpressionNode](t, elseIfNode.Condition, "ast.BinaryExpressionNode")
				if condNode.Operator != ast.TokenLess {
					t.Errorf("Expected operator '<', got %s", condNode.Operator)
				}
				// Check else block
				if len(ifNode.Else.Body) != 1 {
					t.Fatalf("Expected 1 statement in else block, got %d", len(ifNode.Else.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, ifNode.Else.Body[0], "ast.ReturnNode")
				value := assertNodeType[ast.IntLiteralNode](t, returnNode.Value, "ast.IntLiteralNode")
				if value.Value != 0 {
					t.Errorf("Expected return value 0, got %d", value.Value)
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
