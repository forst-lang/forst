package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseShape(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "shape literal",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 11},
				{Type: ast.TokenIdentifier, Value: "name", Line: 3, Column: 8},
				{Type: ast.TokenColon, Value: ":", Line: 3, Column: 12},
				{Type: ast.TokenStringLiteral, Value: "John", Line: 3, Column: 14},
				{Type: ast.TokenComma, Value: ",", Line: 3, Column: 19},
				{Type: ast.TokenIdentifier, Value: "age", Line: 4, Column: 8},
				{Type: ast.TokenColon, Value: ":", Line: 4, Column: 11},
				{Type: ast.TokenIntLiteral, Value: "30", Line: 4, Column: 13},
				{Type: ast.TokenRBrace, Value: "}", Line: 5, Column: 4},
				{Type: ast.TokenRBrace, Value: "}", Line: 6, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 6, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				returnNode := assertNodeType[ast.ReturnNode](t, functionNode.Body[0], "ast.ReturnNode")
				if len(returnNode.Values) != 1 {
					t.Fatal("Expected exactly one return value")
				}
				shapeNode := assertNodeType[ast.ShapeNode](t, returnNode.Values[0], "ast.ShapeNode")
				if len(shapeNode.Fields) != 2 {
					t.Fatalf("Expected 2 fields in shape, got %d", len(shapeNode.Fields))
				}
			},
		},
		{
			name: "shape type",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 12},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 14},
				{Type: ast.TokenIdentifier, Value: "name", Line: 2, Column: 4},
				{Type: ast.TokenColon, Value: ":", Line: 2, Column: 8},
				{Type: ast.TokenString, Value: "string", Line: 2, Column: 10},
				{Type: ast.TokenComma, Value: ",", Line: 2, Column: 16},
				{Type: ast.TokenIdentifier, Value: "age", Line: 3, Column: 4},
				{Type: ast.TokenColon, Value: ":", Line: 3, Column: 7},
				{Type: ast.TokenInt, Value: "int", Line: 3, Column: 9},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenLBrace, Value: "{", Line: 4, Column: 3},
				{Type: ast.TokenReturn, Value: "return", Line: 5, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 5, Column: 11},
				{Type: ast.TokenRBrace, Value: "}", Line: 6, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 6, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.ReturnTypes) != 1 {
					t.Fatalf("Expected 1 return type, got %d", len(functionNode.ReturnTypes))
				}
				if functionNode.ReturnTypes[0].Ident != ast.TypeShape {
					t.Errorf("Expected return type 'shape', got %s", functionNode.ReturnTypes[0].Ident)
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
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}

func TestParseShapeType_preservesFieldOrder(t *testing.T) {
	input := `{ id: String, expiresAt: Int }`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()
	if len(shape.FieldOrder) != 2 {
		t.Fatalf("FieldOrder = %v", shape.FieldOrder)
	}
	if shape.FieldOrder[0] != "id" || shape.FieldOrder[1] != "expiresAt" {
		t.Fatalf("FieldOrder = %v, want [id expiresAt]", shape.FieldOrder)
	}
}

func TestParseShapeType_TopLevel(t *testing.T) {
	input := `{ foo: String, bar: Int }`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()
	if len(shape.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(shape.Fields))
	}
	if shape.Fields["foo"].Type == nil || shape.Fields["foo"].Type.Ident != ast.TypeString {
		t.Errorf("expected foo to be String type, got %+v", shape.Fields["foo"].Type)
	}
	if shape.Fields["bar"].Type == nil || shape.Fields["bar"].Type.Ident != ast.TypeInt {
		t.Errorf("expected bar to be Int type, got %+v", shape.Fields["bar"].Type)
	}
}

func TestParseShapeType_Nested(t *testing.T) {
	input := `{ input: { name: String, age: Int } }`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()
	inputField := shape.Fields["input"]
	if inputField.Type == nil || inputField.Type.Ident != ast.TypeShape {
		t.Fatalf("expected input to be shape type, got %+v", inputField.Type)
	}
	if inputField.Type.Assertion == nil || len(inputField.Type.Assertion.Constraints) == 0 {
		t.Fatalf("expected assertion for nested shape, got %+v", inputField.Type.Assertion)
	}
	nested := inputField.Type.Assertion.Constraints[0].Args[0].Shape
	if nested == nil || len(nested.Fields) != 2 {
		t.Fatalf("expected 2 fields in nested shape, got %+v", nested)
	}
}

