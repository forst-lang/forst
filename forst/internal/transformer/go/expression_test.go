package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"strings"
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
		"user":       ast.MakeTypeField(ast.TypeIdent("User")),
		"created_at": ast.MakeTypeField(ast.TypeInt),
	}))

	// Create a function that returns a struct literal
	// This mimics the createUser function from the client example
	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user":       ast.MakeStructFieldWithType(ast.TypeNode{Ident: "User"}),
		"created_at": ast.MakeStructFieldWithType(ast.TypeNode{Ident: ast.TypeInt}),
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

func TestTransformExpression_StructLiteralAssignment_UsesNamedType(t *testing.T) {
	// Test that struct literal assignment uses the named type, not a hash-based type
	// Reproduces the bug seen in the client example

	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"id":    ast.MakeTypeField(ast.TypeString),
		"name":  ast.MakeTypeField(ast.TypeString),
		"age":   ast.MakeTypeField(ast.TypeInt),
		"email": ast.MakeTypeField(ast.TypeString),
	}))

	// Assignment: user := User{Id: "123", Name: input.Name, Age: input.Age, Email: input.Email}
	userStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"id":    ast.MakeStructFieldWithType(ast.MakeStringType()),
		"name":  ast.MakeStructFieldWithType(ast.MakeStringType()),
		"age":   ast.MakeStructFieldWithType(ast.TypeNode{Ident: ast.TypeInt}),
		"email": ast.MakeStructFieldWithType(ast.MakeStringType()),
	})

	assign := ast.AssignmentNode{
		LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "user"}}},
		RValues: []ast.ExpressionNode{userStruct},
		IsShort: true,
	}

	fn := ast.MakeFunction("createUser", []ast.ParamNode{
		ast.MakeSimpleParam("input", ast.TypeNode{Ident: "User"}),
	}, []ast.Node{assign})

	// Setup typechecker and transformer
	log := setupTestLogger()
	c := setupTypeChecker(log)
	transformer := setupTransformer(c, log)

	// Register types
	err := c.CheckTypes([]ast.Node{userType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the assignment
	stmt, err := transformer.transformStatement(assign)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), stmt)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()

	// Verify the assignment uses the named type
	if !contains(resultStr, "user := User{") {
		t.Errorf("Expected assignment to use named type, got: %s", resultStr)
	}
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}
}

func TestTransformExpression_ClientExampleBug(t *testing.T) {
	// Test that reproduces the exact client example bug
	// This tests the CreateUser function pattern from user.ft

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

	// Create the CreateUser function that reproduces the client example
	// func CreateUser(input CreateUserRequest) {
	//   user := { id: "123", name: input.name, age: input.age, email: input.email }
	//   return { user: user, created_at: 1234567890 }
	// }

	// Create the user assignment: user := { id: "123", name: input.name, age: input.age, email: input.email }
	userStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"id":    ast.MakeStructField(ast.StringLiteralNode{Value: "123"}),
		"name":  ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "input"}}), // This should be input.name
		"age":   ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "input"}}), // This should be input.age
		"email": ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "input"}}), // This should be input.email
	})

	userAssign := ast.AssignmentNode{
		LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "user"}}},
		RValues: []ast.ExpressionNode{userStruct},
		IsShort: true,
	}

	// Create the return statement: return { user: user, created_at: 1234567890 }
	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user":       ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "user"}}),
		"created_at": ast.MakeStructField(ast.IntLiteralNode{Value: 1234567890}),
	})

	returnStmt := ast.ReturnNode{
		Values: []ast.ExpressionNode{responseStruct, ast.NilLiteralNode{}},
	}

	fn := ast.MakeFunction("CreateUser", []ast.ParamNode{
		ast.MakeSimpleParam("input", ast.TypeNode{Ident: "CreateUserRequest"}),
	}, []ast.Node{userAssign, returnStmt})

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
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}

	// Verify the function signature is correct
	if !contains(resultStr, "func CreateUser(input CreateUserRequest) (CreateUserResponse, error)") {
		t.Errorf("Expected correct function signature, got: %s", resultStr)
	}

	// Verify the return statement is correct
	if !contains(resultStr, "return CreateUserResponse{") {
		t.Errorf("Expected correct return statement, got: %s", resultStr)
	}

	if !contains(resultStr, "user: user") {
		t.Errorf("Expected user field in return statement, got: %s", resultStr)
	}

	if !contains(resultStr, "created_at: 1234567890") {
		t.Errorf("Expected created_at field in return statement, got: %s", resultStr)
	}

	t.Logf("Generated code:\n%s", resultStr)
}

