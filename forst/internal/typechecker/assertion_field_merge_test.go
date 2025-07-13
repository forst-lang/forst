package typechecker

import (
	"forst/internal/ast"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAssertionFieldMerging_SimpleExample(t *testing.T) {
	// This test reproduces the assertion field merging bug with a simple example
	// The issue is that when we have BaseType.Constraint(Shape), the typechecker
	// should merge fields from both the base type and the constraint

	// Setup: Create simple type definitions
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	tc := New(logger, false)

	// Define a base shape type: Person with name field
	personShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}
	tc.registerShapeType("Person", personShape)

	// Define a constraint that adds an address field
	addressConstraint := ast.TypeGuardNode{
		Ident: "WithAddress",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "p"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Person")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "address"},
				Type:  ast.TypeNode{Ident: ast.TypeString},
			},
		},
		Body: []ast.Node{},
	}

	// Register the type guard
	tc.Defs["WithAddress"] = &addressConstraint

	// Define the assertion: Person.WithAddress(String)
	// This should merge:
	// - name: String (from base type Person)
	// - address: String (from WithAddress constraint)
	personType := ast.TypeIdent("Person")
	assertion := ast.AssertionNode{
		BaseType: &personType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "WithAddress",
				Args: []ast.ConstraintArgumentNode{
					{
						Type: &ast.TypeNode{Ident: ast.TypeString},
					},
				},
			},
		},
	}

	// Test: Infer the type of the assertion
	inferredTypes, err := tc.InferAssertionType(&assertion, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}

	if len(inferredTypes) == 0 {
		t.Fatal("Expected at least one inferred type")
	}

	// Get the inferred shape type
	inferredType := inferredTypes[0]

	// Look up the shape definition by type ident
	def, ok := tc.Defs[inferredType.Ident]
	if !ok {
		t.Fatalf("No type definition found for inferred type ident: %s", inferredType.Ident)
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Type definition is not a TypeDefNode: %T", def)
	}

	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Type definition expr is not a TypeDefShapeExpr: %T", typeDef.Expr)
	}

	shape := shapeExpr.Shape

	// Debug: Print the actual field names in the merged shape
	t.Logf("DEBUG: Merged shape fields: %v", shape.Fields)
	for fieldName, fieldNode := range shape.Fields {
		t.Logf("DEBUG: Field '%s' has type: %v", fieldName, fieldNode.Type)
	}

	// The shape should have both name and address fields
	if _, hasName := shape.Fields["name"]; !hasName {
		t.Error("Expected 'name' field from base type")
	}
	if _, hasAddress := shape.Fields["address"]; !hasAddress {
		t.Error("Expected 'address' field from WithAddress constraint")
	}

	// Verify both fields are present
	expectedFields := []string{"name", "address"}
	for _, fieldName := range expectedFields {
		if _, exists := shape.Fields[fieldName]; !exists {
			t.Errorf("Missing expected field: %s", fieldName)
		}
	}
	if len(shape.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(shape.Fields))
	}
}

