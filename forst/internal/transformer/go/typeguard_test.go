package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

// typeIdentPtr creates a pointer to a TypeIdent
func typeIdentPtr(s string) *ast.TypeIdent {
	ti := ast.TypeIdent(s)
	return &ti
}

func TestTransformEnsureConstraintWithIsConstraint(t *testing.T) {
	// Test the core issue: the "is" constraint is not implemented
	// This reproduces the issue from the shape_guard.ft example

	// Create a simple ensure node with an "is" constraint
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier("m")},
		},
		Assertion: ast.AssertionNode{
			BaseType: typeIdentPtr("MutationArg"),
			Constraints: []ast.ConstraintNode{
				ast.ConstraintNode{
					Name: "is",
					Args: []ast.ConstraintArgumentNode{
						{
							Shape: &ast.ShapeNode{
								Fields: map[string]ast.ShapeFieldNode{
									"ctx": {
										Type: &ast.TypeNode{Ident: ast.TypeIdent("Shape")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a minimal type checker
	tc := typechecker.New(nil, false)

	// Create a transformer
	transformer := New(tc, nil)

	// Try to transform the ensure constraint
	// This should fail because "is" is not a built-in constraint
	_, err := transformer.transformEnsureConstraint(ensure, ensure.Assertion.Constraints[0], ast.TypeNode{Ident: ast.TypeIdent("MutationArg")})

	// This should fail with "no valid transformation found for constraint: is"
	if err == nil {
		t.Fatal("Expected error 'no valid transformation found for constraint: is', but got nil")
	}

	expectedError := "no valid transformation found for constraint: is"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestTransformEnsureConstraintWithValidConstraint(t *testing.T) {
	// Test that valid constraints work correctly

	// Create a simple ensure node with a valid constraint
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier("ctx")},
		},
		Assertion: ast.AssertionNode{
			BaseType: typeIdentPtr("*String"),
			Constraints: []ast.ConstraintNode{
				ast.ConstraintNode{
					Name: "NotNil",
					Args: []ast.ConstraintArgumentNode{},
				},
			},
		},
	}

	// Create a minimal type checker
	tc := typechecker.New(nil, false)

	// Create a transformer
	transformer := New(tc, nil)

	// Try to transform the ensure constraint
	// This should succeed because NotNil is a valid built-in constraint for Pointer types
	_, err := transformer.transformEnsureConstraint(ensure, ensure.Assertion.Constraints[0], ast.TypeNode{Ident: ast.TypePointer})

	// This should succeed because NotNil is a valid built-in constraint
	if err != nil {
		t.Errorf("Expected no error, but got: %s", err.Error())
	}
}

func TestTransformFunctionCallWithShapeLiteralArgument_UsesExpectedType(t *testing.T) {
	// Setup minimal typechecker and transformer
	tc := typechecker.New(nil, false)
	tr := New(tc, nil)

	// Register the expected type in the typechecker and output
	typeName := "T_ShapeArg"
	shapeType := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"foo": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs[ast.TypeIdent(typeName)] = ast.TypeDefNode{
		Ident: ast.TypeIdent(typeName),
		Expr:  ast.TypeDefShapeExpr{Shape: *shapeType},
	}
	// Force the transformer to emit the type
	_ = tr.defineShapeType(shapeType)

	// Create a function signature expecting this type
	funcName := ast.Identifier("f")
	tc.Functions[funcName] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: funcName},
		Parameters: []typechecker.ParameterSignature{{
			Ident: ast.Ident{ID: ast.Identifier("arg")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent(typeName)},
		}},
		ReturnTypes: nil,
	}

	// Create a function call with a shape literal as the argument
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"foo": {Assertion: &ast.AssertionNode{Constraints: []ast.ConstraintNode{{
				Name: "Value",
				Args: []ast.ConstraintArgumentNode{{
					Value: func() *ast.ValueNode { v := ast.ValueNode(ast.StringLiteralNode{Value: "bar"}); return &v }(),
				}},
			}}}},
		},
	}
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: funcName},
		Arguments: []ast.ExpressionNode{shapeLiteral},
	}

	// Remove the type registration for T_ShapeArg so the shape literal gets a new hash-based type
	delete(tc.Defs, ast.TypeIdent(typeName))

	// Transform the function call
	stmt, err := tr.transformStatement(call)
	if err != nil {
		t.Fatalf("transformStatement failed: %v", err)
	}

	// Check that the generated Go code uses the hash-based type for the shape literal
	callExpr, ok := stmt.(*goast.ExprStmt)
	if !ok {
		t.Fatalf("Expected ExprStmt, got %T", stmt)
	}
	goCall, ok := callExpr.X.(*goast.CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", callExpr.X)
	}
	if len(goCall.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(goCall.Args))
	}
	compLit, ok := goCall.Args[0].(*goast.CompositeLit)
	if !ok {
		t.Fatalf("Expected CompositeLit as argument, got %T", goCall.Args[0])
	}
	ident, ok := compLit.Type.(*goast.Ident)
	if !ok {
		t.Fatalf("Expected Ident as composite literal type, got %T", compLit.Type)
	}
	if ident.Name == typeName {
		t.Errorf("Expected composite literal type to NOT be %q, but got %q (Go AST: %#v)", typeName, ident.Name, compLit.Type)
	}
}

