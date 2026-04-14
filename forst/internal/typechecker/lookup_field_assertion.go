package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// lookupFieldInAssertion looks up a field in an assertion chain
func (tc *TypeChecker) lookupFieldInAssertion(baseType ast.TypeNode, fieldName ast.Ident, parentAssertion *ast.AssertionNode) (ast.TypeNode, error) {
	assertion := baseType.Assertion
	if assertion == nil {
		assertion = parentAssertion
	}
	if assertion == nil {
		return ast.TypeNode{}, fmt.Errorf("no assertion found")
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInAssertion",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
		"assertion": fmt.Sprintf("%+v", assertion),
	}).Debugf("Looking up field in assertion")

	mergedFields := tc.resolveShapeFieldsFromAssertion(assertion)
	tc.log.WithFields(logrus.Fields{
		"function":     "lookupFieldInAssertion",
		"fieldName":    fieldName,
		"baseType":     baseType.Ident,
		"mergedFields": fmt.Sprintf("%+v", mergedFields),
	}).Debugf("Resolved shape fields from assertion")

	// Support dot-paths by splitting and recursing
	fieldPath := strings.Split(string(fieldName.ID), ".")
	return tc.lookupFieldPathOnMergedFields(mergedFields, fieldPath)
}

// lookupFieldPathOnMergedFields recursively looks up a field path in a merged fields map
func (tc *TypeChecker) lookupFieldPathOnMergedFields(fields map[string]ast.ShapeFieldNode, fieldPath []string) (ast.TypeNode, error) {
	if len(fieldPath) == 0 {
		return ast.TypeNode{}, fmt.Errorf("empty field path")
	}
	field, exists := fields[fieldPath[0]]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("field %s not found in assertion fields", fieldPath[0])
	}
	if field.Type != nil && len(fieldPath) == 1 {
		// Resolve type aliases even for single-segment paths
		return tc.resolveTypeAliasChain(*field.Type), nil
	}
	if field.Shape != nil && len(fieldPath) > 1 {
		return tc.lookupFieldPathOnShape(field.Shape, fieldPath[1:])
	}
	if field.Shape != nil && len(fieldPath) == 1 {
		// Single-segment path with shape field - return shape type
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}
	if field.Type != nil {
		return *field.Type, nil
	}
	return ast.TypeNode{}, fmt.Errorf("field %s exists but is not a type or shape", fieldPath[0])
}
