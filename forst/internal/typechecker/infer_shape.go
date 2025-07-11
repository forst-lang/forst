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
func (tc *TypeChecker) inferShapeType(shape ast.ShapeNode) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function": "inferShapeType",
		"shape":    fmt.Sprintf("%+v", shape),
	}).Debug("Starting shape type inference")
	hash, err := tc.Hasher.HashNode(shape)
	if err != nil {
		return nil, fmt.Errorf("failed to hash shape: %w", err)
	}
	typeIdent := hash.ToTypeIdent()
	shapeType := []ast.TypeNode{
		{
			Ident: typeIdent,
		},
	}

	// First, register the shape type itself
	tc.registerType(ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefShapeExpr{
			Shape: shape,
		},
	})

	for name, field := range shape.Fields {
		if field.Shape != nil {
			// First infer the nested shape type
			fieldType, err := tc.inferShapeType(*field.Shape)
			if err != nil {
				return nil, err
			}

			// Register the nested shape type
			fieldHash, err := tc.Hasher.HashNode(field.Shape)
			if err != nil {
				return nil, fmt.Errorf("failed to hash field shape: %w", err)
			}
			fieldTypeIdent := fieldHash.ToTypeIdent()
			tc.log.Tracef("Inferred type of shape field %s: %s, field: %s", name, fieldTypeIdent, field)
			tc.storeInferredType(field.Shape, fieldType)

			// If the field has an assertion, we need to merge it with the shape type
			if field.Assertion != nil {
				mergedFields := tc.resolveShapeFieldsFromAssertion(field.Assertion)
				for k, v := range mergedFields {
					field.Shape.Fields[k] = v
				}
			}

			// Register the field type in the parent shape
			tc.registerType(ast.TypeDefNode{
				Ident: fieldTypeIdent,
				Expr: ast.TypeDefShapeExpr{
					Shape: *field.Shape,
				},
			})
		} else if field.Assertion != nil {
			// Eagerly resolve Value(Variable(...)) constraints
			if len(field.Assertion.Constraints) > 0 && field.Assertion.Constraints[0].Name == ast.ValueConstraint {
				resolvedType, err := tc.inferValueConstraintType(field.Assertion.Constraints[0], name)
				if err == nil {
					// Store the resolved type in the field
					f := field // copy to avoid mutating the original in the map
					f.Type = &resolvedType
					shape.Fields[name] = f
					continue
				}
			}
			// Skip if the assertion type has already been inferred
			inferredType, _ := tc.InferAssertionType(field.Assertion, false)
			if inferredType != nil {
				continue
			}

			fieldHash, err := tc.Hasher.HashNode(field)
			if err != nil {
				return nil, fmt.Errorf("failed to hash field: %w", err)
			}
			fieldTypeIdent := fieldHash.ToTypeIdent()
			tc.log.Tracef("Inferred type of assertion field %s: %s", name, fieldTypeIdent)
			tc.registerType(ast.TypeDefNode{
				Ident: fieldTypeIdent,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: field.Assertion,
				},
			})
		} else if field.Type != nil {
			// Register the field's explicit type if needed (for user-defined types)
			// For built-in types, this is a no-op
			// Optionally, could register a typedef for user types here if not present
			continue
		} else {
			return nil, fmt.Errorf("shape field has neither assertion, shape, nor type: %T", field)
		}
	}

	tc.storeInferredType(shape, shapeType)
	tc.log.Tracef("Inferred shape type: %s", shapeType)

	tc.log.WithFields(logrus.Fields{
		"function": "inferShapeType",
		"shape":    fmt.Sprintf("%+v", shape),
		"type":     shapeType,
	}).Debug("Completed shape type inference")

	return shapeType, nil
}
