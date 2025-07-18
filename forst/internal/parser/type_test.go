package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseType(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic types",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 14},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 18},
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
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypeInt {
					t.Errorf("Expected return type 'int', got %s", functionNode.ReturnTypes[0].Ident)
				}
			},
		},
		{
			name: "pointer type",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenStar, Value: "*", Line: 1, Column: 14},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 16},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 20},
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
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypePointer {
					t.Errorf("Expected return type '*int', got %s", functionNode.ReturnTypes[0].Ident)
				}
				if len(functionNode.ReturnTypes[0].TypeParams) != 1 {
					t.Fatalf("Expected 1 type parameter for pointer, got %d", len(functionNode.ReturnTypes[0].TypeParams))
				}
				if functionNode.ReturnTypes[0].TypeParams[0].Ident != ast.TypeInt {
					t.Errorf("Expected pointer base type 'int', got %s", functionNode.ReturnTypes[0].TypeParams[0].Ident)
				}
			},
		},
		{
			name: "slice type",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenLBracket, Value: "[", Line: 1, Column: 14},
				{Type: ast.TokenRBracket, Value: "]", Line: 1, Column: 15},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 17},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 21},
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
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypeArray {
					t.Errorf("Expected return type '[]int', got %s", functionNode.ReturnTypes[0].Ident)
				}
				if len(functionNode.ReturnTypes[0].TypeParams) != 1 {
					t.Fatalf("Expected 1 type parameter for slice, got %d", len(functionNode.ReturnTypes[0].TypeParams))
				}
				if functionNode.ReturnTypes[0].TypeParams[0].Ident != ast.TypeInt {
					t.Errorf("Expected slice element type 'int', got %s", functionNode.ReturnTypes[0].TypeParams[0].Ident)
				}
			},
		},
		{
			name: "map type",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenMap, Value: "map", Line: 1, Column: 14},
				{Type: ast.TokenLBracket, Value: "[", Line: 1, Column: 18},
				{Type: ast.TokenString, Value: "string", Line: 1, Column: 19},
				{Type: ast.TokenRBracket, Value: "]", Line: 1, Column: 25},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 27},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 31},
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
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypeMap {
					t.Errorf("Expected return type 'map[string]int', got %s", functionNode.ReturnTypes[0].Ident)
				}
				if len(functionNode.ReturnTypes[0].TypeParams) != 2 {
					t.Fatalf("Expected 2 type parameters for map, got %d", len(functionNode.ReturnTypes[0].TypeParams))
				}
				if functionNode.ReturnTypes[0].TypeParams[0].Ident != ast.TypeString {
					t.Errorf("Expected map key type 'string', got %s", functionNode.ReturnTypes[0].TypeParams[0].Ident)
				}
				if functionNode.ReturnTypes[0].TypeParams[1].Ident != ast.TypeInt {
					t.Errorf("Expected map value type 'int', got %s", functionNode.ReturnTypes[0].TypeParams[1].Ident)
				}
			},
		},
		{
			name: "interface type",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenInterface, Value: "interface", Line: 1, Column: 14},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 23},
				{Type: ast.TokenRBrace, Value: "}", Line: 1, Column: 24},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 26},
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
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypeObject {
					t.Errorf("Expected return type 'interface{}', got %s", functionNode.ReturnTypes[0].Ident)
				}
			},
		},
		{
			name: "custom type",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenIdentifier, Value: "MyType", Line: 1, Column: 14},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 21},
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
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypeIdent("MyType") {
					t.Errorf("Expected return type 'MyType', got %s", functionNode.ReturnTypes[0].Ident)
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