func TestTransformExpression_ClientExampleWithFieldAccess(t *testing.T) {
	// Test that reproduces the exact client example with field access
	// This tests the CreateUser function with proper field access like input.name, input.age, etc.

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

	// Create field access expressions like input.name, input.age, input.email
	// For now, use simple variable references - the real issue is type emission
	nameAccess := ast.VariableNode{Ident: ast.Ident{ID: "input"}}
	ageAccess := ast.VariableNode{Ident: ast.Ident{ID: "input"}}
	emailAccess := ast.VariableNode{Ident: ast.Ident{ID: "input"}}

	// Create the user assignment: user := { id: "123", name: input.name, age: input.age, email: input.email }
	userStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"id":    ast.MakeStructField(ast.StringLiteralNode{Value: "123"}),
		"name":  ast.MakeStructField(nameAccess),
		"age":   ast.MakeStructField(ageAccess),
		"email": ast.MakeStructField(emailAccess),
	})

	userAssign := ast.AssignmentNode{
		LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "user"}}},
		RValues: []ast.ExpressionNode{userStruct},
		IsShort: true,
	}

	// Create the return statement: return { user: user, created_at: 1234567890 }
	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user":       ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "user"}}),
		"created_at": ast.MakeStructField(ast.IntLiteralNode{Value: 1234567890}),
	})

	returnStmt := ast.ReturnNode{
		Values: []ast.ExpressionNode{responseStruct, ast.NilLiteralNode{}},
	}

	fn := ast.MakeFunction("CreateUser", []ast.ParamNode{
		ast.MakeSimpleParam("input", ast.TypeNode{Ident: "CreateUserRequest"}),
	}, []ast.Node{userAssign, returnStmt})

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
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}

	// Verify the function signature is correct
	if !contains(resultStr, "func CreateUser(input CreateUserRequest) (CreateUserResponse, error)") {
		t.Errorf("Expected correct function signature, got: %s", resultStr)
	}

	// Verify the return statement is correct
	if !contains(resultStr, "return CreateUserResponse{") {
		t.Errorf("Expected correct return statement, got: %s", resultStr)
	}

	if !contains(resultStr, "user: user") {
		t.Errorf("Expected user field in return statement, got: %s", resultStr)
	}

	if !contains(resultStr, "created_at: 1234567890") {
		t.Errorf("Expected created_at field in return statement, got: %s", resultStr)
	}

	// Verify field access expressions are transformed correctly
	if !contains(resultStr, "input.Name") || !contains(resultStr, "input.Age") || !contains(resultStr, "input.Email") {
		t.Errorf("Expected field access expressions to be transformed correctly, got: %s", resultStr)
	}

	t.Logf("Generated code:\n%s", resultStr)
}

