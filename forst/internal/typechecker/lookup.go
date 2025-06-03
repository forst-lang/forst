package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

// LookupInferredType looks up the inferred type of a node in the current scope
func (tc *TypeChecker) LookupInferredType(node ast.Node, requireInferred bool) ([]ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(node)
	if existingType, exists := tc.Types[hash]; exists {
		// Ignore types that are still marked as implicit, as they are not yet inferred
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
	log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, scope.Node)

	// Split the identifier on dots to handle field access
	parts := strings.Split(string(variable.Ident.ID), ".")

	// Look up the base variable
	symbol, exists := scope.LookupVariable(ast.Identifier(parts[0]))
	if !exists {
		err := fmt.Errorf("undefined symbol: %s", parts[0])
		log.WithError(err).Error("lookup symbol failed")
		return ast.TypeNode{}, err
	}
	if len(symbol.Types) != 1 {
		err := fmt.Errorf("expected single type for variable %s but got %d types", parts[0], len(symbol.Types))
		log.WithError(err).Error("lookup symbol failed")
		return ast.TypeNode{}, err
	}

	// If there are no field accesses, return the base type
	if len(parts) == 1 {
		return symbol.Types[0], nil
	}

	// Handle field access by looking up each field in the type
	currentType := symbol.Types[0]
	for i := 1; i < len(parts); i++ {
		// Look up the field in the current type
		fieldType, err := tc.lookupFieldType(currentType, ast.Ident{ID: ast.Identifier(parts[i])})
		if err != nil {
			return ast.TypeNode{}, err
		}
		currentType = fieldType
	}

	return currentType, nil
}

