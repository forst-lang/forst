package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
	}
	p := &Parser{
		tokens:  tokens,
		context: &Context{},
	}

	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	packageNode, ok := nodes[0].(ast.PackageNode)
	if !ok {
		t.Fatalf("Expected ast.PackageNode, got %T", nodes[0])
	}
	if packageNode.Id() != "main" {
		t.Fatalf("Expected package name 'main', got %s", packageNode.Id())
	}
}

func TestParseImport(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
		{Type: ast.TokenStringLiteral, Value: "fmt", Line: 1, Column: 8},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
	}
	p := &Parser{
		tokens:  tokens,
		context: &Context{},
	}

	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	importNode, ok := nodes[0].(ast.ImportNode)
	if !ok {
		t.Fatalf("Expected ast.ImportNode, got %T", nodes[0])
	}
	if importNode.Path != "fmt" {
		t.Fatalf("Expected import path 'fmt', got %s", importNode.Path)
	}
}

func TestParseStructTypeDef(t *testing.T) {
	t.Skip("Skipping struct type definition test")

	tokens := []ast.Token{
		{Type: ast.TokenType, Value: "type", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "Person", Line: 1, Column: 6},
		{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
		{Type: ast.TokenStruct, Value: "struct", Line: 1, Column: 14},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 20},
	}
	p := &Parser{
		tokens:  tokens,
		context: &Context{},
	}

	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	typeNode, ok := nodes[0].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Expected ast.TypeDefNode, got %T", nodes[0])
	}
	if typeNode.Ident != "Person" {
		t.Fatalf("Expected type name 'Person', got %s", typeNode.Ident)
	}
}

func TestParseBasicTypeDef(t *testing.T) {
	t.Skip("Skipping struct type definition test")

	tokens := []ast.Token{
		{Type: ast.TokenType, Value: "type", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "Username", Line: 1, Column: 6},
		{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
		{Type: ast.TokenStruct, Value: "String", Line: 1, Column: 14},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 20},
		{Type: ast.TokenIdentifier, Value: "Min", Line: 1, Column: 21},
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 24},
		{Type: ast.TokenIntLiteral, Value: "3", Line: 1, Column: 25},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 26},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 20},
	}
	p := &Parser{
		tokens:  tokens,
		context: &Context{},
	}

	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	typeNode, ok := nodes[0].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Expected ast.TypeDefNode, got %T", nodes[0])
	}
	if typeNode.Ident != "Person" {
		t.Fatalf("Expected type name 'Person', got %s", typeNode.Ident)
	}
}

func TestParseBasicFunction(t *testing.T) {
	tokens := []ast.Token{
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
	}
	p := &Parser{
		tokens:  tokens,
		context: &Context{},
	}

	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	functionNode, ok := nodes[0].(ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected ast.FunctionNode, got %T", nodes[0])
	}
	if functionNode.Id() != "main" {
		t.Fatalf("Expected function name 'main', got %s", functionNode.Id())
	}
}

func TestParseEnsureFunction(t *testing.T) {
	tokens := []ast.Token{
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
	}
	p := &Parser{
		tokens:  tokens,
		context: &Context{},
	}

	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	functionNode, ok := nodes[0].(ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected ast.FunctionNode, got %T", nodes[0])
	}

	if functionNode.Id() != "main" {
		t.Fatalf("Expected function name 'main', got %s", functionNode.Id())
	}

	if len(functionNode.Body) != 1 {
		t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
	}

	ensureNode, ok := functionNode.Body[0].(ast.EnsureNode)
	if !ok {
		t.Fatalf("Expected ast.EnsureNode, got %T", functionNode.Body[0])
	}

	if ensureNode.Variable.Id() != "x" {
		t.Fatalf("Expected variable 'x', got %s", ensureNode.Variable.Id())
	}

	if len(ensureNode.Assertion.Constraints) != 1 {
		t.Fatalf("Expected 1 constraint, got %d", len(ensureNode.Assertion.Constraints))
	}

	constraint := ensureNode.Assertion.Constraints[0]
	if constraint.Name != "Min" {
		t.Fatalf("Expected constraint 'Min', got %s", constraint.Name)
	}

	if len(constraint.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(constraint.Args))
	}

	arg := constraint.Args[0]
	if arg.Value == nil {
		t.Fatalf("Expected value argument, got nil")
	}

	value := *arg.Value
	if value.String() != "0" {
		t.Fatalf("Expected value '0', got %s", value.String())
	}
}
