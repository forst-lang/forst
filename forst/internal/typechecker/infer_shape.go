package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// TODO: Improve type inference for complex types
// This should handle:
// 1. Binary type expressions
// 2. Nested shapes
// 3. Type aliases
// 4. Generic types
// Change function signature to accept expectedType
func (tc *TypeChecker) inferShapeType(shape ast.ShapeNode, expectedType *ast.TypeNode) (ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":     "inferShapeType",
		"shape":        fmt.Sprintf("%+v", shape),
		"expectedType": expectedType,
	}).Debug("Inferring shape type")

	// If the shape has a BaseType, use it directly
	if shape.BaseType != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "inferShapeType",
			"baseType": *shape.BaseType,
		}).Debug("Using BaseType for shape literal")
		return ast.TypeNode{Ident: *shape.BaseType}, nil
	}

	// Try to find a matching type definition for this shape
	shapeHash, err := tc.Hasher.HashNode(shape)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("failed to hash shape: %w", err)
	}
	shapeTypeIdent := shapeHash.ToTypeIdent()

	// Check if we already have a type definition for this shape
	if _, exists := tc.Defs[shapeTypeIdent]; exists {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferShapeType",
			"typeIdent": shapeTypeIdent,
		}).Debug("Found existing type definition for shape")
		return ast.TypeNode{Ident: shapeTypeIdent}, nil
	}

	// If expectedType is a named type, try to get its definition for field types
	var expectedFields map[string]*ast.TypeNode
	if expectedType != nil {
		if def, ok := tc.Defs[expectedType.Ident]; ok {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					expectedFields = make(map[string]*ast.TypeNode)
					for fname, fdef := range shapeExpr.Shape.Fields {
						if fdef.Type != nil {
							t := *fdef.Type
							expectedFields[fname] = &t
						}
					}
				}
			}
		}
	}

	// Process each field and infer its type
	processedFields := make(map[string]ast.ShapeFieldNode)

	for name, field := range shape.Fields {
		// Use expectedType for this field if available
		var fieldExpectedType *ast.TypeNode
		if expectedFields != nil {
			if t, ok := expectedFields[name]; ok {
				fieldExpectedType = t
			}
		}
		tc.log.WithFields(logrus.Fields{
			"function":     "inferShapeType",
			"fieldName":    name,
			"expectedType": fieldExpectedType,
		}).Warn("[PINPOINT] Expected type for field (from expectedType param)")

		// If the field has an assertion, infer its type
		if field.Assertion != nil {
			assertionTypes, err := tc.InferAssertionType(field.Assertion, false, name, fieldExpectedType)
			if err != nil {
				return ast.TypeNode{}, fmt.Errorf("failed to infer assertion type for field %s: %w", name, err)
			}
			if len(assertionTypes) == 0 {
				return ast.TypeNode{}, fmt.Errorf("no types inferred for assertion in field %s", name)
			}
			inferredType := assertionTypes[0]
			if !tc.isBuiltinType(inferredType.Ident) {
				tc.registerType(ast.TypeDefNode{
					Ident: inferredType.Ident,
					Expr: ast.TypeDefAssertionExpr{
						Assertion: field.Assertion,
					},
				})
			}
			tc.log.WithFields(logrus.Fields{
				"function":     "inferShapeType",
				"fieldName":    name,
				"inferredType": inferredType.Ident,
			}).Warn("[PINPOINT] Inferred type for field (from assertion)")
			field.Type = &inferredType
		}

		// If the field has a nested shape, infer its type recursively
		if field.Shape != nil {
			nestedType, err := tc.inferShapeType(*field.Shape, fieldExpectedType)
			if err != nil {
				return ast.TypeNode{}, fmt.Errorf("failed to infer nested shape type for field %s: %w", name, err)
			}
			field.Type = &nestedType
		}

		processedFields[name] = field
	}

	// Create a new shape with processed fields
	processedShape := ast.ShapeNode{
		Fields:   processedFields,
		BaseType: shape.BaseType,
	}

	// Special case: if the shape has only one field and that field is a pointer type from Value(nil), return the pointer type directly
	if len(processedShape.Fields) == 1 {
		for _, field := range processedShape.Fields {
			if field.Type != nil && field.Type.Ident == ast.TypePointer {
				tc.log.WithFields(logrus.Fields{
					"function":  "inferShapeType",
					"fieldType": field.Type.Ident,
				}).Debug("Single-field shape with pointer type, returning pointer type directly")
				return *field.Type, nil
			}
		}
	}

	// Now look for a matching named type definition that has the same structure
	var matchingTypeIdent ast.TypeIdent

	for typeIdent, def := range tc.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				if tc.shapesHaveSameStructure(processedShape, shapeExpr.Shape) {
					matchingTypeIdent = typeIdent
					tc.log.WithFields(logrus.Fields{
						"function":     "inferShapeType",
						"matchingType": typeIdent,
					}).Debug("Found matching named type definition")
					break
				}
			}
		}
	}

	if matchingTypeIdent != "" {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferShapeType",
			"namedType": matchingTypeIdent,
		}).Debug("Using matching named type directly")
		return ast.TypeNode{Ident: matchingTypeIdent}, nil
	}

	finalHash, err := tc.Hasher.HashNode(processedShape)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("failed to hash processed shape: %w", err)
	}
	finalTypeIdent := finalHash.ToTypeIdent()
	tc.registerType(ast.TypeDefNode{
		Ident: finalTypeIdent,
		Expr:  ast.TypeDefShapeExpr{Shape: processedShape},
	})

	return ast.TypeNode{Ident: finalTypeIdent}, nil
}

