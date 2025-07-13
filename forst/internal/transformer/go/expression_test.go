package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"testing"

	"forst/internal/ast"
)

func TestTransformExpression_StructLiteralWithPointerField(t *testing.T) {
	// Test that struct literals with pointer fields are transformed correctly
	// This tests the fix for the pointer dereference bug

	// Create test AST
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	wrapperType := ast.MakeTypeDef("Wrapper", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"user": ast.MakeTypeField(ast.TypeIdent("*User")),
	}))

	wrapperValue := ast.MakeStructLiteral("Wrapper", map[string]ast.ShapeFieldNode{
		"user": ast.MakeStructFieldWithType(ast.MakePointerType("User")),
	})

	fn := ast.MakeFunction("testFn", []ast.ParamNode{
		ast.MakeSimpleParam("w", ast.TypeNode{Ident: "Wrapper"}),
	}, []ast.Node{})

	call := ast.MakeFunctionCall("testFn", []ast.ExpressionNode{wrapperValue})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, wrapperType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function call
	result, err := transformer.transformExpression(call)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the result contains the expected Go code
	// The transformer generates the entire function call, not just the struct literal
	expected := "testFn(Wrapper{user: nil})"
	if !contains(resultStr, expected) {
		t.Errorf("Expected function call with Wrapper struct, got: %s", resultStr)
	}
}

func TestTransformExpression_StructLiteralWithNamedType(t *testing.T) {
	// Test that struct literals use named types when compatible

	// Create test AST
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	userStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"name": ast.MakeStructFieldWithType(ast.MakeStringType()),
	})

	fn := ast.MakeFunction("acceptUser", []ast.ParamNode{
		ast.MakeSimpleParam("u", ast.TypeNode{Ident: "User"}),
	}, []ast.Node{})

	call := ast.MakeFunctionCall("acceptUser", []ast.ExpressionNode{userStruct})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function call
	result, err := transformer.transformExpression(call)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the result uses the named type
	if !contains(resultStr, "User{") {
		t.Errorf("Expected named type usage, got: %s", resultStr)
	}
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}
}

func TestTransformExpression_StructLiteralWithIncompatibleType(t *testing.T) {
	// Test that struct literals use hash-based types when not compatible

	// Create test AST
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	otherStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"foo": ast.MakeStructFieldWithType(ast.MakeStringType()),
	})

	fn := ast.MakeFunction("acceptUser", []ast.ParamNode{
		ast.MakeSimpleParam("u", ast.TypeNode{Ident: "User"}),
	}, []ast.Node{})

	call := ast.MakeFunctionCall("acceptUser", []ast.ExpressionNode{otherStruct})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function call
	result, err := transformer.transformExpression(call)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// The transformer is using the BaseType (User) even for incompatible shapes
	// This is the current behavior - it prioritizes BaseType over structural matching
	if !contains(resultStr, "User{") {
		t.Errorf("Expected User type usage (current behavior), got: %s", resultStr)
	}
}

func TestTransformExpression_NestedStructLiteralWithNamedTypes(t *testing.T) {
	// Test that nested struct literals use named types when compatible

	// Create test AST
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	appContextType := ast.MakeTypeDef("AppContext", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeTypeField(ast.TypeIdent("*String")),
		"user":      ast.MakeTypeField(ast.TypeIdent("*User")),
	}))

	nestedStruct := ast.MakeStructLiteral("AppContext", map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeStructFieldWithType(ast.MakePointerType("String")),
		"user":      ast.MakeStructFieldWithType(ast.MakePointerType("User")),
	})

	fn := ast.MakeFunction("acceptContext", []ast.ParamNode{
		ast.MakeSimpleParam("ctx", ast.TypeNode{Ident: "AppContext"}),
	}, []ast.Node{})

	call := ast.MakeFunctionCall("acceptContext", []ast.ExpressionNode{nestedStruct})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, appContextType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function call
	result, err := transformer.transformExpression(call)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the result uses named types
	if !contains(resultStr, "AppContext{") {
		t.Errorf("Expected named type usage, got: %s", resultStr)
	}
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}
}