// resolveMergedShapeFields recursively merges all shape fields from an assertion chain
func (tc *TypeChecker) resolveMergedShapeFields(assertion *ast.AssertionNode) map[string]ast.ShapeFieldNode {
	merged := make(map[string]ast.ShapeFieldNode)
	if assertion == nil {
		return merged
	}

	log.Tracef("[resolveMergedShapeFields] Processing assertion: %v", assertion)

	// Recursively merge fields from the base type if it's an assertion
	if assertion.BaseType != nil {
		log.Tracef("[resolveMergedShapeFields] Checking base type: %s", *assertion.BaseType)
		if def, exists := tc.Defs[*assertion.BaseType]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if baseAssertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					if baseAssertionExpr.Assertion != nil {
						log.Tracef("[resolveMergedShapeFields] Recursively merging fields from base assertion: %v", baseAssertionExpr.Assertion)
						for k, v := range tc.resolveMergedShapeFields(baseAssertionExpr.Assertion) {
							merged[k] = v
						}
					}
				}
			}
		}
	}

	// Merge fields from all constraints and their arguments
	for _, constraint := range assertion.Constraints {
		log.Tracef("[resolveMergedShapeFields] Processing constraint: %s with %d args", constraint.Name, len(constraint.Args))

		// If the constraint is a type guard (e.g., Context), merge the field it adds
		if len(constraint.Args) > 0 {
			log.Tracef("[resolveMergedShapeFields] Processing type guard constraint: %s with args: %v", constraint.Name, constraint.Args)
			// Look up the type guard definition to get its return value
			if def, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
				log.Tracef("[resolveMergedShapeFields] Found type guard definition: %v", def)
				if guardNode, ok := def.(*ast.TypeGuardNode); ok {
					log.Tracef("[resolveMergedShapeFields] Type guard node: %+v", guardNode)
					// Get the parameter name from the type guard
					paramName := guardNode.SubjectParam.GetIdent()
					log.Tracef("[resolveMergedShapeFields] Type guard parameter name: %s", paramName)

					// Look for is operations in the body
					for i, node := range guardNode.Body {
						log.Tracef("[resolveMergedShapeFields] Processing body node %d: %T %+v", i, node, node)
						var binExpr ast.BinaryExpressionNode
						if retNode, ok := node.(ast.ReturnNode); ok {
							log.Tracef("[resolveMergedShapeFields] Body node is a ReturnNode, checking its Value: %T %+v", retNode.Value, retNode.Value)
							if be, ok := retNode.Value.(ast.BinaryExpressionNode); ok {
								binExpr = be
								log.Tracef("[resolveMergedShapeFields] Found binary expression in ReturnNode.Value: %+v", binExpr)
							} else {
								log.Tracef("[resolveMergedShapeFields] ReturnNode.Value is not a BinaryExpressionNode: %T %+v", retNode.Value, retNode.Value)
								continue
							}
						} else if be, ok := node.(ast.BinaryExpressionNode); ok {
							binExpr = be
							log.Tracef("[resolveMergedShapeFields] Found binary expression directly: %+v", binExpr)
						} else {
							log.Tracef("[resolveMergedShapeFields] Body node is not a ReturnNode or BinaryExpressionNode: %T %+v", node, node)
							continue
						}
						log.Tracef("[resolveMergedShapeFields] Found binary expression with operator: %v", binExpr.Operator)
						if binExpr.Operator == ast.TokenIs {
							log.Tracef("[resolveMergedShapeFields] Found is operation. Left: %+v, Right: %+v", binExpr.Left, binExpr.Right)
							if shapeNode, ok := binExpr.Right.(ast.ShapeNode); ok {
								log.Tracef("[resolveMergedShapeFields] Found shape in is operation: %+v", shapeNode)
								log.Tracef("[resolveMergedShapeFields] Type guard parameter name: %s", guardNode.SubjectParam.GetIdent())
								log.Tracef("[resolveMergedShapeFields] Processing %d fields in shape", len(shapeNode.Fields))
								for fieldName, field := range shapeNode.Fields {
									log.Tracef("[resolveMergedShapeFields] Processing field %s: %+v", fieldName, field)
									// If the field name matches the type guard's shape field, create a nested shape
									if arg := constraint.Args[0]; arg.Type != nil || arg.Shape != nil {
										log.Tracef("[resolveMergedShapeFields] Found argument: %+v", arg)
										// Recursively merge fields for the argument type
										mergedFields := map[string]ast.ShapeFieldNode{}
										if arg.Type != nil {
											if def, exists := tc.Defs[arg.Type.Ident]; exists {
												log.Tracef("[resolveMergedShapeFields] Found type definition: %T %+v", def, def)
												if typeDef, ok := def.(ast.TypeDefNode); ok {
													log.Tracef("[resolveMergedShapeFields] Type definition is TypeDefNode: %+v", typeDef)
													if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
														log.Tracef("[resolveMergedShapeFields] Type definition has shape expression: %+v", shapeExpr)
														for k, v := range shapeExpr.Shape.Fields {
															mergedFields[k] = v
															log.Tracef("[resolveMergedShapeFields] Added merged field %s: %+v", k, v)
														}
													} else if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
														log.Tracef("[resolveMergedShapeFields] Type definition has assertion expression: %+v", assertionExpr)
														for k, v := range tc.resolveMergedShapeFields(assertionExpr.Assertion) {
															mergedFields[k] = v
															log.Tracef("[resolveMergedShapeFields] Added merged field %s: %+v", k, v)
														}
													} else {
														log.Tracef("[resolveMergedShapeFields] Type definition expression is neither shape nor assertion: %T %+v", typeDef.Expr, typeDef.Expr)
													}
												} else {
													log.Tracef("[resolveMergedShapeFields] Definition is not a TypeDefNode: %T %+v", def, def)
												}
											} else {
												log.Tracef("[resolveMergedShapeFields] No type definition found for %v", arg.Type.Ident)
											}
										} else if arg.Shape != nil {
											log.Tracef("[resolveMergedShapeFields] Using shape fields directly: %+v", arg.Shape.Fields)
											for k, v := range arg.Shape.Fields {
												mergedFields[k] = v
												log.Tracef("[resolveMergedShapeFields] Added shape field %s: %+v", k, v)
											}
										}
										log.Tracef("[resolveMergedShapeFields] Creating nested shape for field %s with %d fields", fieldName, len(mergedFields))
										// Create a nested shape with the merged fields
										nestedShape := &ast.ShapeNode{Fields: mergedFields}
										// Just store the nested shape directly, no type ID
										merged[fieldName] = ast.ShapeFieldNode{
											Shape: nestedShape,
										}
										continue
									} else {
										log.Tracef("[resolveMergedShapeFields] Constraint argument has no type or shape: %+v", arg)
									}
									// Otherwise, just add the field as is
									merged[fieldName] = field
									log.Tracef("[resolveMergedShapeFields] Added field %s as is: %+v", fieldName, field)
								}
							} else {
								log.Tracef("[resolveMergedShapeFields] Right side of is operation is not a shape: %T %+v", binExpr.Right, binExpr.Right)
							}
						}
					}
				} else {
					log.Tracef("[resolveMergedShapeFields] Definition is not a TypeGuardNode: %T %+v", def, def)
				}
			} else {
				log.Tracef("[resolveMergedShapeFields] No definition found for type guard: %s", constraint.Name)
			}
		}

		// Recursively merge fields from all constraint arguments
		for _, arg := range constraint.Args {
			// If the argument is a type, look up its definition and merge its fields if it's a shape or assertion
			if arg.Type != nil {
				log.Tracef("[resolveMergedShapeFields] Checking type argument: %v", arg.Type)
				if def, exists := tc.Defs[arg.Type.Ident]; exists {
					if typeDef, ok := def.(ast.TypeDefNode); ok {
						if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
							log.Tracef("[resolveMergedShapeFields] Merging fields from shape: %v", shapeExpr.Shape)
							for k, v := range shapeExpr.Shape.Fields {
								merged[k] = v
							}
						} else if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
							if assertionExpr.Assertion != nil {
								log.Tracef("[resolveMergedShapeFields] Recursively merging fields from assertion: %v", assertionExpr.Assertion)
								for k, v := range tc.resolveMergedShapeFields(assertionExpr.Assertion) {
									merged[k] = v
								}
							}
						}
					}
				}
			}
			// If the argument is a shape, merge its fields
			if arg.Shape != nil {
				log.Tracef("[resolveMergedShapeFields] Merging fields from shape: %v", arg.Shape)
				for k, v := range arg.Shape.Fields {
					merged[k] = v
				}
			}
			// If the argument is an assertion, recursively merge its fields
			if arg.Type != nil && arg.Type.Assertion != nil {
				log.Tracef("[resolveMergedShapeFields] Recursively merging fields from assertion: %v", arg.Type.Assertion)
				for k, v := range tc.resolveMergedShapeFields(arg.Type.Assertion) {
					merged[k] = v
				}
			}
		}
	}

	log.Tracef("[resolveMergedShapeFields] Final merged fields: %v", merged)
	return merged
}