// isBuiltinType checks if a type identifier is a built-in type
func (tc *TypeChecker) isBuiltinType(typeIdent ast.TypeIdent) bool {
	builtinTypes := []ast.TypeIdent{
		ast.TypeString,
		ast.TypeInt,
		ast.TypeFloat,
		ast.TypeBool,
		ast.TypeError,
		ast.TypeVoid,
		ast.TypePointer,
		ast.TypeShape,
	}

	for _, builtinType := range builtinTypes {
		if typeIdent == builtinType {
			return true
		}
	}
	return false
}

// shapesHaveSameStructure compares two shapes to see if they have the same field structure
func (tc *TypeChecker) shapesHaveSameStructure(shape1, shape2 ast.ShapeNode) bool {
	tc.log.Debug("shapesHaveSameStructure called")

	tc.log.WithFields(logrus.Fields{
		"function": "shapesHaveSameStructure",
		"shape1":   fmt.Sprintf("%+v", shape1),
		"shape2":   fmt.Sprintf("%+v", shape2),
	}).Debug("Comparing shapes for structural compatibility")

	if len(shape1.Fields) != len(shape2.Fields) {
		tc.log.WithFields(logrus.Fields{
			"function": "shapesHaveSameStructure",
			"reason":   "field count mismatch",
			"fields1":  len(shape1.Fields),
			"fields2":  len(shape2.Fields),
		}).Debug("Shape field count mismatch")
		return false
	}

	for fieldName, field1 := range shape1.Fields {
		field2, exists := shape2.Fields[fieldName]
		if !exists {
			tc.log.WithFields(logrus.Fields{
				"function":  "shapesHaveSameStructure",
				"reason":    "field missing",
				"fieldName": fieldName,
			}).Debug("Field missing in compared shape")
			return false
		}

		// Get the resolved type for field1 (handling assertions)
		var field1Type *ast.TypeNode
		if field1.Type != nil {
			field1Type = field1.Type
		} else if field1.Assertion != nil {
			assertionTypes, err := tc.InferAssertionType(field1.Assertion, false, fieldName, nil)
			if err == nil && len(assertionTypes) > 0 {
				field1Type = &assertionTypes[0]
			}
		}

		// Get the resolved type for field2 (handling assertions)
		var field2Type *ast.TypeNode
		if field2.Type != nil {
			field2Type = field2.Type
		} else if field2.Assertion != nil {
			assertionTypes, err := tc.InferAssertionType(field2.Assertion, false, fieldName, nil)
			if err == nil && len(assertionTypes) > 0 {
				field2Type = &assertionTypes[0]
			}
		}

		tc.log.WithFields(logrus.Fields{
			"function":   "shapesHaveSameStructure",
			"fieldName":  fieldName,
			"field1Type": field1Type,
			"field2Type": field2Type,
			"field1":     field1,
			"field2":     field2,
		}).Debug("Comparing field types")

		if field1Type != nil && field2Type != nil {
			if field1Type.Ident != field2Type.Ident {
				tc.log.WithFields(logrus.Fields{
					"function":   "shapesHaveSameStructure",
					"fieldName":  fieldName,
					"field1Type": field1Type.Ident,
					"field2Type": field2Type.Ident,
				}).Debug("Field type mismatch")
				return false
			}
		} else if field1Type != nil || field2Type != nil {
			tc.log.WithFields(logrus.Fields{
				"function":   "shapesHaveSameStructure",
				"fieldName":  fieldName,
				"field1Type": field1Type,
				"field2Type": field2Type,
			}).Debug("One field has type, the other does not")
			return false
		}

		if field1.Shape != nil && field2.Shape != nil {
			if !tc.shapesHaveSameStructure(*field1.Shape, *field2.Shape) {
				tc.log.WithFields(logrus.Fields{
					"function":  "shapesHaveSameStructure",
					"fieldName": fieldName,
				}).Debug("Nested shape mismatch")
				return false
			}
		}
	}

	return true
}
