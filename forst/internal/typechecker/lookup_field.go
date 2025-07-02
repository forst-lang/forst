package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// lookupFieldType looks up a field's type in a given type
func (tc *TypeChecker) lookupFieldType(baseType ast.TypeNode, fieldName ast.Ident, parentAssertion *ast.AssertionNode) (ast.TypeNode, error) {
	tc.log.Debugf("[lookupFieldType] Looking up field %s in type %s", fieldName, baseType.Ident)

	// Add detailed debugging for the lookup process
	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldType",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
	}).Debugf("=== FIELD LOOKUP DEBUG ===")

	// Try type definition lookup
	if fieldType, err := tc.lookupFieldInTypeDef(baseType, fieldName); err == nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"result":    "found in typeDef",
		}).Debugf("Field found in typeDef")
		return fieldType, nil
	} else {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"error":     err.Error(),
		}).Debugf("Field not found in typeDef")
	}

	// Try type guard lookup
	if fieldType, err := tc.lookupFieldInTypeGuard(baseType, fieldName); err == nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"result":    "found in typeGuard",
		}).Debugf("Field found in typeGuard")
		return fieldType, nil
	} else {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"error":     err.Error(),
		}).Debugf("Field not found in typeGuard")
	}

	// Try assertion chain lookup
	if fieldType, err := tc.lookupFieldInAssertion(baseType, fieldName, parentAssertion); err == nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"result":    "found in assertion",
		}).Debugf("Field found in assertion")
		return fieldType, nil
	} else {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"error":     err.Error(),
		}).Debugf("Field not found in assertion")
	}

	// Don't throw error for Shape type
	if baseType.Ident == ast.TypeShape {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldType",
			"fieldName": fieldName,
			"baseType":  baseType.Ident,
			"result":    "returning Shape type",
		}).Debugf("Returning Shape type for field")
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldType",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
		"result":    "not found",
	}).Debugf("=== END FIELD LOOKUP DEBUG ===")

	return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
}

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

	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
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
		"shape":     fmt.Sprintf("%+v", shapeExpr.Shape),
		"fields":    fmt.Sprintf("%+v", shapeExpr.Shape.Fields),
	}).Debugf("Shape expression fields")

	field, exists := shapeExpr.Shape.Fields[string(fieldName.ID)]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function":        "lookupFieldInTypeDef",
			"fieldName":       fieldName,
			"baseType":        baseType.Ident,
			"availableFields": fmt.Sprintf("%+v", shapeExpr.Shape.Fields),
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

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldInTypeDef",
		"fieldName": fieldName,
		"baseType":  baseType.Ident,
	}).Debugf("Returning Shape type as fallback")
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
		return ast.TypeNode{}, fmt.Errorf("field %s not found in nested shape", fieldName)
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
	return ast.TypeNode{}, fmt.Errorf("field %s exists but is empty (no type, shape, or assertion)", fieldName)
}

// resolveTypeAliasChain follows type aliases until it reaches a non-alias (base) type
func (tc *TypeChecker) resolveTypeAliasChain(typeNode ast.TypeNode) ast.TypeNode {
	visited := map[ast.TypeIdent]bool{}
	current := typeNode
	for {
		def, exists := tc.Defs[current.Ident]
		if !exists {
			break
		}
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			break
		}
		switch expr := typeDef.Expr.(type) {
		case ast.TypeDefShapeExpr:
			// This is a shape, stop here
			return current
		case ast.TypeDefAssertionExpr:
			// Alias to another type (e.g. type Foo = Bar)
			if expr.Assertion != nil && expr.Assertion.BaseType != nil {
				if visited[*expr.Assertion.BaseType] {
					break // cycle
				}
				visited[*expr.Assertion.BaseType] = true
				current = ast.TypeNode{Ident: *expr.Assertion.BaseType}
				continue
			}
			// If no BaseType, treat as non-alias
			return current
		}
		break
	}
	return current
}