func TestParseShapeLiteral_TopLevel(t *testing.T) {
	input := `{ foo: 42, bar: "baz" }`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	shape := p.parseShapeLiteral(ShapeLiteralOpts{})
	if len(shape.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(shape.Fields))
	}
}

func TestParseShapeLiteral_Nested(t *testing.T) {
	input := `{ input: { name: "Alice" } }`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	shape := p.parseShapeLiteral(ShapeLiteralOpts{})
	inputField := shape.Fields["input"]
	if inputField.Shape == nil {
		t.Fatalf("expected input to be a nested shape literal, got %+v", inputField)
	}
	if len(inputField.Shape.Fields) != 1 {
		t.Fatalf("expected 1 field in nested shape, got %+v", inputField.Shape)
	}
}

func TestParseShapeType_AsFunctionParam(t *testing.T) {
	input := `func foo(arg { x: Int, y: String }) {}`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	fn := p.parseFunctionDefinition()
	if len(fn.Params) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Params))
	}
	param := fn.Params[0]
	if param.GetType().Ident != ast.TypeShape {
		t.Fatalf("expected parameter to be shape type, got %+v", param.GetType())
	}
}

func TestParseShapeType_FieldAssertionBaseType(t *testing.T) {
	t.Parallel()

	p := NewTestParser(`{ age: Int.Min(1) }`, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()
	field := shape.Fields["age"]
	if field.Type == nil || field.Type.Ident != ast.TypeAssertion {
		t.Fatalf("age field type = %+v", field.Type)
	}
	if field.Type.Assertion == nil || field.Type.Assertion.BaseType == nil {
		t.Fatalf("missing assertion base type: %+v", field.Type.Assertion)
	}
	if *field.Type.Assertion.BaseType != ast.TypeInt {
		t.Fatalf("base type = %v, want Int", *field.Type.Assertion.BaseType)
	}
	if len(field.Type.Assertion.Constraints) != 1 || field.Type.Assertion.Constraints[0].Name != "Min" {
		t.Fatalf("unexpected constraints: %+v", field.Type.Assertion.Constraints)
	}
}

func TestParseShapeType_ErrorMethodNameAllowed(t *testing.T) {
	t.Parallel()

	p := NewTestParser(`{ error(msg String): Int }`, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()
	field, ok := shape.Fields["error"]
	if !ok {
		t.Fatalf("missing error method field: %+v", shape.Fields)
	}
	if !field.IsMethod {
		t.Fatalf("expected error field to parse as method, got %+v", field)
	}
	if len(field.MethodParams) != 1 {
		t.Fatalf("expected 1 method param, got %d", len(field.MethodParams))
	}
	if len(field.MethodReturnTypes) != 1 || field.MethodReturnTypes[0].Ident != ast.TypeInt {
		t.Fatalf("unexpected method return types: %+v", field.MethodReturnTypes)
	}
}

func TestParseShapeTypeField_Branches(t *testing.T) {
	t.Parallel()

	p := NewTestParser(`{
		method(msg String): Int,
		typed: Int,
		ptr: *Int,
		nested {
			child: Int
		},
		plain
	}`, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()

	if !shape.Fields["method"].IsMethod {
		t.Fatalf("method field should be parsed as method: %+v", shape.Fields["method"])
	}
	if shape.Fields["typed"].Type == nil || shape.Fields["typed"].Type.Ident != ast.TypeInt {
		t.Fatalf("typed field should be Int: %+v", shape.Fields["typed"].Type)
	}
	if shape.Fields["ptr"].Type == nil || shape.Fields["ptr"].Type.Ident != ast.TypePointer {
		t.Fatalf("ptr field should be pointer Int: %+v", shape.Fields["ptr"].Type)
	}
	if got := shape.Fields["ptr"].Type.TypeParams; len(got) != 1 || got[0].Ident != ast.TypeInt {
		t.Fatalf("ptr field type params = %+v", got)
	}
	if shape.Fields["nested"].Type == nil || shape.Fields["nested"].Type.Ident != ast.TypeShape {
		t.Fatalf("nested field should be shape type: %+v", shape.Fields["nested"].Type)
	}
	if shape.Fields["plain"].Type == nil || shape.Fields["plain"].Type.Ident != "plain" {
		t.Fatalf("plain shorthand field type = %+v", shape.Fields["plain"].Type)
	}
}
