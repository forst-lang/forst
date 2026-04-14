package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// lookupFieldInTypeDef looks up a field in a type definition
func (tc *TypeChecker) lookupFieldInTypeDef(baseType ast.TypeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
	}).Debugf("Looking up field in typeDef")

	def, exists := tc.Defs[baseType.Ident]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldInTypeDef",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
		}).Debugf("Type definition not found")
		return ast.TypeNode{}, fmt.Errorf("type definition not found")
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
		"defType":   fmt.Sprintf("%T", def),
		"def":       fmt.Sprintf("%+v", def),
	}).Debugf("Found type definition")

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldInTypeDef",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"defType":   fmt.Sprintf("%T", def),
		}).Debugf("Not a type definition")
		return ast.TypeNode{}, fmt.Errorf("not a type definition")
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
		"exprType":  fmt.Sprintf("%T", typeDef.Expr),
		"expr":      fmt.Sprintf("%+v", typeDef.Expr),
	}).Debugf("Type definition expression")

	shapePtr, ok := ast.PayloadShape(typeDef.Expr)
	if !ok {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldInTypeDef",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"exprType":  fmt.Sprintf("%T", typeDef.Expr),
		}).Debugf("Not a shape expression")
		return ast.TypeNode{}, fmt.Errorf("not a shape expression")
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
		"shape":     fmt.Sprintf("%+v", *shapePtr),
		"fields":    fmt.Sprintf("%+v", shapePtr.Fields),
	}).Debugf("Shape expression fields")

	field, exists := shapePtr.Fields[string(fieldName.ID)]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function":        "lookupFieldInTypeDef",
			"fieldName":       fieldName,
			"baseType":        baseType.Ident,
			"availableFields": fmt.Sprintf("%+v", shapePtr.Fields),
		}).Debugf("Field not found in shape")
		return ast.TypeNode{}, fmt.Errorf("field not found")
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
		"field":     fmt.Sprintf("%+v", field),
	}).Debugf("Found field in shape")

	if field.Type != nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldInTypeDef",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"fieldType": field.Type.Ident,
		}).Debugf("Returning field type")
		return *field.Type, nil
	}

	if field.Assertion != nil && field.Assertion.BaseType != nil {
		tc.log.WithFields(logrus.Fields{
			"function":          "lookupFieldInTypeDef",
			"fieldName":         fieldName,
			"baseType":          baseType.Ident,
			"assertionBaseType": *field.Assertion.BaseType,
		}).Debugf("Returning assertion base type")
		return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
	}

	if field.Shape != nil {
		tc.log.WithFields(logrus.Fields{
			"function":   "lookupFieldInTypeDef",
			"fieldName":  fieldName,
			"baseType":   baseType.Ident,
			"fieldShape": fmt.Sprintf("%+v", field.Shape),
		}).Debugf("Looking up nested field")
		// If the fieldName contains dots, split and use lookupFieldPath for the remainder
		fieldPath := strings.Split(string(fieldName.ID), ".")
		if len(fieldPath) > 1 {
			// The first segment matched this field; pass the rest to lookupFieldPath on the nested shape
			// We need a TypeNode for the nested shape; use a synthetic one for now
			return tc.lookupFieldPath(ast.TypeNode{Ident: ast.TypeShape}, fieldPath[1:])
		}
		// For single-segment field names, return the shape type directly
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}

	// Handle Value constraints (like id: query.id)
	if field.Assertion != nil && len(field.Assertion.Constraints) > 0 && field.Assertion.Constraints[0].Name == ast.ValueConstraint {
		return tc.inferValueConstraintType(field.Assertion.Constraints[0], string(fieldName.ID), nil)
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
	}).Debugf("Returning Shape type as fallback")
	return ast.TypeNode{Ident: ast.TypeShape}, nil
}
