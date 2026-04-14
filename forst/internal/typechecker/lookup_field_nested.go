package typechecker

import (
	"fmt"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// lookupNestedField recursively looks up a field in a nested shape
func (tc *TypeChecker) lookupNestedField(shape *ast.ShapeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":  "lookupNestedField",
		"fieldName": fieldName,
		"shape":     fmt.Sprintf("%+v", shape),
	}).Debugf("Looking up nested field")

	if shape == nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupNestedField",
			"fieldName": fieldName,
		}).Debugf("Shape is nil")
		return ast.TypeNode{}, fmt.Errorf("shape is nil")
	}

	tc.log.WithFields(logrus.Fields{
		"function":    "lookupNestedField",
		"fieldName":   fieldName,
		"shapeFields": fmt.Sprintf("%+v", shape.Fields),
	}).Debugf("Shape fields")

	field, exists := shape.Fields[string(fieldName.ID)]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function":        "lookupNestedField",
			"fieldName":       fieldName,
			"availableFields": fmt.Sprintf("%+v", shape.Fields),
		}).Debugf("Field not found in nested shape")
		return ast.TypeNode{}, fmt.Errorf("field %s not found in nested shape", fieldName.ID)
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupNestedField",
		"fieldName": fieldName,
		"field":     fmt.Sprintf("%+v", field),
	}).Debugf("Found field in nested shape")

	if field.Type != nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupNestedField",
			"fieldName": fieldName,
			"fieldType": field.Type.Ident,
		}).Debugf("Returning field type")
		return *field.Type, nil
	}

	if field.Assertion != nil && field.Assertion.BaseType != nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupNestedField",
			"fieldName": fieldName,
			"baseType":  *field.Assertion.BaseType,
		}).Debugf("Returning assertion base type")
		return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
	}

	if field.Shape != nil {
		tc.log.WithFields(logrus.Fields{
			"function":    "lookupNestedField",
			"fieldName":   fieldName,
			"nestedShape": fmt.Sprintf("%+v", field.Shape),
		}).Debugf("Recursing into nested shape")
		return tc.lookupNestedField(field.Shape, fieldName)
	}

	// If the field exists but is empty, return an error
	tc.log.WithFields(logrus.Fields{
		"function":  "lookupNestedField",
		"fieldName": fieldName,
		"field":     fmt.Sprintf("%+v", field),
	}).Debugf("Field exists but is empty (no type, shape, or assertion)")
	return ast.TypeNode{}, fmt.Errorf("field %s exists but is empty (no type, shape, or assertion)", fieldName.ID)
}
