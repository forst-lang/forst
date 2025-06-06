package typechecker

import (
	"fmt"
	"forst/internal/ast"

	"maps"

	log "github.com/sirupsen/logrus"
)

// isShapeAlias checks if a type is an alias of Shape
func (tc *TypeChecker) isShapeAlias(typeIdent ast.TypeIdent) bool {
	if def, exists := tc.Defs[typeIdent]; exists {
		log.Tracef("[isShapeAlias] Found type definition for %s", typeIdent)
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			log.Tracef("[isShapeAlias] Type definition is TypeDefNode")
			// Check if it's directly defined as Shape
			if typeDefExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil && *typeDefExpr.Assertion.BaseType == ast.TypeShape {
					log.Tracef("[isShapeAlias] Type %s is a Shape alias (via TypeDefAssertionExpr)", typeIdent)
					return true
				}
			} else if typeDefExpr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil && *typeDefExpr.Assertion.BaseType == ast.TypeShape {
					log.Tracef("[isShapeAlias] Type %s is a Shape alias (via *TypeDefAssertionExpr)", typeIdent)
				}
				log.Tracef("[isShapeAlias] Type %s is a Shape alias (via *TypeDefAssertionExpr) but assertion is nil", typeIdent)
				return true
			} else if _, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				// Direct shape definition
				log.Tracef("[isShapeAlias] Type %s is a Shape alias (via TypeDefShapeExpr)", typeIdent)
				return true
			} else {
				log.Tracef("[isShapeAlias] Type definition expression is not a Shape type: %T", typeDef.Expr)
			}
		} else {
			log.Tracef("[isShapeAlias] Type definition is not TypeDefNode: %T", def)
		}
	} else {
		log.Tracef("[isShapeAlias] No type definition found for %s", typeIdent)
	}
	return false
}

// processTypeGuardFields processes type guard constraints and adds their fields to the shape
func (tc *TypeChecker) processTypeGuardFields(shapeNode *ast.ShapeNode, assertionNode *ast.AssertionNode) {
	if assertionNode == nil {
		return
	}
	log.Tracef("[processTypeGuardFields] Processing type guard application with %d constraints", len(assertionNode.Constraints))
	// Add fields from type guards to the right-hand shape
	for _, constraint := range assertionNode.Constraints {
		log.Tracef("[processTypeGuardFields] Processing type guard constraint '%s'", constraint.Name)
		// Look up the type guard definition
		if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
			log.Tracef("[processTypeGuardFields] Found type guard definition for '%s'", constraint.Name)
			if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
				log.Tracef("[processTypeGuardFields] Type guard has %d additional params", len(guardNode.Params))
				// Get the parameter name and type from the type guard
				if len(guardNode.Params) > 0 && len(constraint.Args) > 0 {
					param := guardNode.Params[0]
					paramName := param.GetIdent()
					// Use the actual argument type from the constraint application
					argType := constraint.Args[0]
					// Add the new field to the right-hand shape
					if argType.Type != nil {
						shapeNode.Fields[paramName] = ast.ShapeFieldNode{
							Assertion: &ast.AssertionNode{
								BaseType: &argType.Type.Ident,
							},
						}
						log.Tracef("[processTypeGuardFields] Added field '%s' of type %s from type guard %s to result shape",
							paramName, argType.Type.Ident, constraint.Name)
					} else {
						log.Errorf("[processTypeGuardFields] Constraint argument for field '%s' in type guard %s has no Type",
							paramName, constraint.Name)
					}
				} else {
					log.Tracef("[processTypeGuardFields] Type guard '%s' has insufficient params/args: %d params, %d args",
						constraint.Name, len(guardNode.Params), len(constraint.Args))
				}
			} else {
				log.Tracef("[processTypeGuardFields] Definition for '%s' is not a TypeGuardNode: %T", constraint.Name, guardDef)
			}
		} else {
			log.Tracef("[processTypeGuardFields] No definition found for type guard '%s'", constraint.Name)
		}
	}
}

