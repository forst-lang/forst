package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// lookupFieldType looks up a field's type in a given type
func (tc *TypeChecker) lookupFieldType(baseType ast.TypeNode, fieldName ast.Ident, parentAssertion *ast.AssertionNode) (ast.TypeNode, error) {
	tc.log.Debugf("[lookupFieldType] Looking up field %s in type %s", fieldName, baseType.Ident)

	// Try type definition lookup
	if fieldType, err := tc.lookupFieldInTypeDef(baseType, fieldName); err == nil {
		return fieldType, nil
	}

	// Try type guard lookup
	if fieldType, err := tc.lookupFieldInTypeGuard(baseType, fieldName); err == nil {
		return fieldType, nil
	}

	// Try assertion chain lookup
	if fieldType, err := tc.lookupFieldInAssertion(baseType, fieldName, parentAssertion); err == nil {
		return fieldType, nil
	}

	// Don't throw error for Shape type
	if baseType.Ident == ast.TypeShape {
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}

	return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
}

// lookupFieldInTypeDef looks up a field in a type definition
func (tc *TypeChecker) lookupFieldInTypeDef(baseType ast.TypeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	def, exists := tc.Defs[baseType.Ident]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("type definition not found")
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		return ast.TypeNode{}, fmt.Errorf("not a type definition")
	}

	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return ast.TypeNode{}, fmt.Errorf("not a shape expression")
	}

	field, exists := shapeExpr.Shape.Fields[string(fieldName.ID)]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("field not found")
	}

	if field.Type != nil {
		return *field.Type, nil
	}

	if field.Assertion != nil && field.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
	}

	if field.Shape != nil {
		return tc.lookupNestedField(field.Shape, fieldName)
	}

	return ast.TypeNode{Ident: ast.TypeShape}, nil
}

// lookupFieldInTypeGuard looks up a field in a type guard
func (tc *TypeChecker) lookupFieldInTypeGuard(baseType ast.TypeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	guardDef, ok := tc.Defs[baseType.Ident].(ast.TypeGuardNode)
	if !ok {
		return ast.TypeNode{}, fmt.Errorf("not a type guard")
	}

	for _, param := range guardDef.Parameters() {
		if param.GetIdent() == string(fieldName.ID) {
			return param.GetType(), nil
		}
	}

	return ast.TypeNode{}, fmt.Errorf("field not found in type guard")
}

// lookupFieldInAssertion looks up a field in an assertion chain
func (tc *TypeChecker) lookupFieldInAssertion(baseType ast.TypeNode, fieldName ast.Ident, parentAssertion *ast.AssertionNode) (ast.TypeNode, error) {
	assertion := baseType.Assertion
	if assertion == nil {
		assertion = parentAssertion
	}
	if assertion == nil {
		return ast.TypeNode{}, fmt.Errorf("no assertion found")
	}

	mergedFields := tc.resolveShapeFieldsFromAssertion(assertion)
	field, exists := mergedFields[string(fieldName.ID)]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("field not found in assertion")
	}

	if field.Type != nil {
		return *field.Type, nil
	}

	if field.Assertion != nil && field.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
	}

	if field.Shape != nil {
		return tc.lookupNestedField(field.Shape, fieldName)
	}

	return ast.TypeNode{Ident: ast.TypeShape}, nil
}

// lookupNestedField recursively looks up a field in a nested shape
func (tc *TypeChecker) lookupNestedField(shape *ast.ShapeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	if shape == nil {
		return ast.TypeNode{}, fmt.Errorf("shape is nil")
	}

	field, exists := shape.Fields[string(fieldName.ID)]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("field %s not found in nested shape", fieldName)
	}

	if field.Assertion != nil && field.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
	}

	if field.Shape != nil {
		return tc.lookupNestedField(field.Shape, fieldName)
	}

	return ast.TypeNode{Ident: ast.TypeShape}, nil
}
