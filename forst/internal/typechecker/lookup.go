package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// LookupInferredType looks up the inferred type of a node in the current scope
func (tc *TypeChecker) LookupInferredType(node ast.Node, requireInferred bool) ([]ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(node)
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) == 0 {
			if requireInferred {
				return nil, fmt.Errorf("expected type of node to have been inferred, found: implicit type")
			}
			return nil, nil
		}
		return existingType, nil
	}
	if requireInferred {
		return nil, fmt.Errorf("expected type of node to have been inferred, found: no registered type")
	}
	return nil, nil
}

// LookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode, scope *Scope) (ast.TypeNode, error) {
	tc.log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, scope.String())

	parts := strings.Split(string(variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])

	symbol, exists := scope.LookupVariable(baseIdent, true)
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("undefined symbol: %s [scope: %s]", parts[0], scope.String())
	}

	if len(symbol.Types) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type for variable %s but got %d types", parts[0], len(symbol.Types))
	}

	if len(parts) == 1 {
		return symbol.Types[0], nil
	}

	currentType := symbol.Types[0]
	originalAssertion := currentType.Assertion
	for i := 1; i < len(parts); i++ {
		fieldType, err := tc.lookupFieldType(currentType, ast.Ident{ID: ast.Identifier(parts[i])}, originalAssertion)
		if err != nil {
			return ast.TypeNode{}, err
		}
		currentType = fieldType
	}

	return currentType, nil
}

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

// LookupFunctionReturnType looks up the return type of a function node
func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Ident.ID]
	if !exists {
		return nil, fmt.Errorf("undefined function: %s", function.Ident)
	}
	return sig.ReturnTypes, nil
}

// LookupEnsureBaseType looks up the base type of an ensure node in a given scope
func (tc *TypeChecker) LookupEnsureBaseType(ensure *ast.EnsureNode, scope *Scope) (*ast.TypeNode, error) {
	baseType, err := tc.LookupVariableType(&ensure.Variable, scope)
	if err != nil {
		return nil, err
	}
	return &baseType, nil
}

// LookupAssertionType looks up the type of an assertion node
func (tc *TypeChecker) LookupAssertionType(assertion *ast.AssertionNode) (*ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(assertion)
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) != 1 {
			return nil, fmt.Errorf("expected single type for assertion %s but got %d types", hash.ToTypeIdent(), len(existingType))
		}
		return &existingType[0], nil
	}

	typeNode := &ast.TypeNode{
		Ident:     hash.ToTypeIdent(),
		Assertion: assertion,
	}
	tc.storeInferredType(assertion, []ast.TypeNode{*typeNode})
	return typeNode, nil
}
