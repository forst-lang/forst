package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/logger"
)

func TestLookupNestedFieldNilPointerDereference(t *testing.T) {
	// Create a minimal type checker
	tc := &TypeChecker{
		log: logger.New(),
	}

	// Create a shape with a nested field that should cause the nil pointer dereference
	// This mimics the problematic shape: {name: String}
	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {
				// This field is intentionally nil to reproduce the issue
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}

	// This should NOT panic and should return an error instead
	fieldName := ast.Ident{ID: "name"}
	result, err := tc.lookupNestedField(shape, fieldName)

	// The function should handle the nil field gracefully
	// Either return an error or handle the nil case properly
	if err != nil {
		t.Logf("Expected error: %v", err)
	} else {
		t.Logf("Result: %+v", result)
	}

	// The test should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Function panicked: %v", r)
		}
	}()
	tc.lookupNestedField(shape, fieldName)
}

func TestLookupNestedFieldWithNilShapeField(t *testing.T) {
	// Create a minimal type checker
	tc := &TypeChecker{
		log: logger.New(),
	}

	// Create a shape with a nil field to reproduce the exact issue
	// Note: We can't create a nil ShapeFieldNode directly, but we can test the edge case
	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {
				// Empty ShapeFieldNode to test edge case
			},
		},
	}

	fieldName := ast.Ident{ID: "name"}

	// This should handle the empty field gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Function panicked: %v", r)
		}
	}()
	_, err := tc.lookupNestedField(shape, fieldName)
	// Should return an error for empty field
	if err == nil {
		t.Error("Expected error for empty field, got nil")
	}
}

func TestLookupNestedFieldShapeFieldAccess(t *testing.T) {
	// Create a minimal type checker
	tc := &TypeChecker{
		log: logger.New(),
	}

	// Create a shape that mimics the problematic case from the real code
	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}

	fieldName := ast.Ident{ID: "name"}

	// Test that accessing the field doesn't cause a panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Function panicked: %v", r)
		}
	}()
	result, err := tc.lookupNestedField(shape, fieldName)
	if err == nil {
		if result.Ident != ast.TypeString {
			t.Errorf("Expected TypeString, got %s", result.Ident)
		}
	}
}

func TestLookupNestedFieldPathTraversal(t *testing.T) {
	tc := &TypeChecker{
		log: logger.New(),
	}

	// Simulate: type Outer = { inner: { name: String } }
	innerShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	outerShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"inner": {Shape: innerShape},
		},
	}

	// Simulate lookup for outer.inner.name
	// First, lookup 'inner' in outerShape
	field, exists := outerShape.Fields["inner"]
	if !exists {
		t.Fatal("outerShape should have 'inner' field")
	}
	if field.Shape == nil {
		t.Fatal("'inner' field should be a shape")
	}

	// Now, lookup 'name' in innerShape
	nameFieldName := ast.Ident{ID: "name"}
	result, err := tc.lookupNestedField(field.Shape, nameFieldName)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result.Ident != ast.TypeString {
		t.Errorf("Expected TypeString, got %s", result.Ident)
	}
}

func TestLookupFieldInAssertionType(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: make(map[ast.TypeIdent]ast.Node),
	}

	// Simulate the type definitions from the integration test
	// type MutationArg = Shape
	// type AppMutation = MutationArg.Context(AppContext)

	// Create the nested shape: {name: String}
	nameShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}

	// Create the full shape: {ctx: AppContext, input: {name: String}}
	fullShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx":   {Type: &ast.TypeNode{Ident: "AppContext"}},
			"input": {Shape: nameShape},
		},
	}

	// Register the type definition (simulating T_488eVThFocF)
	typeIdent := ast.TypeIdent("T_488eVThFocF")
	tc.Defs[typeIdent] = ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefShapeExpr{
			Shape: *fullShape,
		},
	}

	t.Logf("=== DEBUG: Test Setup ===")
	t.Logf("Type definition registered: %s", typeIdent)
	t.Logf("Full shape fields: %+v", fullShape.Fields)
	t.Logf("Nested shape fields: %+v", nameShape.Fields)

	// Test lookup for "input" field
	inputFieldName := ast.Ident{ID: "input"}
	t.Logf("=== DEBUG: Looking up 'input' field ===")
	t.Logf("Looking up field '%s' in type %s", inputFieldName.ID, typeIdent)

	result, err := tc.lookupFieldInTypeDef(ast.TypeNode{Ident: typeIdent}, inputFieldName)
	if err != nil {
		t.Logf("ERROR looking up 'input': %v", err)
		t.Fatalf("Expected no error, got: %v", err)
	} else {
		t.Logf("SUCCESS: 'input' field result: %+v", result)
	}
	if result.Ident == "" {
		t.Error("Expected non-empty result")
	}

	// The result should be a shape type that we can then lookup "name" in
	// This simulates the second part of op.input.name lookup
	nameFieldName := ast.Ident{ID: "name"}
	t.Logf("=== DEBUG: Looking up 'name' field in nested shape ===")
	t.Logf("Looking up field '%s' in shape: %+v", nameFieldName.ID, nameShape)

	nestedResult, err := tc.lookupNestedField(nameShape, nameFieldName)
	if err != nil {
		t.Logf("ERROR looking up 'name': %v", err)
		t.Fatalf("Expected no error, got: %v", err)
	} else {
		t.Logf("SUCCESS: 'name' field result: %+v", nestedResult)
	}
	if nestedResult.Ident != ast.TypeString {
		t.Errorf("Expected TypeString, got %s", nestedResult.Ident)
	}
}