func TestTransformFunctionCallWithShapeLiteralArgument_UndefinedTypeError(t *testing.T) {
	// Setup minimal typechecker and transformer
	tc := typechecker.New(nil, false)
	tr := New(tc, nil)

	// Register a function signature expecting a shape type
	funcName := ast.Identifier("f")
	paramType := ast.TypeNode{Ident: ast.TypeIdent("T_ShapeArg")}
	tc.Functions[funcName] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: funcName},
		Parameters: []typechecker.ParameterSignature{{
			Ident: ast.Ident{ID: ast.Identifier("arg")},
			Type:  paramType,
		}},
		ReturnTypes: nil,
	}

	// Do NOT register the type T_ShapeArg in tc.Defs (simulate missing type)

	// Create a function call with a shape literal as the argument
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"foo": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: funcName},
		Arguments: []ast.ExpressionNode{shapeLiteral},
	}

	// Transform the function call
	stmt, err := tr.transformStatement(call)
	if err != nil {
		t.Fatalf("transformStatement failed: %v", err)
	}

	// Check that the generated Go code uses a type name that is not defined
	callExpr, ok := stmt.(*goast.ExprStmt)
	if !ok {
		t.Fatalf("Expected ExprStmt, got %T", stmt)
	}
	goCall, ok := callExpr.X.(*goast.CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", callExpr.X)
	}
	if len(goCall.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(goCall.Args))
	}
	compLit, ok := goCall.Args[0].(*goast.CompositeLit)
	if !ok {
		t.Fatalf("Expected CompositeLit as argument, got %T", goCall.Args[0])
	}
	ident, ok := compLit.Type.(*goast.Ident)
	if !ok {
		t.Fatalf("Expected Ident as composite literal type, got %T", compLit.Type)
	}
	// The type name should not be T_ShapeArg, and should not be in tc.Defs
	if ident.Name == "T_ShapeArg" {
		t.Errorf("Expected composite literal type to NOT be %q, but got %q (Go AST: %#v)", "T_ShapeArg", ident.Name, compLit.Type)
	}
	if _, exists := tc.Defs[ast.TypeIdent(ident.Name)]; exists {
		t.Errorf("Type %q should not be defined in tc.Defs, but it was", ident.Name)
	}
}