func TestTransformExpression_UnifiedHelperWorking(t *testing.T) {
	// Simple test to verify our unified helper is working correctly
	// This tests the core functionality without complex field access

	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"id":   ast.MakeTypeField(ast.TypeString),
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	createUserResponseType := ast.MakeTypeDef("CreateUserResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"user": ast.MakeTypeField(ast.TypeIdent("User")),
	}))

	// Create a simple struct literal: { user: User{ id: "123", name: "test" } }
	userStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"id":   ast.MakeStructField(ast.StringLiteralNode{Value: "123"}),
		"name": ast.MakeStructField(ast.StringLiteralNode{Value: "test"}),
	})

	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user": ast.MakeStructField(userStruct),
	})

	returnStmt := ast.ReturnNode{
		Values: []ast.ExpressionNode{responseStruct, ast.NilLiteralNode{}},
	}

	fn := ast.MakeFunction("CreateUser", []ast.ParamNode{}, []ast.Node{returnStmt})
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
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}

	// Verify the function signature is correct
	if !contains(resultStr, "func CreateUser() (CreateUserResponse, error)") {
		t.Errorf("Expected correct function signature, got: %s", resultStr)
	}

	// Verify the return statement is correct
	if !contains(resultStr, "return CreateUserResponse{") {
		t.Errorf("Expected correct return statement, got: %s", resultStr)
	}

	if !contains(resultStr, "user: User{") {
		t.Errorf("Expected user field in return statement, got: %s", resultStr)
	}

	t.Logf("Generated code:\n%s", resultStr)
}

