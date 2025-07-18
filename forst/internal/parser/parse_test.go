package parser

import (
	"forst/internal/ast"
	"reflect"
	"testing"
)

func TestParseFile(t *testing.T) {
	logger := ast.SetupTestLogger()

	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "empty file",
			tokens: []ast.Token{
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 1},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 0 {
					t.Fatalf("Expected 0 nodes for empty file, got %d", len(nodes))
				}
			},
		},
		{
			name: "file with comments only",
			tokens: []ast.Token{
				{Type: ast.TokenComment, Value: "// This is a comment", Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 2, Column: 1},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 0 {
					t.Fatalf("Expected 0 nodes for file with comments only, got %d", len(nodes))
				}
			},
		},
		{
			name: "file with package and function",
			tokens: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
				{Type: ast.TokenFunc, Value: "func", Line: 2, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 2, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 2, Column: 12},
				{Type: ast.TokenReturn, Value: "return", Line: 3, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 3, Column: 11},
				{Type: ast.TokenRBrace, Value: "}", Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 4, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(nodes))
				}
				packageNode := assertNodeType[ast.PackageNode](t, nodes[0], "ast.PackageNode")
				if packageNode.Ident.ID != "main" {
					t.Errorf("Expected package name 'main', got %s", packageNode.Ident.ID)
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
				if functionNode.Ident.ID != "main" {
					t.Errorf("Expected function name 'main', got %s", functionNode.Ident.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}

func TestParseFile_WithUnexpectedToken(t *testing.T) {
	logger := ast.SetupTestLogger()

	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "unexpected", Line: 1, Column: 1},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 11},
	}

	p := setupParser(tokens, logger)
	_, err := p.ParseFile()
	if err == nil {
		t.Fatal("Expected parse error for unexpected token, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected ParseError, got %T", err)
	}

	if parseErr.Token.Value != "unexpected" {
		t.Errorf("Expected error token 'unexpected', got %s", parseErr.Token.Value)
	}
}

func TestParseFile_ShapeLiteralInReturn(t *testing.T) {
	logger := ast.SetupTestLogger()

	input := `
package user

type User = {
  id: String,
  name: String,
  age: Int,
  email: String
}

type CreateUserResponse = {
  user: User,
  created_at: Int
}

func CreateUser(input CreateUserRequest) {
  user := {
    id: "123",
    name: input.name,
    age: input.age,
    email: input.email
  }
  
  return {
    user: user,
    created_at: 1234567890
  }, nil
}
`

	p := NewTestParser(input, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Find the function node
	var functionNode *ast.FunctionNode
	for _, node := range nodes {
		if fn, ok := node.(*ast.FunctionNode); ok && fn.GetIdent() == "CreateUser" {
			functionNode = fn
			break
		}
		// Handle value type
		if fnVal, ok := node.(ast.FunctionNode); ok && fnVal.GetIdent() == "CreateUser" {
			functionNode = &fnVal
			break
		}
	}

	if functionNode == nil {
		t.Fatal("Could not find CreateUser function")
	}

	// Find the return statement
	var returnStmt *ast.ReturnNode
	for _, stmt := range functionNode.Body {
		if ret, ok := stmt.(*ast.ReturnNode); ok {
			returnStmt = ret
			break
		}
		// Handle value type
		if retVal, ok := stmt.(ast.ReturnNode); ok {
			returnStmt = &retVal
			break
		}
	}

	if returnStmt == nil {
		t.Fatal("Could not find return statement")
	}

	// Check that the return statement has the expected structure
	if len(returnStmt.Values) != 2 {
		t.Fatalf("Expected 2 return values, got %d", len(returnStmt.Values))
	}

	// Check the first return value (the shape literal)
	firstValue := returnStmt.Values[0]
	var shapeNode *ast.ShapeNode
	if sn, ok := firstValue.(*ast.ShapeNode); ok {
		shapeNode = sn
	} else if snVal, ok := firstValue.(ast.ShapeNode); ok {
		shapeNode = &snVal
	} else {
		t.Fatalf("Expected first return value to be ShapeNode, got %T", firstValue)
	}

	// The shape should have exactly 2 fields: user and created_at
	if len(shapeNode.Fields) != 2 {
		t.Fatalf("Expected shape to have 2 fields, got %d", len(shapeNode.Fields))
	}

	// Check field names
	fieldNames := make(map[string]bool)
	for fieldName := range shapeNode.Fields {
		fieldNames[fieldName] = true
	}

	expectedFields := map[string]bool{
		"user":       true,
		"created_at": true,
	}

	if !reflect.DeepEqual(fieldNames, expectedFields) {
		t.Errorf("Expected fields %v, got %v", expectedFields, fieldNames)
	}

	// Check that the 'user' field contains a variable reference, not shape fields
	userField, exists := shapeNode.Fields["user"]
	if !exists {
		t.Fatal("Expected 'user' field to exist")
	}

	// The user field should contain a VariableNode in its Node field
	var userValue *ast.VariableNode
	if uv, ok := userField.Node.(*ast.VariableNode); ok {
		userValue = uv
	} else if uvVal, ok := userField.Node.(ast.VariableNode); ok {
		userValue = &uvVal
	} else {
		t.Fatalf("Expected user field Node to be VariableNode, got %T", userField.Node)
	}

	if userValue.Ident.ID != "user" {
		t.Errorf("Expected user field to reference variable 'user', got %s", userValue.Ident.ID)
	}

	// Check the created_at field
	createdAtField, exists := shapeNode.Fields["created_at"]
	if !exists {
		t.Fatal("Expected 'created_at' field to exist")
	}

	// The created_at field should contain an integer literal in its Node field
	var createdAtValue *ast.IntLiteralNode
	if cav, ok := createdAtField.Node.(*ast.IntLiteralNode); ok {
		createdAtValue = cav
	} else if cavVal, ok := createdAtField.Node.(ast.IntLiteralNode); ok {
		createdAtValue = &cavVal
	} else {
		t.Fatalf("Expected created_at field Node to be IntLiteralNode, got %T", createdAtField.Node)
	}

	if createdAtValue.Value != 1234567890 {
		t.Errorf("Expected created_at value to be 1234567890, got %d", createdAtValue.Value)
	}
}