// lookupFieldPath recursively looks up a field path (e.g., ["input", "name"]) in a type or shape
func (tc *TypeChecker) lookupFieldPath(baseType ast.TypeNode, fieldPath []string) (ast.TypeNode, error) {
	if len(fieldPath) == 0 {
		return baseType, nil
	}
	fieldName := ast.Ident{ID: ast.Identifier(fieldPath[0])}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldPath",
		"baseType":  baseType.Ident,
		"fieldName": fieldName.ID,
		"fieldPath": fieldPath,
	}).Debugf("Looking up field path")

	// Resolve type aliases before lookup
	resolvedType := tc.resolveTypeAliasChain(baseType)

	tc.log.WithFields(logrus.Fields{
		"function":     "lookupFieldPath",
		"baseType":     baseType.Ident,
		"resolvedType": resolvedType.Ident,
		"fieldName":    fieldName.ID,
	}).Debugf("Resolved type alias")

	// Try type definition lookup
	if def, exists := tc.Defs[resolvedType.Ident]; exists {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			switch expr := typeDef.Expr.(type) {
			case ast.TypeDefShapeExpr:
				tc.log.WithFields(logrus.Fields{
					"function":  "lookupFieldPath",
					"baseType":  baseType.Ident,
					"fieldName": fieldName.ID,
					"shape":     fmt.Sprintf("%+v", expr.Shape),
				}).Debugf("Looking up field in shape expression")

				field, exists := expr.Shape.Fields[string(fieldName.ID)]
				if !exists {
					return ast.TypeNode{}, fmt.Errorf("field %s not found in shape", fieldName)
				}

				tc.log.WithFields(logrus.Fields{
					"function":     "lookupFieldPath",
					"baseType":     baseType.Ident,
					"fieldName":    fieldName.ID,
					"field":        fmt.Sprintf("%+v", field),
					"fieldType":    fmt.Sprintf("%T", field),
					"hasType":      field.Type != nil,
					"hasShape":     field.Shape != nil,
					"hasAssertion": field.Assertion != nil,
				}).Debugf("Found field in shape")

				if field.Type != nil && len(fieldPath) == 1 {
					// Resolve type aliases even for single-segment paths
					resolvedFieldType := tc.resolveTypeAliasChain(*field.Type)
					tc.log.WithFields(logrus.Fields{
						"function":          "lookupFieldPath",
						"baseType":          baseType.Ident,
						"fieldName":         fieldName.ID,
						"fieldType":         field.Type.Ident,
						"resolvedFieldType": resolvedFieldType.Ident,
					}).Debugf("Resolved field type alias")
					return resolvedFieldType, nil
				}
				if field.Shape != nil && len(fieldPath) > 1 {
					return tc.lookupFieldPathOnShape(field.Shape, fieldPath[1:])
				}
				if field.Shape != nil && len(fieldPath) == 1 {
					return ast.TypeNode{Ident: ast.TypeShape}, nil
				}
				if field.Type != nil && len(fieldPath) > 1 {
					// If the field is a type alias, resolve and continue
					return tc.lookupFieldPath(*field.Type, fieldPath[1:])
				}
				if field.Assertion != nil && len(fieldPath) == 1 {
					// Resolve assertion to get the underlying type
					if field.Assertion.BaseType != nil {
						resolvedType := tc.resolveTypeAliasChain(ast.TypeNode{Ident: *field.Assertion.BaseType})
						tc.log.WithFields(logrus.Fields{
							"function":          "lookupFieldPath",
							"baseType":          baseType.Ident,
							"fieldName":         fieldName.ID,
							"assertionBaseType": *field.Assertion.BaseType,
							"resolvedType":      resolvedType.Ident,
						}).Debugf("Resolved assertion field")
						return resolvedType, nil
					}
				}
				if field.Assertion != nil && len(fieldPath) > 1 {
					// If the field is an assertion, resolve and continue
					if field.Assertion.BaseType != nil {
						return tc.lookupFieldPath(ast.TypeNode{Ident: *field.Assertion.BaseType}, fieldPath[1:])
					}
				}
				tc.log.WithFields(logrus.Fields{
					"function":  "lookupFieldPath",
					"baseType":  baseType.Ident,
					"fieldName": fieldName.ID,
					"field":     fmt.Sprintf("%+v", field),
				}).Debugf("Field exists but is not a type or shape")
				return ast.TypeNode{}, fmt.Errorf("field %s exists but is not a type or shape", fieldName)
			}
		}
	}
	return ast.TypeNode{}, fmt.Errorf("field path %v not found in type %s", fieldPath, baseType.Ident)
}

// lookupFieldPathOnShape recursively looks up a field path in a ShapeNode
func (tc *TypeChecker) lookupFieldPathOnShape(shape *ast.ShapeNode, fieldPath []string) (ast.TypeNode, error) {
	if shape == nil || len(fieldPath) == 0 {
		return ast.TypeNode{}, fmt.Errorf("invalid shape or empty path")
	}
	fieldName := fieldPath[0]
	field, exists := shape.Fields[fieldName]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("field %s not found in shape", fieldName)
	}
	if field.Type != nil && len(fieldPath) == 1 {
		// Resolve type aliases even for single-segment paths
		return tc.resolveTypeAliasChain(*field.Type), nil
	}
	if field.Shape != nil && len(fieldPath) > 1 {
		return tc.lookupFieldPathOnShape(field.Shape, fieldPath[1:])
	}
	if field.Shape != nil && len(fieldPath) == 1 {
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}
	if field.Type != nil && len(fieldPath) > 1 {
		// If the field is a type alias, resolve and continue
		return tc.lookupFieldPath(*field.Type, fieldPath[1:])
	}
	if field.Type != nil {
		return *field.Type, nil
	}
	return ast.TypeNode{}, fmt.Errorf("field %s exists but is not a type or shape", fieldName)
}