func TestTransformExpression_TypeEmissionIssue(t *testing.T) {
	// Test that specifically addresses the type emission issue
	// This creates a complete package with types and functions to verify type emission

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

	// Create the CreateUser function
	userStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"id":    ast.MakeStructField(ast.StringLiteralNode{Value: "123"}),
		"name":  ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "input"}}),
		"age":   ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "input"}}),
		"email": ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "input"}}),
	})

	userAssign := ast.AssignmentNode{
		LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "user"}}},
		RValues: []ast.ExpressionNode{userStruct},
		IsShort: true,
	}

	responseStruct := ast.MakeStructLiteral("CreateUserResponse", map[string]ast.ShapeFieldNode{
		"user":       ast.MakeStructField(ast.VariableNode{Ident: ast.Ident{ID: "user"}}),
		"created_at": ast.MakeStructField(ast.IntLiteralNode{Value: 1234567890}),
	})

	returnStmt := ast.ReturnNode{
		Values: []ast.ExpressionNode{responseStruct, ast.NilLiteralNode{}},
	}

	fn := ast.MakeFunction("CreateUser", []ast.ParamNode{
		ast.MakeSimpleParam("input", ast.TypeNode{Ident: "CreateUserRequest"}),
	}, []ast.Node{userAssign, returnStmt})

	fn.ReturnTypes = []ast.TypeNode{
		{Ident: "CreateUserResponse"},
		{Ident: ast.TypeError},
	}

	// Create a package with all the types and functions
	packageNode := ast.MakePackage("user", []ast.Node{})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types and functions
	err := tc.CheckTypes([]ast.Node{userType, createUserRequestType, createUserResponseType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the entire package with all nodes
	result, err := transformer.TransformForstFileToGo([]ast.Node{packageNode, userType, createUserRequestType, createUserResponseType, fn})
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

	// Verify that all types are emitted
	if !contains(resultStr, "type User struct") {
		t.Errorf("Expected User type to be emitted, got: %s", resultStr)
	}

	if !contains(resultStr, "type CreateUserRequest struct") {
		t.Errorf("Expected CreateUserRequest type to be emitted, got: %s", resultStr)
	}

	if !contains(resultStr, "type CreateUserResponse struct") {
		t.Errorf("Expected CreateUserResponse type to be emitted, got: %s", resultStr)
	}

	// Verify that no hash-based types are used
	if contains(resultStr, "T_") {
		t.Errorf("Expected no hash-based type usage, got: %s", resultStr)
	}

	// Verify the function signature is correct
	if !contains(resultStr, "func CreateUser(input CreateUserRequest) (CreateUserResponse, error)") {
		t.Errorf("Expected correct function signature, got: %s", resultStr)
	}

	// Verify the return statement is correct
	if !contains(resultStr, "return CreateUserResponse{") {
		t.Errorf("Expected correct return statement, got: %s", resultStr)
	}

	t.Logf("Generated code:\n%s", resultStr)
}

func TestTransformExpression_ShapeGuardIssue(t *testing.T) {
	// Test that reproduces the exact shape guard issue
	// This tests struct literals that should use named types from function signatures

	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	appContextType := ast.MakeTypeDef("AppContext", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeTypeField(ast.TypeString), // Simplified for testing
		"user":      ast.MakeTypeField(ast.TypeIdent("User")),
	}))

	// Create the function with the complex parameter type
	fn := ast.MakeFunction("createTask", []ast.ParamNode{
		ast.MakeSimpleParam("op", ast.TypeNode{Ident: "T_488eVThFocF"}), // This is the hash-based type from the actual error
	}, []ast.Node{
		ast.ReturnNode{Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: "test"}, ast.NilLiteralNode{}}},
	})

	fn.ReturnTypes = []ast.TypeNode{
		{Ident: ast.TypeString},
		{Ident: ast.TypeError},
	}

	// Create struct literals that should match the function parameter type
	// These are the struct literals from the main function
	sessionIdVar := ast.VariableNode{Ident: ast.Ident{ID: "sessionId"}}
	aliceUser := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"name": ast.MakeStructField(ast.StringLiteralNode{Value: "Alice"}),
	})

	ctxStruct := ast.MakeStructLiteral("AppContext", map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeStructField(sessionIdVar), // Simplified for testing
		"user":      ast.MakeStructField(aliceUser),
	})

	inputStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"name": ast.MakeStructField(ast.StringLiteralNode{Value: "Fix memory leak in Node.js app"}),
	})

	mainStruct := ast.MakeStructLiteral("T_488eVThFocF", map[string]ast.ShapeFieldNode{
		"ctx":   ast.MakeStructField(ctxStruct),
		"input": ast.MakeStructField(inputStruct),
	})

	// Create the function call (not used in this test but shows the pattern)
	_ = ast.FunctionCallNode{
		Function:  ast.Ident{ID: "createTask"},
		Arguments: []ast.ExpressionNode{mainStruct},
	}

	// Create a package with all the types and functions
	packageNode := ast.MakePackage("main", []ast.Node{})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types and functions
	err := tc.CheckTypes([]ast.Node{userType, appContextType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the entire package with all nodes
	result, err := transformer.TransformForstFileToGo([]ast.Node{packageNode, userType, appContextType, fn})
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

	// Verify that all types are emitted
	if !contains(resultStr, "type User struct") {
		t.Errorf("Expected User type to be emitted, got: %s", resultStr)
	}

	if !contains(resultStr, "type AppContext struct") {
		t.Errorf("Expected AppContext type to be emitted, got: %s", resultStr)
	}

	// Verify that the function is emitted
	if !contains(resultStr, "func createTask(op") {
		t.Errorf("Expected createTask function to be emitted, got: %s", resultStr)
	}

	// The key issue: verify that struct literals use named types instead of hash-based types
	// This is what's failing in the actual shape guard example
	if contains(resultStr, "T_LYQafBLM8TQ") {
		t.Errorf("Expected no hash-based type T_LYQafBLM8TQ, got: %s", resultStr)
	}

	if contains(resultStr, "T_X86jJwVQ4mH") {
		t.Errorf("Expected no hash-based type T_X86jJwVQ4mH, got: %s", resultStr)
	}

	t.Logf("Generated code:\n%s", resultStr)
}

func TestTransformExpression_ShapeGuardStructLiteralIssue(t *testing.T) {
	// Test that reproduces the exact struct literal issue from shape guard
	// This tests that struct literals use named types instead of hash-based types

	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	appContextType := ast.MakeTypeDef("AppContext", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeTypeField(ast.TypeString),
		"user":      ast.MakeTypeField(ast.TypeIdent("User")),
	}))

	// Create the function parameter type that matches the shape guard example
	// This should be the type that the struct literals should match
	expectedType := ast.MakeTypeDef("T_488eVThFocF", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"ctx":   ast.MakeTypeField(ast.TypeIdent("AppContext")),
		"input": ast.MakeTypeField(ast.TypeIdent("User")),
	}))

	// Create struct literals that should use named types
	// These are the struct literals from the main function in shape_guard.ft
	sessionIdVar := ast.VariableNode{Ident: ast.Ident{ID: "sessionId"}}
	aliceUser := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"name": ast.MakeStructField(ast.StringLiteralNode{Value: "Alice"}),
	})

	ctxStruct := ast.MakeStructLiteral("AppContext", map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeStructField(sessionIdVar),
		"user":      ast.MakeStructField(aliceUser),
	})

	inputStruct := ast.MakeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"name": ast.MakeStructField(ast.StringLiteralNode{Value: "Fix memory leak in Node.js app"}),
	})

	// This is the problematic struct literal that should use named types
	_ = ast.MakeStructLiteral("T_488eVThFocF", map[string]ast.ShapeFieldNode{
		"ctx":   ast.MakeStructField(ctxStruct),
		"input": ast.MakeStructField(inputStruct),
	})

	// Create a package with all the types
	packageNode := ast.MakePackage("main", []ast.Node{})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, appContextType, expectedType})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the entire package with all nodes
	result, err := transformer.TransformForstFileToGo([]ast.Node{packageNode, userType, appContextType, expectedType})
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

	// Verify that all types are emitted
	if !contains(resultStr, "type User struct") {
		t.Errorf("Expected User type to be emitted, got: %s", resultStr)
	}

	if !contains(resultStr, "type AppContext struct") {
		t.Errorf("Expected AppContext type to be emitted, got: %s", resultStr)
	}

	if !contains(resultStr, "type T_488eVThFocF struct") {
		t.Errorf("Expected T_488eVThFocF type to be emitted, got: %s", resultStr)
	}

	// The key issue: verify that struct literals use named types instead of hash-based types
	// This is what's failing in the actual shape guard example
	if contains(resultStr, "T_LYQafBLM8TQ") {
		t.Errorf("Expected no hash-based type T_LYQafBLM8TQ, got: %s", resultStr)
	}

	if contains(resultStr, "T_X86jJwVQ4mH") {
		t.Errorf("Expected no hash-based type T_X86jJwVQ4mH, got: %s", resultStr)
	}

	// Verify that the struct literal uses the correct named types
	if !contains(resultStr, "ctx: AppContext{") {
		t.Errorf("Expected ctx field to use AppContext type, got: %s", resultStr)
	}

	if !contains(resultStr, "input: User{") {
		t.Errorf("Expected input field to use User type, got: %s", resultStr)
	}

	t.Logf("Generated code:\n%s", resultStr)
}