func TestAssertionFieldMerging_ShapeLiteralArgument(t *testing.T) {
	// This test demonstrates the bug: when a constraint argument is a shape literal,
	// the merged type should include the fields from the shape literal, but currently does not.
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	tc := New(logger, false)

	// Base type: Thing with id field
	thingShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id": {
				Type: &ast.TypeNode{Ident: ast.TypeInt},
			},
		},
	}
	tc.registerShapeType("Thing", thingShape)

	// Constraint: WithStuff, takes a shape argument
	withStuffConstraint := ast.TypeGuardNode{
		Ident: "WithStuff",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "t"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Thing")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "stuff"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}
	tc.Defs["WithStuff"] = &withStuffConstraint

	// Assertion: Thing.WithStuff({ foo: Int })
	thingType := ast.TypeIdent("Thing")
	assertion := ast.AssertionNode{
		BaseType: &thingType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "WithStuff",
				Args: []ast.ConstraintArgumentNode{
					{
						Shape: &ast.ShapeNode{
							Fields: map[string]ast.ShapeFieldNode{
								"foo": {
									Type: &ast.TypeNode{Ident: ast.TypeInt},
								},
							},
						},
					},
				},
			},
		},
	}

	inferredTypes, err := tc.InferAssertionType(&assertion, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	if len(inferredTypes) == 0 {
		t.Fatal("Expected at least one inferred type")
	}
	inferredType := inferredTypes[0]
	def, ok := tc.Defs[inferredType.Ident]
	if !ok {
		t.Fatalf("No type definition found for inferred type ident: %s", inferredType.Ident)
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Type definition is not a TypeDefNode: %T", def)
	}
	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Type definition expr is not a TypeDefShapeExpr: %T", typeDef.Expr)
	}
	shape := shapeExpr.Shape

	// Debug: Print the actual field names in the merged shape
	t.Logf("DEBUG: Merged shape fields: %v", shape.Fields)
	for fieldName, fieldNode := range shape.Fields {
		t.Logf("DEBUG: Field '%s' has type: %v", fieldName, fieldNode.Type)
	}

	// The shape should have both id and foo fields
	if _, hasId := shape.Fields["id"]; !hasId {
		t.Error("Expected 'id' field from base type")
	}
	if _, hasFoo := shape.Fields["foo"]; !hasFoo {
		t.Error("Expected 'foo' field from shape literal argument (BUG: this will fail)")
	}
	expectedFields := []string{"id", "foo"}
	for _, fieldName := range expectedFields {
		if _, exists := shape.Fields[fieldName]; !exists {
			t.Errorf("Missing expected field: %s", fieldName)
		}
	}
	if len(shape.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(shape.Fields))
	}
}

func TestAssertionFieldMerging_OneLevelTooDeep(t *testing.T) {
	// This test reproduces the "one level too deep" bug where shape literals
	// create nested shape structures instead of flat ones.
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	tc := New(logger, false)

	// Base type: Operation with op field
	operationShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"op": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}
	tc.registerShapeType("Operation", operationShape)

	// Constraint: Match, takes a shape argument
	matchConstraint := ast.TypeGuardNode{
		Ident: "Match",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "op"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Operation")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}
	tc.Defs["Match"] = &matchConstraint

	// Assertion: Operation.Match({ name: String })
	operationType := ast.TypeIdent("Operation")
	assertion := ast.AssertionNode{
		BaseType: &operationType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "Match",
				Args: []ast.ConstraintArgumentNode{
					{
						Shape: &ast.ShapeNode{
							Fields: map[string]ast.ShapeFieldNode{
								"name": {
									Type: &ast.TypeNode{Ident: ast.TypeString},
								},
							},
						},
					},
				},
			},
		},
	}

	inferredTypes, err := tc.InferAssertionType(&assertion, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}
	if len(inferredTypes) == 0 {
		t.Fatal("Expected at least one inferred type")
	}
	inferredType := inferredTypes[0]
	def, ok := tc.Defs[inferredType.Ident]
	if !ok {
		t.Fatalf("No type definition found for inferred type ident: %s", inferredType.Ident)
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Type definition is not a TypeDefNode: %T", def)
	}
	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Type definition expr is not a TypeDefShapeExpr: %T", typeDef.Expr)
	}
	shape := shapeExpr.Shape

	// Debug: Print the actual field names in the merged shape
	t.Logf("DEBUG: Merged shape fields: %v", shape.Fields)
	for fieldName, fieldNode := range shape.Fields {
		t.Logf("DEBUG: Field '%s' has type: %v", fieldName, fieldNode.Type)
	}

	// The shape should have both op and name fields (name comes from the shape literal)
	if _, hasOp := shape.Fields["op"]; !hasOp {
		t.Error("Expected 'op' field from base type")
	}
	if _, hasName := shape.Fields["name"]; !hasName {
		t.Error("Expected 'name' field from shape literal in Match constraint")
	}

	// Check that name field has the correct type (should be String)
	nameField := shape.Fields["name"]
	if nameField.Type == nil {
		t.Fatal("Name field type is nil")
	}

	// The name field should be String type
	if nameField.Type.Ident != ast.TypeString {
		t.Fatalf("BUG: Name field is %s type instead of String. This indicates the shape literal is being treated as a nested shape. Expected: name: String, Got: name: %s", nameField.Type.Ident, nameField.Type.Ident)
	}

	expectedFields := []string{"op", "name"}
	for _, fieldName := range expectedFields {
		if _, exists := shape.Fields[fieldName]; !exists {
			t.Errorf("Missing expected field: %s", fieldName)
		}
	}
	if len(shape.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(shape.Fields))
	}
}

