package typechecker

import (
	"fmt"
	"forst/internal/ast"

	"maps"

	logrus "github.com/sirupsen/logrus"
)

// isShapeAlias checks if a type is an alias of Shape
func (tc *TypeChecker) isShapeAlias(typeIdent ast.TypeIdent) bool {
	if def, exists := tc.Defs[typeIdent]; exists {
		tc.log.WithFields(logrus.Fields{
			"function":  "isShapeAlias",
			"typeIdent": typeIdent,
		}).Tracef("Found type definition")
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			tc.log.WithFields(logrus.Fields{
				"function":  "isShapeAlias",
				"typeIdent": typeIdent,
			}).Tracef("Type definition is TypeDefNode")
			// Check if it's directly defined as Shape
			if typeDefExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil && *typeDefExpr.Assertion.BaseType == ast.TypeShape {
					tc.log.WithFields(logrus.Fields{
						"function":  "isShapeAlias",
						"typeIdent": typeIdent,
					}).Tracef("Type is a Shape alias (via TypeDefAssertionExpr)")
					return true
				}
			} else if typeDefExpr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil && *typeDefExpr.Assertion.BaseType == ast.TypeShape {
					tc.log.WithFields(logrus.Fields{
						"function":  "isShapeAlias",
						"typeIdent": typeIdent,
					}).Tracef("Type is a Shape alias (via *TypeDefAssertionExpr)")
				}
				tc.log.WithFields(logrus.Fields{
					"function":  "isShapeAlias",
					"typeIdent": typeIdent,
				}).Tracef("Type is a Shape alias (via *TypeDefAssertionExpr) but assertion is nil")
				return true
			} else if _, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				// Direct shape definition
				tc.log.WithFields(logrus.Fields{
					"function":  "isShapeAlias",
					"typeIdent": typeIdent,
				}).Tracef("Type is a Shape alias (via TypeDefShapeExpr)")
				return true
			} else {
				tc.log.WithFields(logrus.Fields{
					"function":  "isShapeAlias",
					"typeIdent": typeIdent,
					"expr":      fmt.Sprintf("%T", typeDef.Expr),
				}).Tracef("Type definition expression is not a Shape type")
			}
		} else {
			tc.log.WithFields(logrus.Fields{
				"function":  "isShapeAlias",
				"typeIdent": typeIdent,
				"def":       fmt.Sprintf("%T", def),
			}).Tracef("Type definition is not TypeDefNode")
		}
	} else {
		tc.log.WithFields(logrus.Fields{
			"function":  "isShapeAlias",
			"typeIdent": typeIdent,
		}).Tracef("No type definition found")
	}
	return false
}

// getShapeFields retrieves shape fields from various sources
func (tc *TypeChecker) getShapeFields(underlyingType ast.TypeNode, leftmostVar ast.Node) (map[string]ast.ShapeFieldNode, error) {
	leftShapeFields := make(map[string]ast.ShapeFieldNode)

	// Check if the left-hand type is a type alias of Shape
	if tc.isShapeAlias(underlyingType.Ident) {
		tc.log.WithFields(logrus.Fields{
			"function": "getShapeFields",
			"type":     underlyingType.Ident,
		}).Tracef("Type is a Shape alias")

		// For Shape/shape aliases, get the fields from the type definition
		if def, exists := tc.Defs[underlyingType.Ident]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if typeDefExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					if typeDefExpr.Assertion != nil {
						// Collect all fields from constraints
						mergedFields := MergeShapeFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
						tc.log.WithFields(logrus.Fields{
							"function": "getShapeFields",
							"type":     underlyingType.Ident,
							"fields":   mergedFields,
						}).Tracef("Collected fields for type")
						maps.Copy(leftShapeFields, mergedFields)
					}
				} else if typeDefExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					// Direct shape definition
					maps.Copy(leftShapeFields, typeDefExpr.Shape.Fields)
				}
			}
		}
	}

	// If we still have no fields, try to get them from VariableTypes
	if len(leftShapeFields) == 0 {
		if varNode, ok := leftmostVar.(ast.VariableNode); ok {
			if varTypes, exists := tc.VariableTypes[varNode.Ident.ID]; exists {
				for _, varType := range varTypes {
					if varType.Ident == ast.TypeShape && varType.Assertion != nil {
						for _, constraint := range varType.Assertion.Constraints {
							if constraint.Name == "Match" && len(constraint.Args) > 0 {
								if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
									maps.Copy(leftShapeFields, shapeArg.Fields)
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// If we still have no fields, try to get them from the left type's assertion
	if len(leftShapeFields) == 0 && underlyingType.Assertion != nil {
		for _, constraint := range underlyingType.Assertion.Constraints {
			if constraint.Name == "Match" && len(constraint.Args) > 0 {
				if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
					maps.Copy(leftShapeFields, shapeArg.Fields)
					break
				}
			}
		}
	}

	if len(leftShapeFields) == 0 {
		return nil, fmt.Errorf("no shape fields found for type %s", underlyingType.Ident)
	}

	return leftShapeFields, nil
}

// validateShapeFields validates that the right-hand shape fields are compatible with the left-hand shape fields
func (tc *TypeChecker) validateShapeFields(shapeNode ast.ShapeNode, leftShapeFields map[string]ast.ShapeFieldNode, underlyingType ast.TypeIdent) error {
	for fieldName, rightField := range shapeNode.Fields {
		leftField, exists := leftShapeFields[fieldName]
		if !exists {
			return fmt.Errorf("field %s not found in shape type %s", fieldName, underlyingType)
		}
		// Compare field types
		if rightField.Assertion != nil && leftField.Assertion != nil {
			if rightField.Assertion.BaseType != nil && leftField.Assertion.BaseType != nil {
				rightType := ast.TypeNode{Ident: *rightField.Assertion.BaseType}
				leftType := ast.TypeNode{Ident: *leftField.Assertion.BaseType}
				if !tc.IsTypeCompatible(leftType, rightType) {
					return fmt.Errorf("field %s type mismatch: %s vs %s",
						fieldName, leftType.Ident, rightType.Ident)
				}
			}
		}
	}
	return nil
}