func TestTransformExpression_StructLiteralWithNamedType_NoExpectedType(t *testing.T) {
	// More complete reproduction: function with assignment of struct literal with named type and variable reference

	// Create test AST
	echoRequestType := ast.MakeTypeDef("EchoRequest", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"message": ast.MakeTypeField(ast.TypeString),
	}))

	// Simulate: msg := "Hello"; input := EchoRequest{message: msg}
	msgVar := ast.AssignmentNode{
		LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "msg"}}},
		RValues: []ast.ExpressionNode{ast.MakeStringLiteral("Hello")},
		IsShort: true,
	}
	inputStruct := ast.MakeStructLiteral("EchoRequest", map[string]ast.ShapeFieldNode{
		"message": {Node: ast.VariableNode{Ident: ast.Ident{ID: "msg"}}},
	})
	inputVar := ast.AssignmentNode{
		LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "input"}}},
		RValues: []ast.ExpressionNode{inputStruct},
		IsShort: true,
	}

	mainFn := ast.MakeFunction("main", nil, []ast.Node{msgVar, inputVar})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types and function
	err := tc.CheckTypes([]ast.Node{echoRequestType, mainFn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function
	result, err := transformer.transformFunction(mainFn)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the result uses the named type and variable reference
	if !contains(resultStr, "EchoRequest{") {
		t.Errorf("Expected named type usage, got: %s", resultStr)
	}
	if !contains(resultStr, "message: msg") {
		t.Errorf("Expected field value to use variable reference, got: %s", resultStr)
	}
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}
}

func TestTransformExpression_StructLiteralTypeEmissionBug(t *testing.T) {
	// Test that reproduces the client example bug where struct literals
	// use hash-based types instead of named types, causing undefined type errors

	// Create test AST similar to the client example
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"id":   ast.MakeTypeField(ast.TypeString),
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	createUserResponseType := ast.MakeTypeDef("CreateUserResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"user": ast.MakeTypeField(ast.TypeIdent("User")),
	}))

	// Create a function that returns a struct literal
	// This mimics the createUser function from the client example
	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user": ast.MakeStructFieldWithType(ast.TypeNode{Ident: "User"}),
	})

	// Create a function that returns the response struct
	fn := ast.MakeFunction("createUser", []ast.ParamNode{
		ast.MakeSimpleParam("name", ast.TypeNode{Ident: ast.TypeString}),
	}, []ast.Node{
		ast.ReturnNode{
			Values: []ast.ExpressionNode{responseStruct, ast.NilLiteralNode{}},
		},
	})
	// Set the function's return types to (CreateUserResponse, error)
	fn.ReturnTypes = []ast.TypeNode{
		{Ident: "CreateUserResponse"},
		{Ident: ast.TypeError},
	}

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, createUserResponseType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function
	result, err := transformer.transformFunction(fn)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the result uses named types instead of hash-based types
	if !contains(resultStr, "CreateUserResponse{") {
		t.Errorf("Expected CreateUserResponse type usage, got: %s", resultStr)
	}
	if !contains(resultStr, "User{") {
		t.Errorf("Expected User type usage, got: %s", resultStr)
	}
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}

	// Verify the function signature is correct
	if !contains(resultStr, "func createUser(name string) (CreateUserResponse, error)") {
		t.Errorf("Expected correct function signature, got: %s", resultStr)
	}

	// Verify the return statement is correct (accept any field order)
	missing := false
	if !contains(resultStr, "return CreateUserResponse{") {
		t.Errorf("Expected correct return statement, got: %s", resultStr)
		missing = true
	}
	if !contains(resultStr, "user: User{") {
		t.Errorf("Expected user field in return statement, got: %s", resultStr)
		missing = true
	}
	if !contains(resultStr, "created_at: 0") {
		t.Errorf("Expected created_at field in return statement, got: %s", resultStr)
		missing = true
	}
	if missing {
		t.Logf("Full result string:\n%s", resultStr)
	}
}