// This test reproduces the bug in the shape guard example where a shape literal constraint argument
// should unwrap nested shape fields into the parent shape, not create a nested structure.
func TestAssertionFieldMerging_UnwrapsNestedShapeFields(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	tc := New(logger, false)

	// Base type: Request with user field
	requestShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"user": {
				Type: &ast.TypeNode{Ident: "User"},
			},
		},
	}
	tc.registerShapeType("Request", requestShape)

	// Constraint: WithData, takes a shape argument
	dataConstraint := ast.TypeGuardNode{
		Ident: ast.Identifier("WithData"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("r")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Request")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("data")},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}
	tc.Defs["WithData"] = &dataConstraint

	// Create the assertion: Request.WithData({ data: { name: String } })
	// This should result in a shape with fields: user, data
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"data": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {
							Type: &ast.TypeNode{Ident: ast.TypeString},
						},
					},
				},
			},
		},
	}

	requestType := ast.TypeIdent("Request")
	assertion := ast.AssertionNode{
		BaseType: &requestType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "WithData",
				Args: []ast.ConstraintArgumentNode{
					{
						Shape: &shapeLiteral,
					},
				},
			},
		},
	}

	inferredTypes, err := tc.InferAssertionType(&assertion, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer assertion type: %v", err)
	}

	if len(inferredTypes) == 0 {
		t.Fatal("Expected at least one inferred type")
	}

	inferredType := inferredTypes[0]
	def, ok := tc.Defs[inferredType.Ident]
	if !ok {
		t.Fatalf("No type definition found for inferred type ident: %s", inferredType.Ident)
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Type definition is not a TypeDefNode: %T", def)
	}

	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Type definition expr is not a TypeDefShapeExpr: %T", typeDef.Expr)
	}

	shape := shapeExpr.Shape

	// Debug: Print the actual field names in the merged shape
	t.Logf("DEBUG: Merged shape fields: %v", shape.Fields)
	for fieldName, fieldNode := range shape.Fields {
		t.Logf("DEBUG: Field '%s' has type: %v", fieldName, fieldNode.Type)
	}

	// The shape should have both user and data fields (data should be a shape)
	if _, hasUser := shape.Fields["user"]; !hasUser {
		t.Error("Expected 'user' field from base type")
	}
	if _, hasData := shape.Fields["data"]; !hasData {
		t.Error("Expected 'data' field from WithData constraint")
	}

	// Check that data field has the correct type (should be a shape)
	dataField := shape.Fields["data"]
	if dataField.Type == nil {
		t.Fatal("Data field type is nil")
	}

	// The data field should be a shape type
	if dataField.Type.Ident == ast.TypeShape {
		if dataField.Shape == nil {
			t.Fatal("Data field has TypeShape but no Shape node")
		}
		// Check that the shape has the correct nested field: 'name'
		if _, hasName := dataField.Shape.Fields["name"]; !hasName {
			t.Errorf("BUG: data field's shape should have 'name' field, but it's missing")
		}
		// The shape should NOT have a 'data' field (that would be wrong)
		if _, hasData := dataField.Shape.Fields["data"]; hasData {
			t.Errorf("BUG: data field's shape should not have 'data' field")
		}
	} else {
		// If the type is a hash type, try to resolve its fields
		dataTypeDef, ok := tc.Defs[dataField.Type.Ident].(ast.TypeDefNode)
		if !ok {
			t.Fatalf("BUG: data field type %s is not a TypeDefNode", dataField.Type.Ident)
		}
		shapeExpr, ok := dataTypeDef.Expr.(ast.TypeDefShapeExpr)
		if !ok {
			t.Fatalf("BUG: data field type %s is not a TypeDefShapeExpr", dataField.Type.Ident)
		}
		// Check that the shape has the correct nested field: 'name'
		if _, hasName := shapeExpr.Shape.Fields["name"]; !hasName {
			t.Errorf("BUG: data field's shape should have 'name' field, but it's missing")
		}
		// The shape should NOT have a 'data' field (that would be wrong)
		if _, hasData := shapeExpr.Shape.Fields["data"]; hasData {
			t.Errorf("BUG: data field's shape should not have 'data' field")
		}
	}
}

