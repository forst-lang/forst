package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestPointerStructLiteralShapeGuardBug(t *testing.T) {
	log := setupTestLogger()
	tc := setupTypeChecker(log)

	// Define the User type as it would appear in Forst source
	userType := makeTypeDef("User", makeShape(map[string]ast.ShapeFieldNode{
		"name": makeShapeField(makeStringType()),
	}))

	// Define a wrapper type with a pointer to User
	wrapperType := makeTypeDef("Wrapper", makeShape(map[string]ast.ShapeFieldNode{
		"user": makeShapeField(makePointerType("User")),
	}))

	// Create the value to test: Wrapper{user: &User{name: "Alice"}}
	userStruct := makeStructLiteral("User", map[string]ast.ShapeFieldNode{
		"name": makeStructFieldWithType(makeStringType()),
	})

	wrapperValue := makeStructLiteral("Wrapper", map[string]ast.ShapeFieldNode{
		"user": makeStructFieldWithType(makePointerType("User")),
	})

	// Create a function that takes a Wrapper parameter
	fn := makeFunction("testFn", []ast.ParamNode{
		makeSimpleParam("w", ast.TypeNode{Ident: "Wrapper"}),
	}, []ast.Node{})

	// Create a function call with the wrapper value
	call := makeFunctionCall("testFn", []ast.ExpressionNode{wrapperValue})

	// Pass individual nodes directly to typechecker since PackageNode doesn't have child nodes
	if err := tc.CheckTypes([]ast.Node{userType, wrapperType, fn, call}); err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	transformer := setupTransformer(tc, log)

	expectedType := &ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: "User"}},
	}

	// Use the user struct node for transformation (not the wrapper)
	result, err := transformer.transformShapeNodeWithExpectedType(&userStruct, expectedType)
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

	if strings.Contains(resultStr, "&*User") {
		t.Errorf("BUG REPRODUCED: Generated invalid Go syntax &*User{name: \"Alice\"}")
	}
	if !strings.Contains(resultStr, "&User{") {
		t.Errorf("Missing correct pointer syntax: expected &User{...}")
	}
	// Since we're using explicit types, field values will be nil, which is acceptable
	if !strings.Contains(resultStr, "name:") {
		t.Errorf("Missing expected field name: name:")
	}
}

func TestShapeGuard_UsesNamedType_WhenCompatible(t *testing.T) {
	log := setupTestLogger()
	tc := setupTypeChecker(log)

	// Define a named type User { name: String }
	userType := makeTypeDef("User", makeShape(map[string]ast.ShapeFieldNode{
		"name": makeShapeField(makeStringType()),
	}))

	// Create a struct literal matching User exactly
	userStruct := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": makeStructFieldWithType(makeStringType()),
		},
	}

	// Function expecting User
	fn := makeFunction("acceptUser", []ast.ParamNode{
		makeSimpleParam("u", ast.TypeNode{Ident: "User"}),
	}, []ast.Node{})
	call := makeFunctionCall("acceptUser", []ast.ExpressionNode{userStruct})

	if err := tc.CheckTypes([]ast.Node{userType, fn, call}); err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	transformer := setupTransformer(tc, log)
	result, err := transformer.transformShapeNodeWithExpectedType(&userStruct, &ast.TypeNode{Ident: "User"})
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

	if !strings.Contains(resultStr, "User{") {
		t.Errorf("Expected named type 'User' for matching shape literal, got: %s", resultStr)
	}
	if strings.Contains(resultStr, "T_") {
		t.Errorf("Should not use hash-based type when shape matches named type, got: %s", resultStr)
	}
}

func TestShapeGuard_UsesHashType_WhenNotCompatible(t *testing.T) {
	log := setupTestLogger()
	tc := setupTypeChecker(log)

	// Define a named type User { name: String }
	userType := makeTypeDef("User", makeShape(map[string]ast.ShapeFieldNode{
		"name": makeShapeField(makeStringType()),
	}))

	// Create a struct literal with a different field (does not match User)
	otherStruct := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"foo": makeStructFieldWithType(makeStringType()),
		},
	}

	// Function expecting User
	fn := makeFunction("acceptUser", []ast.ParamNode{
		makeSimpleParam("u", ast.TypeNode{Ident: "User"}),
	}, []ast.Node{})
	call := makeFunctionCall("acceptUser", []ast.ExpressionNode{otherStruct})

	if err := tc.CheckTypes([]ast.Node{userType, fn, call}); err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	transformer := setupTransformer(tc, log)
	result, err := transformer.transformShapeNodeWithExpectedType(&otherStruct, &ast.TypeNode{Ident: "User"})
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

	if !strings.Contains(resultStr[0:2], "T_") {
		t.Errorf("Expected hash-based type for non-matching shape literal, got: %s", resultStr)
	}
	if strings.Contains(resultStr, "User{") {
		t.Errorf("Should not use named type when shape does not match, got: %s", resultStr)
	}
}

// --- Minimal test reproducing shape guard nested type issue ---
func TestShapeGuard_NestedShapes_UseHashTypesInsteadOfNamedTypes(t *testing.T) {
	log := setupTestLogger()
	tc := setupTypeChecker(log)

	// Define the types as in the shape guard example
	userType := makeTypeDef("User", makeShape(map[string]ast.ShapeFieldNode{
		"name": makeShapeField(makeStringType()),
	}))
	appContextType := makeTypeDef("AppContext", makeShape(map[string]ast.ShapeFieldNode{
		"sessionId": makeShapeField(makePointerType("String")),
		"user":      makeShapeField(makePointerType("User")),
	}))

	// Create a struct literal that should use AppContext directly
	// This mimics the shape guard example: {sessionId: &sessionId, user: {name: "Alice"}}
	nestedStruct := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {
				Assertion: &ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{
						Name: "Value",
						Args: []ast.ConstraintArgumentNode{{
							Value: func() *ast.ValueNode {
								v := ast.ValueNode(ast.VariableNode{Ident: ast.Ident{ID: "sessionId"}})
								return &v
							}(),
						}},
					}},
				},
			},
			"user": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {
							Assertion: &ast.AssertionNode{
								Constraints: []ast.ConstraintNode{{
									Name: "Value",
									Args: []ast.ConstraintArgumentNode{{
										Value: func() *ast.ValueNode {
											v := ast.ValueNode(ast.StringLiteralNode{Value: "Alice"})
											return &v
										}(),
									}},
								}},
							},
						},
					},
				},
			},
		},
	}

	// Function expecting AppContext
	fn := makeFunction("acceptContext", []ast.ParamNode{
		makeSimpleParam("ctx", ast.TypeNode{Ident: "AppContext"}),
	}, []ast.Node{})
	call := makeFunctionCall("acceptContext", []ast.ExpressionNode{nestedStruct})

	if err := tc.CheckTypes([]ast.Node{userType, appContextType, fn, call}); err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	transformer := setupTransformer(tc, log)
	result, err := transformer.transformShapeNodeWithExpectedType(&nestedStruct, &ast.TypeNode{Ident: "AppContext"})
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

	// The issue: nested shapes should use named types (AppContext, User) but are using hash-based types
	// This reproduces the exact compilation error from the shape guard example
	if strings.Contains(resultStr[0:2], "T_") {
		t.Errorf("BUG REPRODUCED: Nested shapes using hash-based types instead of named types, got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "AppContext{") {
		t.Errorf("Expected AppContext for nested shape, got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "User{") {
		t.Errorf("Expected User for nested user shape, got: %s", resultStr)
	}
}
