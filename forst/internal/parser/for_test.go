package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseForLoops(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "for infinite",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenFor, Value: "for", Line: 2, Column: 4},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 8},
				{Type: ast.TokenBreak, Value: "break", Line: 3, Column: 6},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 5, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 5, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				fn := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
				if loop.IsRange || loop.Cond != nil || loop.Init != nil {
					t.Fatalf("expected bare infinite for")
				}
				if len(loop.Body) != 1 {
					t.Fatalf("body len %d", len(loop.Body))
				}
				assertNodeType[*ast.BreakNode](t, loop.Body[0], "*ast.BreakNode")
			},
		},
		{
			name: "for condition",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenFor, Value: "for", Line: 2, Column: 4},
				{Type: ast.TokenTrue, Value: "true", Line: 2, Column: 8},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 13},
				{Type: ast.TokenContinue, Value: "continue", Line: 3, Column: 6},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 5, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 5, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				fn := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
				if loop.IsRange || loop.Init != nil {
					t.Fatal("expected condition for")
				}
				assertNodeType[ast.BoolLiteralNode](t, loop.Cond, "ast.BoolLiteralNode")
				assertNodeType[*ast.ContinueNode](t, loop.Body[0], "*ast.ContinueNode")
			},
		},
		{
			name: "for condition single identifier is not a shape literal",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenFor, Value: "for", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "ok", Line: 2, Column: 8},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 11},
				{Type: ast.TokenBreak, Value: "break", Line: 3, Column: 6},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 5, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 5, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				fn := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
				v := assertNodeType[ast.VariableNode](t, loop.Cond, "ast.VariableNode")
				if v.Ident.ID != "ok" {
					t.Fatalf("cond ident: %s", v.Ident.ID)
				}
			},
		},
		{
			name: "for three clause",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenFor, Value: "for", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "i", Line: 2, Column: 8},
				{Type: ast.TokenColonEquals, Value: ":=", Line: 2, Column: 10},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 2, Column: 13},
				{Type: ast.TokenSemicolon, Value: ";", Line: 2, Column: 14},
				{Type: ast.TokenIdentifier, Value: "i", Line: 2, Column: 16},
				{Type: ast.TokenLess, Value: "<", Line: 2, Column: 18},
				{Type: ast.TokenIdentifier, Value: "n", Line: 2, Column: 20},
				{Type: ast.TokenSemicolon, Value: ";", Line: 2, Column: 21},
				{Type: ast.TokenIdentifier, Value: "i", Line: 2, Column: 23},
				{Type: ast.TokenPlusPlus, Value: "++", Line: 2, Column: 24},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 26},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				fn := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
				if loop.IsRange {
					t.Fatal("expected classic for")
				}
				assertNodeType[ast.AssignmentNode](t, loop.Init, "ast.AssignmentNode")
				assertNodeType[ast.UnaryExpressionNode](t, loop.Post, "ast.UnaryExpressionNode")
			},
		},
		{
			name: "for range two vars",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenFor, Value: "for", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "k", Line: 2, Column: 8},
				{Type: ast.TokenComma, Value: ",", Line: 2, Column: 9},
				{Type: ast.TokenIdentifier, Value: "v", Line: 2, Column: 11},
				{Type: ast.TokenColonEquals, Value: ":=", Line: 2, Column: 13},
				{Type: ast.TokenRange, Value: "range", Line: 2, Column: 16},
				{Type: ast.TokenIdentifier, Value: "m", Line: 2, Column: 22},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 24},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				fn := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
				if !loop.IsRange || !loop.RangeShort {
					t.Fatal("expected range :=")
				}
				if loop.RangeKey == nil || loop.RangeKey.ID != "k" || loop.RangeValue == nil || loop.RangeValue.ID != "v" {
					t.Fatalf("range vars %+v %+v", loop.RangeKey, loop.RangeValue)
				}
				assertNodeType[ast.VariableNode](t, loop.RangeX, "VariableNode")
			},
		},
		{
			name: "for range no vars",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenFor, Value: "for", Line: 2, Column: 4},
				{Type: ast.TokenRange, Value: "range", Line: 2, Column: 8},
				{Type: ast.TokenIdentifier, Value: "xs", Line: 2, Column: 14},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 17},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				fn := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
				if !loop.IsRange || loop.RangeKey != nil {
					t.Fatal("expected for range xs")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ast.SetupTestLogger(nil)
			p := setupParser(tt.tokens, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}