// Simulate the integration bug: create a type assertion with a Match constraint
// Base type: Shape(?), Constraint: Match({input: Shape})
func TestAssertionFieldMerging_IntegrationBug(t *testing.T) {
	// This test reproduces the integration bug where assertion field merging
	// fails when the constraint argument is a shape literal
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	tc := New(logger, false)

	// Define the base type: AppMutation
	appMutationShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}

	// Create type definitions for the types we need
	appMutationDef := ast.TypeDefNode{
		Ident: "AppMutation",
		Expr: ast.TypeDefShapeExpr{
			Shape: appMutationShape,
		},
	}

	// Define Shape type for the test
	shapeDef := ast.TypeDefNode{
		Ident: "Shape",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{},
			},
		},
	}

	// Define the constraint: Match({input: Shape})
	matchShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"input": {Type: &ast.TypeNode{Ident: ast.TypeShape}},
		},
	}
	baseType := ast.TypeIdent("Shape")
	matchConstraint := &ast.AssertionNode{
		BaseType: &baseType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "Match",
				Args: []ast.ConstraintArgumentNode{{Shape: matchShape}},
			},
		},
	}

	// Use CheckTypes to register the type definitions properly
	err := tc.CheckTypes([]ast.Node{appMutationDef, shapeDef})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Infer the type for the assertion
	inferredTypes, err := tc.InferAssertionType(matchConstraint, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer type for assertion: %v", err)
	}
	if len(inferredTypes) == 0 {
		t.Fatal("No types inferred for assertion")
	}
	// Get the merged shape
	mergedTypeIdent := inferredTypes[0].Ident
	def, ok := tc.Defs[mergedTypeIdent].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("BUG: merged type %s is not a TypeDefNode", mergedTypeIdent)
	}
	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("BUG: merged type %s is not a TypeDefShapeExpr", mergedTypeIdent)
	}
	// The merged shape should have an 'input' field
	inputField, hasInput := shapeExpr.Shape.Fields["input"]
	if !hasInput {
		t.Fatalf("Expected 'input' field in merged shape")
	}
	// The input field should be a generic Shape, not a concrete type
	if inputField.Type == nil || (inputField.Type.Ident != ast.TypeShape && inputField.Type.Ident != ast.TypeIdent("Shape")) {
		t.Fatalf("Expected input field to have generic Shape type, got: %v", inputField.Type)
	}
	// There should NOT be a concrete type definition for the input field
	if _, ok := tc.Defs[inputField.Type.Ident]; ok {
		t.Errorf("BUG: input field type %s should not have a concrete type definition", inputField.Type.Ident)
	} else {
		t.Logf("GOOD: input field is generic Shape, no concrete type definition (matches integration behavior)")
	}
}