// getLeftmostVariable returns the leftmost variable in an expression
func (tc *TypeChecker) getLeftmostVariable(node ast.Node) (ast.Node, error) {
	switch n := node.(type) {
	case ast.VariableNode:
		return n, nil
	case ast.BinaryExpressionNode:
		return tc.getLeftmostVariable(n.Left)
	default:
		return nil, fmt.Errorf("expression does not start with a variable: %T", node)
	}
}

// unifyIsOperator handles type unification for the 'is' operator
func (tc *TypeChecker) unifyIsOperator(left ast.Node, right ast.Node, leftType ast.TypeNode, rightType ast.TypeNode) (ast.TypeNode, error) {
	// Get the leftmost variable to check against type guard receiver
	leftmostVar, err := tc.getLeftmostVariable(left)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("invalid left-hand side of 'is' operator: %v", err)
	}

	// Get inferred types for the leftmost variable
	varLeftTypes, err := tc.inferExpressionType(leftmostVar)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("failed to infer type of leftmost variable: %v", err)
	}
	if len(varLeftTypes) == 0 {
		return ast.TypeNode{}, fmt.Errorf("leftmost variable in IS expression %s has an empty type", leftmostVar.(ast.VariableNode).Ident.ID)
	}
	varLeftType := varLeftTypes[0]

	// Handle TypeDefAssertionExpr
	if typeDefAssertion, ok := right.(ast.TypeDefAssertionExpr); ok {
		assertionNode := typeDefAssertion.Assertion
		if assertionNode == nil {
			return ast.TypeNode{}, fmt.Errorf("right-hand side of 'is' must be an assertion")
		}

		// Check that the assertion's base type matches the left-hand side type or is a subtype
		if assertionNode.BaseType != nil {
			baseType := ast.TypeNode{Ident: *assertionNode.BaseType}
			if !tc.isTypeCompatible(varLeftType, baseType) {
				return ast.TypeNode{}, fmt.Errorf("assertion base type %s is not compatible with left-hand side type %s", baseType.Ident, varLeftType.Ident)
			}
		}

		// Process type guard constraints
		for _, constraint := range assertionNode.Constraints {
			if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
				if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
					// Check that the leftmost variable's type matches the guard's subject type
					subjectType := guardNode.Subject.GetType()
					if !tc.isTypeCompatible(varLeftType, subjectType) {
						return ast.TypeNode{}, fmt.Errorf("type guard '%s' requires subject type %s, but got %s",
							constraint.Name, subjectType.Ident, varLeftType.Ident)
					}
				}
			}
		}
		// Process type guard fields
		tc.processTypeGuardFields(&ast.ShapeNode{}, assertionNode)
	} else if shapeNode, ok := right.(ast.ShapeNode); ok {
		// For ShapeNode, check that each field's assertion base type is compatible with the corresponding field in the left-hand shape
		underlyingType := varLeftType
		leftShapeFields := make(map[string]ast.ShapeFieldNode)

		log.WithFields(log.Fields{
			"leftType":    varLeftType.Ident,
			"rightType":   "Shape",
			"shapeFields": fmt.Sprintf("%v", shapeNode.Fields),
		}).Debug("Starting shape comparison")

		// Check if the left-hand type is a type alias of Shape
		if tc.isShapeAlias(underlyingType.Ident) {
			log.Tracef("[unifyIsOperator] Type %s is a Shape alias", underlyingType.Ident)

			// For Shape/shape aliases, skip field validation but store fields for future use
			if def, exists := tc.Defs[underlyingType.Ident]; exists {
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					if typeDefExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
						if typeDefExpr.Assertion != nil {
							// Collect all fields from constraints
							mergedFields := MergeShapeFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
							log.Tracef("[unifyIsOperator] Collected fields for type %s: %v", underlyingType.Ident, mergedFields)

							// Store the merged fields in VariableTypes for the leftmost variable
							if leftmostVar, ok := leftmostVar.(ast.VariableNode); ok {
								// Get existing types or initialize new slice
								existingTypes := tc.VariableTypes[leftmostVar.Ident.ID]
								if existingTypes == nil {
									existingTypes = []ast.TypeNode{}
								}

								// Create a new shape type with the merged fields
								baseType := ast.TypeShape
								shapeType := ast.TypeNode{
									Ident: ast.TypeShape,
									Assertion: &ast.AssertionNode{
										BaseType: &baseType,
										Constraints: []ast.ConstraintNode{{
											Name: "Match",
											Args: []ast.ConstraintArgumentNode{{
												Shape: &ast.ShapeNode{Fields: mergedFields},
											}},
										}},
									},
								}

								// Add the shape type to the variable's types
								tc.VariableTypes[leftmostVar.Ident.ID] = append(existingTypes, shapeType)
								log.Tracef("[unifyIsOperator] Stored shape type for variable %s: %v", leftmostVar.Ident.ID, shapeType)
							}
						}
					}
				}
			}
			return ast.TypeNode{Ident: ast.TypeBool}, nil
		}

		// For non-Shape types, collect and validate fields
		leftShapeFields = make(map[string]ast.ShapeFieldNode)

		// First try to get the shape type from VariableTypes
		if leftmostVar, ok := leftmostVar.(ast.VariableNode); ok {
			if varTypes, exists := tc.VariableTypes[leftmostVar.Ident.ID]; exists {
				log.Tracef("[unifyIsOperator] Found %d types for variable %s", len(varTypes), leftmostVar.Ident.ID)
				// Look for a shape type in the variable's types
				for _, varType := range varTypes {
					if varType.Ident == ast.TypeShape && varType.Assertion != nil {
						// Extract fields from the shape type's constraints
						for _, constraint := range varType.Assertion.Constraints {
							if constraint.Name == "Match" && len(constraint.Args) > 0 {
								if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
									log.Tracef("[unifyIsOperator] Found shape fields in variable type: %v", shapeArg.Fields)
									maps.Copy(leftShapeFields, shapeArg.Fields)
									break
								}
							}
						}
					}
				}
			} else {
				log.Tracef("[unifyIsOperator] No types found for variable %s", leftmostVar.Ident.ID)
				// return ast.TypeNode{}, fmt.Errorf("no type information found for variable %s", leftmostVar.Ident.ID)
			}
		}

		// If we didn't find shape fields in VariableTypes, try to collect from constraints
		if len(leftShapeFields) == 0 {
			// Helper to collect fields from constraints
			collectFieldsFromConstraints := func(constraints []ast.ConstraintNode) {
				log.Tracef("[collectFieldsFromConstraints] Processing %d constraints", len(constraints))
				mergedFields := MergeShapeFieldsFromConstraints(constraints)

				// Store the merged fields in VariableTypes for the leftmost variable
				if leftmostVar, ok := leftmostVar.(ast.VariableNode); ok {
					// Get existing types or initialize new slice
					existingTypes := tc.VariableTypes[leftmostVar.Ident.ID]
					if existingTypes == nil {
						existingTypes = []ast.TypeNode{}
					}

					// Create a new shape type with the merged fields
					baseType := ast.TypeShape
					shapeType := ast.TypeNode{
						Ident: ast.TypeShape,
						Assertion: &ast.AssertionNode{
							BaseType: &baseType,
							Constraints: []ast.ConstraintNode{{
								Name: "Match",
								Args: []ast.ConstraintArgumentNode{{
									Shape: &ast.ShapeNode{Fields: mergedFields},
								}},
							}},
						},
					}

					// Add the shape type to the variable's types
					tc.VariableTypes[leftmostVar.Ident.ID] = append(existingTypes, shapeType)
					log.Tracef("[unifyIsOperator] Stored shape type for variable %s: %v", leftmostVar.Ident.ID, shapeType)
				}

				maps.Copy(leftShapeFields, mergedFields)
			}

			// Look up the type definition to resolve aliases and collect constraint fields
			if def, exists := tc.Defs[underlyingType.Ident]; exists {
				log.Tracef("[unifyIsOperator] Found type definition for %s", underlyingType.Ident)
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					log.Tracef("[unifyIsOperator] Type definition is TypeDefNode")
					if typeDefExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
						log.Tracef("[unifyIsOperator] Type definition expression is TypeDefAssertionExpr")
						if typeDefExpr.Assertion != nil {
							log.Tracef("[unifyIsOperator] Processing %d constraints from type definition", len(typeDefExpr.Assertion.Constraints))
							collectFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
						}
					}
				}
			}
		}

		// If we still have no fields, fail
		if len(leftShapeFields) == 0 {
			return ast.TypeNode{}, fmt.Errorf("no shape fields found for type %s", underlyingType.Ident)
		}

		// If we have collected fields, validate them against the right-hand shape
		if rightShape, ok := right.(ast.ShapeNode); ok {
			log.Tracef("[unifyIsOperator] Comparing shape fields structurally")
			// Check that all fields in rightShape exist in leftShapeFields with compatible types
			for fieldName, rightField := range rightShape.Fields {
				leftField, exists := leftShapeFields[fieldName]
				if !exists {
					log.Tracef("[unifyIsOperator] Field %s not found in left shape", fieldName)
					return ast.TypeNode{}, fmt.Errorf("field %s not found in shape type %s", fieldName, underlyingType.Ident)
				}
				// Compare field types
				if rightField.Assertion != nil && leftField.Assertion != nil {
					if rightField.Assertion.BaseType != nil && leftField.Assertion.BaseType != nil {
						rightType := ast.TypeNode{Ident: *rightField.Assertion.BaseType}
						leftType := ast.TypeNode{Ident: *leftField.Assertion.BaseType}
						if !tc.isTypeCompatible(leftType, rightType) {
							return ast.TypeNode{}, fmt.Errorf("field %s type mismatch: %s vs %s",
								fieldName, leftType.Ident, rightType.Ident)
						}
					}
				}
			}
		}

		return ast.TypeNode{Ident: ast.TypeBool}, nil
	} else if assertionNode, ok := right.(ast.AssertionNode); ok {
		// Handle direct assertions (like NotNil)
		// For NotNil, we need to check that the left-hand side is a pointer type
		for _, constraint := range assertionNode.Constraints {
			if constraint.Name == "NotNil" {
				// Check if left type is a pointer type
				if varLeftType.Ident != ast.TypePointer {
					return ast.TypeNode{}, fmt.Errorf("NotNil assertion requires a pointer type, got %s", varLeftType.Ident)
				}
			} else {
				// Check type guard subject type for other constraints
				if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
					if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
						subjectType := guardNode.Subject.GetType()
						if !tc.isTypeCompatible(varLeftType, subjectType) {
							return ast.TypeNode{}, fmt.Errorf("type guard '%s' requires subject type %s, but got %s",
								constraint.Name, subjectType.Ident, varLeftType.Ident)
						}
					}
				}
			}
		}
	} else if rightType.Ident != ast.TypeShape {
		return ast.TypeNode{}, fmt.Errorf("right-hand side of 'is' must be a Shape type or assertion, got %s", rightType.Ident)
	}

	return ast.TypeNode{Ident: ast.TypeBool}, nil
}