func TestTransformFunctionCallWithShapeLiteralArgument_UsesParameterTypeDef(t *testing.T) {
	// Setup minimal typechecker and transformer
	tc := typechecker.New(nil, false)
	tr := New(tc, nil)

	// Register a function signature expecting an assertion type (like AppMutation.Input)
	funcName := ast.Identifier("f")
	paramTypeName := "T_KBdY4FCchfk" // This is the inferred type for AppMutation.Input({...})
	paramType := ast.TypeNode{Ident: ast.TypeIdent(paramTypeName)}
	tc.Functions[funcName] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: funcName},
		Parameters: []typechecker.ParameterSignature{{
			Ident: ast.Ident{ID: ast.Identifier("arg")},
			Type:  paramType,
		}},
		ReturnTypes: nil,
	}

	// Debug: Check what getGeneratedTypeNameForTypeNode returns for the parameter type
	generatedTypeName, err := tr.getGeneratedTypeNameForTypeNode(paramType)
	if err != nil {
		t.Logf("getGeneratedTypeNameForTypeNode error: %v", err)
	} else {
		t.Logf("getGeneratedTypeNameForTypeNode returned: %q", generatedTypeName)
	}

	// Debug: Check what's in tc.Defs
	t.Logf("tc.Defs contains %d entries:", len(tc.Defs))
	for key, value := range tc.Defs {
		t.Logf("  %q: %T", key, value)
	}

	// Debug: Check if the parameter type is actually in tc.Defs
	if _, exists := tc.Defs[ast.TypeIdent(paramTypeName)]; exists {
		t.Logf("Parameter type %q IS in tc.Defs", paramTypeName)
	} else {
		t.Logf("Parameter type %q is NOT in tc.Defs", paramTypeName)
	}

	// Register the expected type in tc.Defs (simulate real codegen)
	shapeType := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx":   {Type: &ast.TypeNode{Ident: ast.TypeIdent("AppContext")}},
			"input": {Type: &ast.TypeNode{Ident: ast.TypeIdent("T_azh9nsqmxaF")}},
		},
	}
	tc.Defs[ast.TypeIdent(paramTypeName)] = ast.TypeDefNode{
		Ident: ast.TypeIdent(paramTypeName),
		Expr:  ast.TypeDefShapeExpr{Shape: *shapeType},
	}
	_ = tr.defineShapeType(shapeType) // ensure the transformer emits the type

	// Create a function call with a shape literal as the argument
	// This shape literal has a different structure than the parameter type
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"sessionId": {Assertion: &ast.AssertionNode{Constraints: []ast.ConstraintNode{{
							Name: "Value",
							Args: []ast.ConstraintArgumentNode{{
								Value: func() *ast.ValueNode { v := ast.ValueNode(ast.StringLiteralNode{Value: "bar"}); return &v }(),
							}},
						}}}},
					},
				},
			},
			"input": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {Assertion: &ast.AssertionNode{Constraints: []ast.ConstraintNode{{
							Name: "Value",
							Args: []ast.ConstraintArgumentNode{{
								Value: func() *ast.ValueNode { v := ast.ValueNode(ast.StringLiteralNode{Value: "Alice"}); return &v }(),
							}},
						}}}},
					},
				},
			},
		},
	}
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: funcName},
		Arguments: []ast.ExpressionNode{shapeLiteral},
	}

	// Transform the function call
	stmt, err := tr.transformStatement(call)
	if err != nil {
		t.Fatalf("transformStatement failed: %v", err)
	}

	// Check that the generated Go code uses the parameter type (which is defined)
	callExpr, ok := stmt.(*goast.ExprStmt)
	if !ok {
		t.Fatalf("Expected ExprStmt, got %T", stmt)
	}
	goCall, ok := callExpr.X.(*goast.CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", callExpr.X)
	}
	if len(goCall.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(goCall.Args))
	}
	compLit, ok := goCall.Args[0].(*goast.CompositeLit)
	if !ok {
		t.Fatalf("Expected CompositeLit as argument, got %T", goCall.Args[0])
	}
	ident, ok := compLit.Type.(*goast.Ident)
	if !ok {
		t.Fatalf("Expected Ident as composite literal type, got %T", compLit.Type)
	}
	if ident.Name != paramTypeName {
		t.Errorf("Expected composite literal type %q (the parameter type), got %q (Go AST: %#v)", paramTypeName, ident.Name, compLit.Type)
	}
	if _, exists := tc.Defs[ast.TypeIdent(ident.Name)]; !exists {
		t.Errorf("Type %q used in composite literal is not defined in tc.Defs (bug: undefined type in output)", ident.Name)
	}
}