// lookupNestedField recursively looks up a field in a nested shape
func (tc *TypeChecker) lookupNestedField(shape *ast.ShapeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	if shape == nil {
		return ast.TypeNode{}, fmt.Errorf("shape is nil")
	}
	if field, exists := shape.Fields[string(fieldName.ID)]; exists {
		log.Tracef("[lookupNestedField] Found field %s: %v", fieldName.ID, field)
		if field.Assertion != nil && field.Assertion.BaseType != nil {
			return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
		}
		if field.Shape != nil {
			return tc.lookupNestedField(field.Shape, fieldName)
		}
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}
	return ast.TypeNode{}, fmt.Errorf("field %s not found in nested shape", fieldName)
}

// lookupFieldType looks up a field's type in a given type
func (tc *TypeChecker) lookupFieldType(baseType ast.TypeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	log.Tracef("[lookupFieldType] Looking up field %s in type %s", fieldName, baseType.Ident)

	// If the base type is an assertion type, recursively merge all fields from the assertion chain
	if baseType.Assertion != nil {
		mergedFields := tc.resolveMergedShapeFields(baseType.Assertion)
		log.Tracef("[lookupFieldType] Recursively merged fields from assertion chain: %v", mergedFields)
		if field, exists := mergedFields[string(fieldName.ID)]; exists {
			log.Tracef("[lookupFieldType] Found field %s in merged fields: %v", fieldName.ID, field)
			if field.Assertion != nil && field.Assertion.BaseType != nil {
				return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
			}
			if field.Shape != nil {
				log.Tracef("[lookupFieldType] Field has nested shape: %v", field.Shape)
				return tc.lookupNestedField(field.Shape, fieldName)
			}
			return ast.TypeNode{Ident: ast.TypeShape}, nil
		}
	}

	// If the base type is an assertion type, first try to resolve its base type to a shape
	if baseType.Assertion != nil && baseType.Assertion.BaseType != nil {
		log.Tracef("[lookupFieldType] Checking assertion base type: %s", *baseType.Assertion.BaseType)
		// Look for the base type in VariableTypes
		if varTypes, exists := tc.VariableTypes[ast.Identifier(*baseType.Assertion.BaseType)]; exists {
			log.Tracef("[lookupFieldType] Found %d types for base type %s: %v", len(varTypes), *baseType.Assertion.BaseType, varTypes)
			// Look for a shape type in the variable's types
			for _, varType := range varTypes {
				log.Tracef("[lookupFieldType] Checking variable type: %v", varType)
				if varType.Ident == ast.TypeShape && varType.Assertion != nil {
					log.Tracef("[lookupFieldType] Found shape type with assertion: %v", varType.Assertion)
					// Extract fields from the shape type's constraints
					for _, constraint := range varType.Assertion.Constraints {
						log.Tracef("[lookupFieldType] Checking constraint: %s with %d args", constraint.Name, len(constraint.Args))
						if constraint.Name == "Match" && len(constraint.Args) > 0 {
							if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
								log.Tracef("[lookupFieldType] Found shape fields in variable type: %v", shapeArg.Fields)
								if field, exists := shapeArg.Fields[string(fieldName.ID)]; exists {
									log.Tracef("[lookupFieldType] Found field %s with type %v", fieldName.ID, field)
									if field.Assertion != nil && field.Assertion.BaseType != nil {
										return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
									}
								} else {
									log.Tracef("[lookupFieldType] Field %s not found in shape fields", fieldName.ID)
								}
							}
						}
					}
				}
			}
		}

		// If we didn't find it in VariableTypes, try to resolve the base type's definition
		if def, exists := tc.Defs[*baseType.Assertion.BaseType]; exists {
			log.Tracef("[lookupFieldType] Found type definition for base type %s: %v", *baseType.Assertion.BaseType, def)
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				log.Tracef("[lookupFieldType] Type definition is TypeDefNode: %v", typeDef)
				// Try both value and pointer types for TypeDefAssertionExpr
				var typeDefExpr ast.TypeDefAssertionExpr
				if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					typeDefExpr = expr
				} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
					typeDefExpr = *expr
				} else {
					log.Tracef("[lookupFieldType] Type definition expression is not TypeDefAssertionExpr: %T", typeDef.Expr)
					return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
				}

				// Now look for the assertion in the base type's definition
				if typeDefExpr.Assertion != nil {
					log.Tracef("[lookupFieldType] Processing %d constraints from type definition: %v",
						len(typeDefExpr.Assertion.Constraints), typeDefExpr.Assertion.Constraints)
					// Collect fields from constraints
					mergedFields := MergeShapeFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
					log.Tracef("[lookupFieldType] Merged fields from constraints: %v", mergedFields)
					if field, exists := mergedFields[string(fieldName.ID)]; exists {
						log.Tracef("[lookupFieldType] Found field %s in merged fields: %v", fieldName.ID, field)
						if field.Assertion != nil && field.Assertion.BaseType != nil {
							return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
						}
					} else {
						log.Tracef("[lookupFieldType] Field %s not found in merged fields", fieldName.ID)
					}
				}
			}
		}
	}

	// If we didn't find the field in the base type's shape, try looking in the assertion's shape
	if baseType.Assertion != nil {
		log.Tracef("[lookupFieldType] Checking assertion type: %v", baseType.Assertion)
		// Look for a Match constraint with a shape
		for _, constraint := range baseType.Assertion.Constraints {
			if constraint.Name == "Match" && len(constraint.Args) > 0 {
				if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
					log.Tracef("[lookupFieldType] Found shape in Match constraint: %v", shapeArg)
					if field, exists := shapeArg.Fields[string(fieldName.ID)]; exists {
						log.Tracef("[lookupFieldType] Found field %s in assertion shape: %v", fieldName.ID, field)
						if field.Assertion != nil && field.Assertion.BaseType != nil {
							return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
						}
					}
				}
			}
		}
	}

	// First try to get the shape type from VariableTypes
	if varTypes, exists := tc.VariableTypes[fieldName.ID]; exists {
		log.Tracef("[lookupFieldType] Found %d types for variable %s: %v", len(varTypes), fieldName.ID, varTypes)
		// Look for a shape type in the variable's types
		for _, varType := range varTypes {
			log.Tracef("[lookupFieldType] Checking variable type: %v", varType)
			if varType.Ident == ast.TypeShape && varType.Assertion != nil {
				log.Tracef("[lookupFieldType] Found shape type with assertion: %v", varType.Assertion)
				// Extract fields from the shape type's constraints
				for _, constraint := range varType.Assertion.Constraints {
					log.Tracef("[lookupFieldType] Checking constraint: %s with %d args", constraint.Name, len(constraint.Args))
					if constraint.Name == "Match" && len(constraint.Args) > 0 {
						if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
							log.Tracef("[lookupFieldType] Found shape fields in variable type: %v", shapeArg.Fields)
							if field, exists := shapeArg.Fields[string(fieldName.ID)]; exists {
								log.Tracef("[lookupFieldType] Found field %s with type %v", fieldName.ID, field)
								if field.Assertion != nil && field.Assertion.BaseType != nil {
									return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
								}
							} else {
								log.Tracef("[lookupFieldType] Field %s not found in shape fields", fieldName.ID)
							}
						}
					}
				}
			}
		}
	} else {
		log.Tracef("[lookupFieldType] No types found in VariableTypes for %s", fieldName.ID)
	}

	// If we didn't find the field in VariableTypes, try to look it up in the type definition
	if def, exists := tc.Defs[baseType.Ident]; exists {
		log.Tracef("[lookupFieldType] Found type definition for %s: %v", baseType.Ident, def)
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			log.Tracef("[lookupFieldType] Type definition is TypeDefNode: %v", typeDef)
			// Try both value and pointer types for TypeDefAssertionExpr
			var typeDefExpr ast.TypeDefAssertionExpr
			if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
				typeDefExpr = expr
			} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
				typeDefExpr = *expr
			} else if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				// Handle direct shape definition
				log.Tracef("[lookupFieldType] Processing direct shape definition: %v", shapeExpr.Shape)
				if field, exists := shapeExpr.Shape.Fields[string(fieldName.ID)]; exists {
					log.Tracef("[lookupFieldType] Found field %s in shape: %v", fieldName.ID, field)
					if field.Assertion != nil && field.Assertion.BaseType != nil {
						return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
					}
				} else {
					log.Tracef("[lookupFieldType] Field %s not found in shape", fieldName.ID)
				}
				return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
			} else {
				log.Tracef("[lookupFieldType] Type definition expression is not TypeDefAssertionExpr or TypeDefShapeExpr: %T", typeDef.Expr)
				return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
			}

			log.Tracef("[lookupFieldType] Type definition expression is TypeDefAssertionExpr: %v", typeDefExpr)
			if typeDefExpr.Assertion != nil {
				log.Tracef("[lookupFieldType] Processing %d constraints from type definition: %v",
					len(typeDefExpr.Assertion.Constraints), typeDefExpr.Assertion.Constraints)
				// Collect fields from constraints
				mergedFields := MergeShapeFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
				log.Tracef("[lookupFieldType] Merged fields from constraints: %v", mergedFields)
				if field, exists := mergedFields[string(fieldName.ID)]; exists {
					log.Tracef("[lookupFieldType] Found field %s in merged fields: %v", fieldName.ID, field)
					if field.Assertion != nil && field.Assertion.BaseType != nil {
						return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
					}
				} else {
					log.Tracef("[lookupFieldType] Field %s not found in merged fields", fieldName.ID)
				}
			}
		} else {
			log.Tracef("[lookupFieldType] Type definition is not TypeDefNode: %T", def)
		}
	} else {
		log.Tracef("[lookupFieldType] No type definition found for %s", baseType.Ident)
	}

	return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
}

// LookupFunctionReturnType looks up the return type of a function node
func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Ident.ID]
	if !exists {
		err := fmt.Errorf("undefined function: %s", function.Ident)
		log.WithError(err).Error("lookup function return type failed")
		return nil, err
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
			err := fmt.Errorf("expected single type for assertion %s but got %d types", hash.ToTypeIdent(), len(existingType))
			log.WithError(err).Error("lookup assertion type failed")
			return nil, err
		}
		log.Trace(fmt.Sprintf("existingType: %s", existingType))
		return &existingType[0], nil
	}
	typeNode := &ast.TypeNode{
		Ident:     hash.ToTypeIdent(),
		Assertion: assertion,
	}
	log.Trace(fmt.Sprintf("Storing new looked up assertion type: %s", typeNode.Ident))
	tc.storeInferredType(assertion, []ast.TypeNode{*typeNode})
	return typeNode, nil
}