// unifyTypes attempts to unify two types based on the operator and operand types
func (tc *TypeChecker) unifyTypes(left ast.Node, right ast.Node, operator ast.TokenIdent) (ast.TypeNode, error) {
	leftTypes, err := tc.inferExpressionType(left)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(leftTypes) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type but got %d types", len(leftTypes))
	}
	leftType := leftTypes[0]

	rightTypes, err := tc.inferExpressionType(right)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(rightTypes) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type but got %d types", len(rightTypes))
	}
	rightType := rightTypes[0]

	// Check type compatibility and determine result type
	if operator.IsArithmeticBinaryOperator() {
		if leftType.Ident != rightType.Ident {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in arithmetic expression: %s and %s",
				leftType.Ident, rightType.Ident)
		}
		return leftType, nil

	} else if operator.IsComparisonBinaryOperator() {
		if leftType.Ident != rightType.Ident {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in comparison expression: %s and %s",
				leftType.Ident, rightType.Ident)
		}
		return ast.TypeNode{Ident: ast.TypeBool}, nil

	} else if operator.IsLogicalBinaryOperator() {
		if leftType.Ident != rightType.Ident {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in logical expression: %s and %s",
				leftType.Ident, rightType.Ident)
		}
		return ast.TypeNode{Ident: ast.TypeBool}, nil
	} else if operator == ast.TokenIs {
		return tc.unifyIsOperator(left, right, leftType, rightType)
	}

	panic(typecheckError("unsupported operator"))
}