func TestTransformFunctionCallWithShapeLiteralArgument_UsesInferredParameterTypeDef(t *testing.T) {
	// Setup minimal typechecker and transformer
	tc := typechecker.New(nil, false)
	tr := New(tc, nil)

	// Register a minimal type guard for "Input" (as in the real pipeline)
	inputGuard := &ast.TypeGuardNode{
		Ident: "Input",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("m")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("MutationArg")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("input")},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("Shape")},
			},
		},
		Body: nil,
	}
	tc.Defs[ast.TypeIdent("Input")] = inputGuard

	// Simulate a type assertion like AppMutation.Input({input: {name: String}})
	assertion := &ast.AssertionNode{
		BaseType: (*ast.TypeIdent)(typeIdentPtr("AppMutation")),
		Constraints: []ast.ConstraintNode{{
			Name: "Input",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"input": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			}},
		}},
	}
	// Infer the parameter type as the typechecker would
	inferredTypes, err := tc.InferAssertionType(assertion, false)
	if err != nil || len(inferredTypes) == 0 {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	paramType := inferredTypes[0]
	paramTypeName := string(paramType.Ident)
	t.Logf("Inferred parameter type: %q", paramTypeName)

	// Register the inferred type in tc.Defs (simulate real codegen)
	// (In real pipeline, this would be done by typechecker during inference)
	shapeType := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx":   {Type: &ast.TypeNode{Ident: ast.TypeIdent("AppContext")}},
			"input": {Type: &ast.TypeNode{Ident: ast.TypeIdent("T_azh9nsqmxaF")}},
		},
	}
	tc.Defs[ast.TypeIdent(paramTypeName)] = ast.TypeDefNode{
		Ident: ast.TypeIdent(paramTypeName),
		Expr:  ast.TypeDefShapeExpr{Shape: *shapeType},
	}
	_ = tr.defineShapeType(shapeType) // ensure the transformer emits the type

	// Debug: Check what getGeneratedTypeNameForTypeNode returns for the parameter type
	generatedTypeName, err := tr.getGeneratedTypeNameForTypeNode(paramType)
	if err != nil {
		t.Logf("getGeneratedTypeNameForTypeNode error: %v", err)
	} else {
		t.Logf("getGeneratedTypeNameForTypeNode returned: %q", generatedTypeName)
	}

	// Debug: Check what's in tc.Defs
	t.Logf("tc.Defs contains %d entries:", len(tc.Defs))
	for key, value := range tc.Defs {
		t.Logf("  %q: %T", key, value)
	}

	// Debug: Check if the parameter type is actually in tc.Defs
	if _, exists := tc.Defs[ast.TypeIdent(paramTypeName)]; exists {
		t.Logf("Parameter type %q IS in tc.Defs", paramTypeName)
	} else {
		t.Logf("Parameter type %q is NOT in tc.Defs", paramTypeName)
	}

	// Register a function signature expecting the inferred assertion type
	funcName := ast.Identifier("f")
	tc.Functions[funcName] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: funcName},
		Parameters: []typechecker.ParameterSignature{{
			Ident: ast.Ident{ID: ast.Identifier("arg")},
			Type:  paramType,
		}},
		ReturnTypes: nil,
	}

	// Create a function call with a shape literal as the argument
	// Use a shape literal that is structurally different from the parameter type
	// to force the transformer to use a hash-based type
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"sessionId": {Assertion: &ast.AssertionNode{Constraints: []ast.ConstraintNode{{
							Name: "Value",
							Args: []ast.ConstraintArgumentNode{{
								Value: func() *ast.ValueNode { v := ast.ValueNode(ast.StringLiteralNode{Value: "bar"}); return &v }(),
							}},
						}}}},
					},
				},
			},
			"input": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {Assertion: &ast.AssertionNode{Constraints: []ast.ConstraintNode{{
							Name: "Value",
							Args: []ast.ConstraintArgumentNode{{
								Value: func() *ast.ValueNode { v := ast.ValueNode(ast.StringLiteralNode{Value: "Alice"}); return &v }(),
							}},
						}}}},
					},
				},
			},
		},
	}
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: funcName},
		Arguments: []ast.ExpressionNode{shapeLiteral},
	}

	// Debug: Check what type the shape literal would get
	shapeHash, err := tc.Hasher.HashNode(shapeLiteral)
	if err == nil {
		shapeTypeName := string(shapeHash.ToTypeIdent())
		t.Logf("Shape literal would get type: %q", shapeTypeName)
	}

	// Transform the function call
	stmt, err := tr.transformStatement(call)
	if err != nil {
		t.Fatalf("transformStatement failed: %v", err)
	}

	// Check that the generated Go code uses the inferred parameter type (which is defined)
	callExpr, ok := stmt.(*goast.ExprStmt)
	if !ok {
		t.Fatalf("Expected ExprStmt, got %T", stmt)
	}
	goCall, ok := callExpr.X.(*goast.CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", callExpr.X)
	}
	if len(goCall.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(goCall.Args))
	}
	compLit, ok := goCall.Args[0].(*goast.CompositeLit)
	if !ok {
		t.Fatalf("Expected CompositeLit as argument, got %T", goCall.Args[0])
	}
	ident, ok := compLit.Type.(*goast.Ident)
	if !ok {
		t.Fatalf("Expected Ident as composite literal type, got %T", compLit.Type)
	}
	t.Logf("Generated composite literal type: %q", ident.Name)

	if ident.Name != paramTypeName {
		t.Errorf("Expected composite literal type %q (the inferred parameter type), got %q (Go AST: %#v)", paramTypeName, ident.Name, compLit.Type)
	}
	if _, exists := tc.Defs[ast.TypeIdent(ident.Name)]; !exists {
		t.Errorf("Type %q used in composite literal is not defined in tc.Defs (bug: undefined type in output)", ident.Name)
	}
}