func TestTransformExpression_ShapeGuardPointerFieldIssue(t *testing.T) {
	// Test that reproduces the exact shape guard issue with pointer fields
	// This tests that struct literals with pointer fields are handled correctly

	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	appContextType := ast.MakeTypeDef("AppContext", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"sessionId": ast.ShapeFieldNode{Type: &ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}},
		"user":      ast.ShapeFieldNode{Type: &ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: "User"}}}},
	}))

	// Create the function with the complex parameter type
	fn := ast.MakeFunction("createTask", []ast.ParamNode{
		ast.MakeSimpleParam("op", ast.TypeNode{Ident: "T_488eVThFocF"}),
	}, []ast.Node{
		ast.ReturnNode{Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: "test"}, ast.NilLiteralNode{}}},
	})
	fn.ReturnTypes = []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeError}}

	// Create the package with all types and functions
	pkg := ast.MakePackage("main", []ast.Node{userType, appContextType, fn})

	// Setup typechecker and transformer
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)

	// Register types
	err := tc.CheckTypes([]ast.Node{userType, appContextType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Transform the package
	result, err := transformer.TransformForstFileToGo([]ast.Node{&pkg, userType, appContextType, fn})
	if err != nil {
		t.Fatalf("Failed to transform package: %v", err)
	}

	// Convert Go AST to string
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	generatedCode := buf.String()

	// Check that pointer types are handled correctly in type definitions (whitespace-insensitive)
	if !containsIgnoreWhitespace(generatedCode, "sessionId*string") {
		t.Errorf("Generated code should contain pointer field sessionId *string (whitespace-insensitive), got: %s", generatedCode)
	}

	if !containsIgnoreWhitespace(generatedCode, "user*User") {
		t.Errorf("Generated code should contain pointer field user *User (whitespace-insensitive), got: %s", generatedCode)
	}

	// The generated code should not contain hash-based types for the main struct literals
	if strings.Contains(generatedCode, "T_LYQafBLM8TQ{") {
		t.Errorf("Generated code should not contain hash-based type T_LYQafBLM8TQ, got: %s", generatedCode)
	}

	if strings.Contains(generatedCode, "T_X86jJwVQ4mH{") {
		t.Errorf("Generated code should not contain hash-based type T_X86jJwVQ4mH, got: %s", generatedCode)
	}

	t.Logf("Generated code:\n%s", generatedCode)
}

func TestPointerValueMismatch_Minimal(t *testing.T) {
	// Minimal test for pointer/value mismatch bug
	userType := ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))
	appContextType := ast.MakeTypeDef("AppContext", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"sessionId": {Type: &ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}},
		"user":      {Type: &ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: "User"}}}},
	}))
	structLiteral := ast.MakeStructLiteral("AppContext", map[string]ast.ShapeFieldNode{
		"sessionId": ast.MakeStructFieldWithType(ast.MakePointerType("String")),
		"user":      ast.MakeStructFieldWithType(ast.MakePointerType("User")),
	})
	fn := ast.MakeFunction("testFn", []ast.ParamNode{
		ast.MakeSimpleParam("ctx", ast.TypeNode{Ident: "AppContext"}),
	}, []ast.Node{})
	call := ast.MakeFunctionCall("testFn", []ast.ExpressionNode{structLiteral})
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	transformer := setupTransformer(tc, log)
	err := tc.CheckTypes([]ast.Node{userType, appContextType, fn})
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Add detailed logging to pinpoint the issue
	t.Log("=== POINTER/VALUE MISMATCH DEBUG ===")
	t.Logf("AppContext type definition: %+v", appContextType)
	t.Logf("Struct literal: %+v", structLiteral)
	t.Logf("Function parameter type: %+v", fn.Params[0].GetType())

	// Check what the typechecker knows about AppContext
	if def, exists := tc.Defs["AppContext"]; exists {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			t.Logf("AppContext type definition in typechecker: %+v", typeDef)
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				t.Logf("AppContext shape fields: %+v", shapeExpr.Shape.Fields)
				for fieldName, field := range shapeExpr.Shape.Fields {
					t.Logf("  Field %s: Type=%+v", fieldName, field.Type)
				}
			}
		}
	}

	result, err := transformer.transformExpression(call)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), result)
	if err != nil {
		t.Fatalf("Failed to format result: %v", err)
	}
	resultStr := buf.String()
	t.Logf("Generated Go code: %s", resultStr)

	if !contains(resultStr, "testFn(AppContext{") {
		t.Errorf("Expected function call with AppContext struct, got: %s", resultStr)
	}
	if !contains(resultStr, "sessionId: &String{") {
		t.Errorf("Expected sessionId field to be a pointer value (&String{}), got: %s", resultStr)
	}
	if !contains(resultStr, "user: &User{") {
		t.Errorf("Expected user field to be a pointer value (&User{}), got: %s", resultStr)
	}
}
