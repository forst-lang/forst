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
func (tc *TypeChecker) inferShapeType(shape ast.ShapeNode) (ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function": "inferShapeType",
		"shape":    fmt.Sprintf("%+v", shape),
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

	// Look for a matching named type definition that has the same structure
	var matchingTypeDef *ast.TypeDefNode
	var matchingTypeIdent ast.TypeIdent

	for typeIdent, def := range tc.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				// Compare the shape structure
				if tc.shapesHaveSameStructure(shape, shapeExpr.Shape) {
					matchingTypeDef = &typeDef
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

	// Process each field and infer its type
	processedFields := make(map[string]ast.ShapeFieldNode)

	for name, field := range shape.Fields {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferShapeType",
			"fieldName": name,
			"field":     fmt.Sprintf("%+v", field),
		}).Debug("Processing field")

		// Get the expected type for this field from the matching type definition
		var expectedFieldType *ast.TypeNode
		if matchingTypeDef != nil {
			if shapeExpr, ok := matchingTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
				if expectedField, exists := shapeExpr.Shape.Fields[name]; exists {
					expectedFieldType = expectedField.Type
					tc.log.WithFields(logrus.Fields{
						"function":     "inferShapeType",
						"fieldName":    name,
						"expectedType": expectedFieldType,
					}).Debug("Found expected type for field from matching type definition")
				}
			}
		}

		// If the field has an assertion, infer its type
		if field.Assertion != nil {
			// Infer assertion type with the expected field type
			assertionTypes, err := tc.InferAssertionType(field.Assertion, false, name, expectedFieldType)
			if err != nil {
				return ast.TypeNode{}, fmt.Errorf("failed to infer assertion type for field %s: %w", name, err)
			}
			if len(assertionTypes) == 0 {
				return ast.TypeNode{}, fmt.Errorf("no types inferred for assertion in field %s", name)
			}
			// Register the assertion type
			tc.registerType(ast.TypeDefNode{
				Ident: assertionTypes[0].Ident,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: field.Assertion,
				},
			})
		}

		// If the field has a nested shape, infer its type recursively
		if field.Shape != nil {
			nestedType, err := tc.inferShapeType(*field.Shape)
			if err != nil {
				return ast.TypeNode{}, fmt.Errorf("failed to infer nested shape type for field %s: %w", name, err)
			}
			// Update the field with the inferred type
			field.Type = &nestedType
		}

		processedFields[name] = field
	}

	// Create a new shape with processed fields
	processedShape := ast.ShapeNode{
		Fields:   processedFields,
		BaseType: shape.BaseType,
	}

	// Hash the processed shape to get the final type identifier
	finalHash, err := tc.Hasher.HashNode(processedShape)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("failed to hash processed shape: %w", err)
	}
	finalTypeIdent := finalHash.ToTypeIdent()

	// Register the shape type definition
	tc.registerType(ast.TypeDefNode{
		Ident: finalTypeIdent,
		Expr:  ast.TypeDefShapeExpr{Shape: processedShape},
	})

	// If we found a matching named type, use it instead of the hash-based type
	if matchingTypeIdent != "" {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferShapeType",
			"hashType":  finalTypeIdent,
			"namedType": matchingTypeIdent,
		}).Debug("Using named type instead of hash-based type")
		return ast.TypeNode{Ident: matchingTypeIdent}, nil
	}

	return ast.TypeNode{Ident: finalTypeIdent}, nil
}

// shapesHaveSameStructure compares two shapes to see if they have the same field structure
func (tc *TypeChecker) shapesHaveSameStructure(shape1, shape2 ast.ShapeNode) bool {
	if len(shape1.Fields) != len(shape2.Fields) {
		return false
	}

	for fieldName, field1 := range shape1.Fields {
		field2, exists := shape2.Fields[fieldName]
		if !exists {
			return false
		}

		// Compare field types if both have types
		if field1.Type != nil && field2.Type != nil {
			if field1.Type.Ident != field2.Type.Ident {
				return false
			}
		}

		// Compare nested shapes if both have shapes
		if field1.Shape != nil && field2.Shape != nil {
			if !tc.shapesHaveSameStructure(*field1.Shape, *field2.Shape) {
				return false
			}
		}
	}

	return true
}
