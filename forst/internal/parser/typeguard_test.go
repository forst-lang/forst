package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile_WithTypeGuards(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic type guard",
			tokens: []ast.Token{
				{Type: ast.TokenIs, Value: "is", Line: 1, Column: 1},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 4},
				{Type: ast.TokenIdentifier, Value: "password", Line: 1, Column: 5},
				{Type: ast.TokenIdentifier, Value: "Password", Line: 1, Column: 15},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 22},
				{Type: ast.TokenIdentifier, Value: "Strong", Line: 1, Column: 24},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 30},
				{Type: ast.TokenEnsure, Value: "ensure", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "password", Line: 2, Column: 11},
				{Type: ast.TokenIs, Value: "is", Line: 2, Column: 14},
				{Type: ast.TokenIdentifier, Value: "Min", Line: 2, Column: 15},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 18},
				{Type: ast.TokenIntLiteral, Value: "12", Line: 2, Column: 19},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 22},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				typeGuardNode := assertNodeType[*ast.TypeGuardNode](t, nodes[0], "*ast.TypeGuardNode")
				if typeGuardNode.GetIdent() != "Strong" {
					t.Errorf("Expected type guard name 'Strong', got %s", typeGuardNode.GetIdent())
				}
				if len(typeGuardNode.Parameters()) != 1 {
					t.Errorf("Expected 1 parameter, got %d", len(typeGuardNode.Parameters()))
				}
				param, ok := typeGuardNode.Parameters()[0].(ast.SimpleParamNode)
				if !ok {
					t.Fatalf("Expected SimpleParamNode, got %T", typeGuardNode.Parameters()[0])
				}
				if param.GetIdent() != "password" {
					t.Errorf("Expected parameter name 'password', got %s", param.Ident.ID)
				}
				if param.Type.Ident != "Password" {
					t.Errorf("Expected parameter type 'Password', got %s", param.Type.Ident)
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
