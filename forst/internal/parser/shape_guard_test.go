package parser

import (
	"forst/internal/ast"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseShapeGuardAssertion(t *testing.T) {
	// Test the shape guard assertion: AppMutation.Input({input: {name: String}})

	tokens := []ast.Token{
		{Type: ast.TokenFunc, Value: "func"},
		{Type: ast.TokenIdentifier, Value: "createTask"},
		{Type: ast.TokenLParen, Value: "("},
		{Type: ast.TokenIdentifier, Value: "op"},
		{Type: ast.TokenIdentifier, Value: "AppMutation"},
		{Type: ast.TokenDot, Value: "."},
		{Type: ast.TokenIdentifier, Value: "Input"},
		{Type: ast.TokenLParen, Value: "("},
		{Type: ast.TokenLBrace, Value: "{"},
		{Type: ast.TokenIdentifier, Value: "input"},
		{Type: ast.TokenColon, Value: ":"},
		{Type: ast.TokenLBrace, Value: "{"},
		{Type: ast.TokenIdentifier, Value: "name"},
		{Type: ast.TokenColon, Value: ":"},
		{Type: ast.TokenIdentifier, Value: "String"},
		{Type: ast.TokenRBrace, Value: "}"},
		{Type: ast.TokenRBrace, Value: "}"},
		{Type: ast.TokenRParen, Value: ")"},
		{Type: ast.TokenRParen, Value: ")"},
		{Type: ast.TokenLBrace, Value: "{"},
		{Type: ast.TokenReturn, Value: "return"},
		{Type: ast.TokenIdentifier, Value: "op"},
		{Type: ast.TokenDot, Value: "."},
		{Type: ast.TokenIdentifier, Value: "input"},
		{Type: ast.TokenDot, Value: "."},
		{Type: ast.TokenIdentifier, Value: "name"},
		{Type: ast.TokenRBrace, Value: "}"},
		{Type: ast.TokenEOF, Value: ""},
	}

	p := setupParserForShapeGuard(tokens)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	functionNode := assertNodeTypeForShapeGuard[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
	if len(functionNode.Params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(functionNode.Params))
	}

	param := functionNode.Params[0]
	if param.GetIdent() != "op" {
		t.Errorf("Expected parameter name 'op', got %s", param.GetIdent())
	}

	paramType := param.GetType()
	if paramType.Ident != ast.TypeAssertion {
		t.Errorf("Expected parameter type to be assertion, got %s", paramType.Ident)
	}

	if paramType.Assertion == nil {
		t.Fatal("Expected assertion in parameter type")
	}

	assertion := paramType.Assertion
	if assertion.BaseType == nil || *assertion.BaseType != "AppMutation" {
		t.Errorf("Expected base type 'AppMutation', got %v", assertion.BaseType)
	}

	if len(assertion.Constraints) != 1 {
		t.Fatalf("Expected 1 constraint, got %d", len(assertion.Constraints))
	}

	constraint := assertion.Constraints[0]
	if constraint.Name != "Input" {
		t.Errorf("Expected constraint name 'Input', got %s", constraint.Name)
	}

	if len(constraint.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(constraint.Args))
	}

	arg := constraint.Args[0]
	if arg.Shape == nil {
		t.Fatal("Expected shape argument")
	}

	t.Logf("DEBUG: Shape argument structure:")
	t.Logf("  Shape: %+v", arg.Shape)
	t.Logf("  Fields: %+v", arg.Shape.Fields)

	if len(arg.Shape.Fields) != 1 {
		t.Fatalf("Expected 1 field in shape argument, got %d", len(arg.Shape.Fields))
	}

	inputField, hasInput := arg.Shape.Fields["input"]
	if !hasInput {
		t.Fatal("Expected 'input' field in shape argument")
	}

	t.Logf("DEBUG: Input field structure:")
	t.Logf("  InputField: %+v", inputField)
	t.Logf("  InputField.Type: %+v", inputField.Type)
	t.Logf("  InputField.Shape: %+v", inputField.Shape)
	t.Logf("  InputField.Assertion: %+v", inputField.Assertion)

	if inputField.Shape == nil {
		t.Fatal("Expected nested shape for input field")
	}

	if len(inputField.Shape.Fields) != 1 {
		t.Fatalf("Expected 1 field in nested shape, got %d", len(inputField.Shape.Fields))
	}

	nameField, hasName := inputField.Shape.Fields["name"]
	if !hasName {
		t.Fatal("Expected 'name' field in nested shape")
	}

	t.Logf("DEBUG: Name field structure:")
	t.Logf("  NameField: %+v", nameField)
	t.Logf("  NameField.Type: %+v", nameField.Type)
	t.Logf("  NameField.Shape: %+v", nameField.Shape)
	t.Logf("  NameField.Assertion: %+v", nameField.Assertion)

	if nameField.Type == nil || nameField.Type.Ident != ast.TypeIdent("String") {
		t.Errorf("Expected name field to be String type, got %v", nameField.Type)
	}

	t.Logf("Successfully parsed shape guard assertion:")
	t.Logf("  BaseType: %s", *assertion.BaseType)
	t.Logf("  Constraint: %s", constraint.Name)
	t.Logf("  Shape argument: %+v", arg.Shape)
}

func setupParserForShapeGuard(tokens []ast.Token) *Parser {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return New(tokens, "test.ft", logger)
}

func assertNodeTypeForShapeGuard[T ast.Node](t *testing.T, node ast.Node, expectedType string) T {
	if n, ok := node.(T); ok {
		return n
	}
	t.Fatalf("Expected node type %s, got %T", expectedType, node)
	return *new(T)
}
