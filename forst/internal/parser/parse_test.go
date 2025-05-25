package parser

import (
	"forst/internal/ast"
	"testing"
)

// setupParser creates a new parser with the given tokens
func setupParser(tokens []ast.Token) *Parser {
	return &Parser{
		tokens:  tokens,
		context: &Context{},
	}
}

// assertNodeType checks if a node is of the expected type
func assertNodeType[T ast.Node](t *testing.T, node ast.Node, expectedType string) T {
	t.Helper()
	typedNode, ok := node.(T)
	if !ok {
		t.Fatalf("Expected %s, got %T", expectedType, node)
	}
	return typedNode
}

func TestParseFile_WithPackageDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "package declaration",
			tokens: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				packageNode := assertNodeType[ast.PackageNode](t, nodes[0], "ast.PackageNode")
				if packageNode.Id() != "main" {
					t.Errorf("Expected package name 'main', got %s", packageNode.Id())
				}
			},
		},
		{
			name: "import declaration",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: "fmt", Line: 1, Column: 8},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				importNode := assertNodeType[ast.ImportNode](t, nodes[0], "ast.ImportNode")
				if importNode.Path != "fmt" {
					t.Errorf("Expected import path 'fmt', got %s", importNode.Path)
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

func TestParseFile_WithTypeDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic assertion type definition",
			tokens: []ast.Token{
				{Type: ast.TokenType, Value: "type", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "Username", Line: 1, Column: 6},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
				{Type: ast.TokenString, Value: "String", Line: 1, Column: 14},
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 20},
				{Type: ast.TokenIdentifier, Value: "Min", Line: 1, Column: 21},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 24},
				{Type: ast.TokenIntLiteral, Value: "3", Line: 1, Column: 25},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 26},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 20},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				typeNode := assertNodeType[ast.TypeDefNode](t, nodes[0], "ast.TypeDefNode")
				if typeNode.Ident != "Username" {
					t.Errorf("Expected type name 'Username', got %s", typeNode.Ident)
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

func TestParseFile_WithFunctions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic function with parameter",
			tokens: []ast.Token{
				{Type: ast.TokenFunction, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenInt, Value: "Int", Line: 1, Column: 14},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 17},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 19},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "1", Line: 2, Column: 11},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if functionNode.Id() != "main" {
					t.Errorf("Expected function name 'main', got %s", functionNode.Id())
				}
				if len(functionNode.Body) != 1 {
					t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
			},
		},
		{
			name: "function with ensure statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunction, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenInt, Value: "Int", Line: 1, Column: 14},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 17},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 19},
				{Type: ast.TokenEnsure, Value: "ensure", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 11},
				{Type: ast.TokenIs, Value: "is", Line: 2, Column: 13},
				{Type: ast.TokenInt, Value: "Int", Line: 2, Column: 15},
				{Type: ast.TokenDot, Value: ".", Line: 2, Column: 18},
				{Type: ast.TokenIdentifier, Value: "Min", Line: 2, Column: 19},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 22},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 2, Column: 23},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 24},
				{Type: ast.TokenOr, Value: "or", Line: 2, Column: 26},
				{Type: ast.TokenIdentifier, Value: "TooSmall", Line: 2, Column: 28},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 36},
				{Type: ast.TokenStringLiteral, Value: "x must be at least 0", Line: 2, Column: 37},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 51},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if functionNode.Id() != "main" {
					t.Errorf("Expected function name 'main', got %s", functionNode.Id())
				}
				if len(functionNode.Body) != 1 {
					t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}

				ensureNode := assertNodeType[ast.EnsureNode](t, functionNode.Body[0], "ast.EnsureNode")
				if ensureNode.Variable.Id() != "x" {
					t.Errorf("Expected variable 'x', got %s", ensureNode.Variable.Id())
				}

				if len(ensureNode.Assertion.Constraints) != 1 {
					t.Errorf("Expected 1 constraint, got %d", len(ensureNode.Assertion.Constraints))
				}

				constraint := ensureNode.Assertion.Constraints[0]
				if constraint.Name != "Min" {
					t.Errorf("Expected constraint 'Min', got %s", constraint.Name)
				}

				if len(constraint.Args) != 1 {
					t.Errorf("Expected 1 argument, got %d", len(constraint.Args))
				}

				arg := constraint.Args[0]
				if arg.Value == nil {
					t.Fatal("Expected value argument, got nil")
				}

				value := *arg.Value
				if value.String() != "0" {
					t.Errorf("Expected value '0', got %s", value.String())
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