func TestLookupFieldPath_NestedShape(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: make(map[ast.TypeIdent]ast.Node),
	}

	// Create the nested shape: {name: String}
	nameShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	// Create the full shape: {input: {name: String}}
	fullShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"input": {Shape: nameShape},
		},
	}
	// Register the type definition
	typeIdent := ast.TypeIdent("T_ShapeWithInput")
	tc.Defs[typeIdent] = ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefShapeExpr{
			Shape: *fullShape,
		},
	}

	// Try to resolve ["input", "name"]
	result, err := tc.lookupFieldPath(ast.TypeNode{Ident: typeIdent}, []string{"input", "name"})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Ident != ast.TypeString {
		t.Errorf("Expected TypeString, got %s", result.Ident)
	}
}

func TestLookupFieldInAssertionType_FieldExistsButNotType(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: make(map[ast.TypeIdent]ast.Node),
	}

	// Create the nested shape: {name: String}
	nameShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}

	// Create the full shape: {ctx: AppContext, input: {name: String}}
	fullShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx":   {Type: &ast.TypeNode{Ident: "AppContext"}},
			"input": {Shape: nameShape},
		},
	}

	// Register the type definition
	typeDef := ast.TypeDefNode{
		Ident: "T_488eVThFocF",
		Expr:  ast.TypeDefShapeExpr{Shape: *fullShape},
	}
	tc.Defs["T_488eVThFocF"] = typeDef

	// Register the AppContext type
	appContextShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {Type: &ast.TypeNode{Ident: ast.TypePointer}},
		},
	}
	appContextDef := ast.TypeDefNode{
		Ident: "AppContext",
		Expr:  ast.TypeDefShapeExpr{Shape: *appContextShape},
	}
	tc.Defs["AppContext"] = appContextDef

	// Test: Look up "ctx" field in the assertion type
	fieldName := ast.Ident{ID: "ctx"}
	result, err := tc.lookupFieldInTypeDef(ast.TypeNode{Ident: "T_488eVThFocF"}, fieldName)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
		return
	}

	// Should return AppContext type
	if result.Ident != "AppContext" {
		t.Errorf("Expected AppContext, got: %s", result.Ident)
	}
}

func TestLookupFieldInAssertionType_FromInferredType(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: make(map[ast.TypeIdent]ast.Node),
	}

	// Register the base types that would be created during inference
	// AppContext type
	appContextShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {Type: &ast.TypeNode{Ident: ast.TypePointer}},
		},
	}
	appContextDef := ast.TypeDefNode{
		Ident: "AppContext",
		Expr:  ast.TypeDefShapeExpr{Shape: *appContextShape},
	}
	tc.Defs["AppContext"] = appContextDef

	// The inferred assertion type T_488eVThFocF
	inferredShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx": {Type: &ast.TypeNode{Ident: "AppContext"}},
			"input": {Shape: &ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			}},
		},
	}
	inferredDef := ast.TypeDefNode{
		Ident: "T_488eVThFocF",
		Expr:  ast.TypeDefShapeExpr{Shape: *inferredShape},
	}
	tc.Defs["T_488eVThFocF"] = inferredDef

	// Test: Look up "ctx" field in the inferred assertion type
	fieldName := ast.Ident{ID: "ctx"}
	result, err := tc.lookupFieldInTypeDef(ast.TypeNode{Ident: "T_488eVThFocF"}, fieldName)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
		return
	}

	// Should return AppContext type
	if result.Ident != "AppContext" {
		t.Errorf("Expected AppContext, got: %s", result.Ident)
	}
}

func TestLookupFieldPath_SingleSegmentShapeField(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: make(map[ast.TypeIdent]ast.Node),
	}

	// Create a shape with a field that is itself a shape
	innerShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}

	outerShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"input": {Shape: innerShape}, // This field is a shape, not a type
		},
	}

	// Register the type definition
	typeDef := ast.TypeDefNode{
		Ident: "T_488eVThFocF",
		Expr:  ast.TypeDefShapeExpr{Shape: *outerShape},
	}
	tc.Defs["T_488eVThFocF"] = typeDef

	// Test: Look up "input" field (single segment) in the type
	// This should return a shape type, not fail with "field exists but is not a type or shape"
	result, err := tc.lookupFieldPath(ast.TypeNode{Ident: "T_488eVThFocF"}, []string{"input"})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
		return
	}

	// Should return a shape type
	if result.Ident != ast.TypeShape {
		t.Errorf("Expected Shape type, got: %s", result.Ident)
	}
}

func TestLookupFieldPath_TypeAliasToShape(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: make(map[ast.TypeIdent]ast.Node),
	}

	// Register AppContext as a shape with a field sessionId
	appContextShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	appContextDef := ast.TypeDefNode{
		Ident: "AppContext",
		Expr:  ast.TypeDefShapeExpr{Shape: *appContextShape},
	}
	tc.Defs["AppContext"] = appContextDef

	// Register outer shape with a field ctx: AppContext
	outerShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx": {Type: &ast.TypeNode{Ident: "AppContext"}},
		},
	}
	outerDef := ast.TypeDefNode{
		Ident: "Outer",
		Expr:  ast.TypeDefShapeExpr{Shape: *outerShape},
	}
	tc.Defs["Outer"] = outerDef

	// Try to look up ctx.sessionId on Outer
	result, err := tc.lookupFieldPath(ast.TypeNode{Ident: "Outer"}, []string{"ctx", "sessionId"})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
		return
	}
	if result.Ident != ast.TypeString {
		t.Errorf("Expected String type, got: %s", result.Ident)
	}
}
