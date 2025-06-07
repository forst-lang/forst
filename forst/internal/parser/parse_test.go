package parser

import (
	"forst/internal/ast"
	"testing"

	"github.com/sirupsen/logrus"
)

// setupParser creates a new parser with the given tokens
func setupParser(tokens []ast.Token) *Parser {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
	})
	return NewParser(tokens, "", logger)
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
				if packageNode.GetIdent() != "main" {
					t.Errorf("Expected package name 'main', got '%s'", packageNode.GetIdent())
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
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "abc123", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 10},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 14},
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
				if functionNode.GetIdent() != "abc123" {
					t.Errorf("Expected function name 'abc123', got %s", functionNode.Ident.ID)
				}
				if len(functionNode.Body) != 1 {
					t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
			},
		},
		{
			name: "function with ensure statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 10},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 14},
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
				if functionNode.GetIdent() != "main" {
					t.Errorf("Expected function name 'main', got %s", functionNode.GetIdent())
				}
				if len(functionNode.Body) != 1 {
					t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}

				ensureNode := assertNodeType[ast.EnsureNode](t, functionNode.Body[0], "ast.EnsureNode")
				if ensureNode.Variable.GetIdent() != "x" {
					t.Errorf("Expected variable name 'x', got '%s'", ensureNode.Variable.GetIdent())
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
		if _, ok := returnNode.Value.(ast.BinaryExpressionNode); !ok {
			t.Errorf("Expected return value to be a BinaryExpressionNode, got %T", returnNode.Value)
		}
	} else {
		t.Fatalf("ParseFile failed: %v", err)
	}
}

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
			p := setupParser(tt.tokens)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
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