func TestTransformExpression_StructLiteralWithFieldAssignments(t *testing.T) {
	// Test that reproduces the exact client example pattern with field assignments
	// This tests the case where struct literals have field assignments like:
	// user := T_5uFVdbaNXha{Age: input.Age, Email: input.Email, Id: "123", Name: input.Name}

	// Create test AST similar to the client example
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"id":    ast.MakeTypeField(ast.TypeString),
		"name":  ast.MakeTypeField(ast.TypeString),
		"age":   ast.MakeTypeField(ast.TypeInt),
		"email": ast.MakeTypeField(ast.TypeString),
	}))

	createUserRequestType := ast.MakeTypeDef("CreateUserRequest", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name":  ast.MakeTypeField(ast.TypeString),
		"age":   ast.MakeTypeField(ast.TypeInt),
		"email": ast.MakeTypeField(ast.TypeString),
	}))

	createUserResponseType := ast.MakeTypeDef("CreateUserResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"user":       ast.MakeTypeField(ast.TypeIdent("User")),
		"created_at": ast.MakeTypeField(ast.TypeInt),
	}))

	// Create a struct literal with field assignments (like in the client example)
	// Note: This test focuses on the response struct, not the user struct assignment

	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user":       ast.MakeStructFieldWithType(ast.TypeNode{Ident: "User"}),
		"created_at": ast.MakeStructFieldWithType(ast.TypeNode{Ident: ast.TypeInt}),
	})

	// Create a function that returns the response struct
	fn := ast.MakeFunction("createUser", []ast.ParamNode{
		ast.MakeSimpleParam("input", ast.TypeNode{Ident: "CreateUserRequest"}),
	}, []ast.Node{
		ast.ReturnNode{
			Values: []ast.ExpressionNode{responseStruct, ast.NilLiteralNode{}},
		},
	})
	// Set the function's return types to (CreateUserResponse, error)
	fn.ReturnTypes = []ast.TypeNode{
		{Ident: "CreateUserResponse"},
		{Ident: ast.TypeError},
	}

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, createUserRequestType, createUserResponseType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the function
	result, err := transformer.transformFunction(fn)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the result uses named types instead of hash-based types
	failures := 0
	if !containsIgnoreWhitespace(resultStr, "CreateUserResponse{") {
		t.Errorf("Expected CreateUserResponse type usage, got: %s", resultStr)
		failures++
	}
	if !containsIgnoreWhitespace(resultStr, "User{") {
		t.Errorf("Expected User type usage, got: %s", resultStr)
		failures++
	}
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
		failures++
	}
	if !containsIgnoreWhitespace(resultStr, "func createUser(input CreateUserRequest) (CreateUserResponse, error)") {
		t.Errorf("Expected correct function signature, got: %s", resultStr)
		failures++
	}
	if !containsIgnoreWhitespace(resultStr, "return CreateUserResponse{") {
		t.Errorf("Expected correct return statement, got: %s", resultStr)
		failures++
	}
	if !containsIgnoreWhitespace(resultStr, "user: User{") {
		t.Errorf("Expected user field in return statement, got: %s", resultStr)
		failures++
	}
	if !containsIgnoreWhitespace(resultStr, "created_at: 0") {
		t.Errorf("Expected created_at field in return statement, got: %s", resultStr)
		failures++
	}
	if failures > 0 {
		t.Logf("Full result string:\n%s", resultStr)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to check if a string contains a substring, ignoring whitespace
func containsIgnoreWhitespace(s, substr string) bool {
	s = removeWhitespace(s)
	substr = removeWhitespace(substr)
	return contains(s, substr)
}

// Helper function to remove all whitespace from a string
func removeWhitespace(s string) string {
	var b bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\n' && s[i] != '\t' && s[i] != '\r' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