// Test that reproduces the exact integration bug from shape_guard.ft
func TestAssertionFieldMerging_ShapeGuardIntegration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	tc := New(logger, false)

	// Define MutationArg = Shape
	mutationArgShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{},
	}
	tc.registerShapeType("MutationArg", mutationArgShape)

	// Define Input constraint: is (m MutationArg) Input(input Shape)
	inputConstraint := ast.TypeGuardNode{
		Ident: ast.Identifier("Input"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("m")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("MutationArg")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("input")},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}
	tc.Defs["Input"] = &inputConstraint

	// Define Context constraint: is (m MutationArg) Context(ctx Shape)
	contextConstraint := ast.TypeGuardNode{
		Ident: ast.Identifier("Context"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("m")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("MutationArg")},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("ctx")},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{},
	}
	tc.Defs["Context"] = &contextConstraint

	// Define AppContext = { sessionId: *String, user: *User }
	refType := ast.TypeIdent("Ref")
	appContextShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {Type: &ast.TypeNode{Ident: ast.TypeString, Assertion: &ast.AssertionNode{BaseType: &refType}}},
			"user":      {Type: &ast.TypeNode{Ident: ast.TypeIdent("User")}},
		},
	}
	tc.registerShapeType("AppContext", appContextShape)

	// Define AppMutation = MutationArg.Context(AppContext)
	// This creates a type with ctx field
	mutationArgType := ast.TypeIdent("MutationArg")
	appContextType := ast.TypeIdent("AppContext")
	appMutationAssertion := ast.AssertionNode{
		BaseType: &mutationArgType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "Context",
				Args: []ast.ConstraintArgumentNode{
					{Type: &ast.TypeNode{Ident: appContextType}},
				},
			},
		},
	}
	appMutationTypes, err := tc.InferAssertionType(&appMutationAssertion, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer AppMutation type: %v", err)
	}
	appMutationType := appMutationTypes[0].Ident

	// Now create the assertion: AppMutation.Input({input: {name: String}})
	// This should add an input field with a shape containing name: String
	inputShapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"input": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			},
		},
	}

	finalAssertion := ast.AssertionNode{
		BaseType: &appMutationType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "Input",
				Args: []ast.ConstraintArgumentNode{
					{Shape: &inputShapeLiteral},
				},
			},
		},
	}

	inferredTypes, err := tc.InferAssertionType(&finalAssertion, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to infer final assertion type: %v", err)
	}

	if len(inferredTypes) == 0 {
		t.Fatal("Expected at least one inferred type")
	}

	finalType := inferredTypes[0]
	def, ok := tc.Defs[finalType.Ident].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("No type definition found for final type: %s", finalType.Ident)
	}

	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Type definition expr is not a TypeDefShapeExpr: %T", def.Expr)
	}

	// The final shape should have ctx (from AppContext), and input fields
	if _, hasCtx := shapeExpr.Shape.Fields["ctx"]; !hasCtx {
		t.Error("Expected 'ctx' field from AppContext")
	}
	if _, hasInput := shapeExpr.Shape.Fields["input"]; !hasInput {
		t.Error("Expected 'input' field from Input constraint")
	}

	// Check that ctx field has the correct type (should be Shape(?) for generic shape parameter)
	ctxField := shapeExpr.Shape.Fields["ctx"]
	if ctxField.Type == nil && ctxField.Assertion == nil {
		t.Fatal("Ctx field has neither Type nor Assertion")
	}
	if ctxField.Type != nil && ctxField.Type.Ident != ast.TypeShape {
		t.Errorf("Expected ctx field to have type Shape(?), got %s", ctxField.Type.Ident)
	}
	if ctxField.Type == nil && ctxField.Assertion != nil {
		t.Logf("Ctx field is generic shape with only assertion (expected for generic Shape param)")
	}

	// Check that input field has the correct nested structure
	inputField := shapeExpr.Shape.Fields["input"]
	if inputField.Type == nil {
		t.Fatal("Input field type is nil")
	}

	// The input field should be a shape type with name field
	if inputField.Type.Ident == ast.TypeShape {
		if inputField.Shape == nil {
			t.Fatal("Input field has TypeShape but no Shape node")
		}
		if _, hasName := inputField.Shape.Fields["name"]; !hasName {
			t.Errorf("BUG: input field's shape should have 'name' field")
		}
	} else {
		// If it's a hash type, check the resolved type
		inputTypeDef, ok := tc.Defs[inputField.Type.Ident].(ast.TypeDefNode)
		if !ok {
			t.Fatalf("BUG: input field type %s is not a TypeDefNode", inputField.Type.Ident)
		}
		inputShapeExpr, ok := inputTypeDef.Expr.(ast.TypeDefShapeExpr)
		if !ok {
			t.Fatalf("BUG: input field type %s is not a TypeDefShapeExpr", inputField.Type.Ident)
		}
		if _, hasName := inputShapeExpr.Shape.Fields["name"]; !hasName {
			t.Errorf("BUG: input field's shape should have 'name' field")
		}
	}
}
