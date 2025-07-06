package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"time"

	logrus "github.com/sirupsen/logrus"
)

const ConstraintMatch = "Match"

// TODO: Improve assertion type inference
// This should handle:
// 1. Complex constraints
// 2. Nested assertions
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) InferAssertionType(assertion *ast.AssertionNode, isTypeGuard bool) ([]ast.TypeNode, error) {
	if assertion == nil {
		return nil, nil
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"assertion": assertion,
	}).Tracef("Inferring type for assertion")

	// First, get the base type if it exists
	var baseType ast.TypeIdent
	if assertion.BaseType != nil {
		baseType = *assertion.BaseType
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"baseType": baseType,
		}).Tracef("Assertion has base type")
	}

	// Create a new shape type to hold the merged fields
	mergedFields := make(map[string]ast.ShapeFieldNode)

	// If we have a base type, get its fields first
	if baseType != "" {
		if def, exists := tc.Defs[baseType]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					for k, v := range shapeExpr.Shape.Fields {
						mergedFields[k] = v
						tc.log.WithFields(logrus.Fields{
							"function": "inferAssertionType",
							"field":    k,
							"value":    v,
						}).Tracef("Added field from base type")
					}
				} else if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					if assertionExpr.Assertion != nil {
						baseFields := tc.resolveShapeFieldsFromAssertion(assertionExpr.Assertion)
						for k, v := range baseFields {
							mergedFields[k] = v
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"value":    v,
							}).Tracef("Added field from base assertion")
						}
					}
				}
			}
		}
	}

	// Process each constraint
	for _, constraint := range assertion.Constraints {
		tc.log.WithFields(logrus.Fields{
			"function":   "inferAssertionType",
			"constraint": constraint.Name,
			"args":       constraint.Args,
		}).Tracef("Processing constraint")

		// Handle Match constraint specially - merge shape fields directly
		if constraint.Name == ConstraintMatch {
			tc.log.WithFields(logrus.Fields{
				"function":   "inferAssertionType",
				"constraint": constraint.Name,
			}).Debugf("Processing Match constraint")

			for _, arg := range constraint.Args {
				if arg.Shape != nil {
					tc.log.WithFields(logrus.Fields{
						"function":    "inferAssertionType",
						"shape":       arg.Shape,
						"shapeFields": arg.Shape.Fields,
					}).Debugf("Merging fields from Match constraint shape")

					// Ensure the shape type is inferred and registered (including nested shapes)
					tc.log.WithFields(logrus.Fields{
						"function": "inferAssertionType",
						"argShape": fmt.Sprintf("%#v", arg.Shape),
					}).Debugf("Calling inferShapeType with shape node")
					inferredTypes, err := tc.inferShapeType(*arg.Shape)
					if err != nil {
						return nil, err
					}

					var shapeTypeIdent ast.TypeIdent
					if len(inferredTypes) > 0 {
						shapeTypeIdent = inferredTypes[0].Ident
						tc.log.WithFields(logrus.Fields{
							"function":       "inferAssertionType",
							"shapeTypeIdent": shapeTypeIdent,
							"shape":          arg.Shape,
						}).Debugf("Using shapeTypeIdent for merged field type")
					}

					// Merge shape fields directly, but set the field type to the registered shape type
					for k, v := range arg.Shape.Fields {
						tc.log.WithFields(logrus.Fields{
							"function":     "inferAssertionType",
							"field":        k,
							"fieldNode":    fmt.Sprintf("%+v", v),
							"hasShape":     v.Shape != nil,
							"hasType":      v.Type != nil,
							"hasAssertion": v.Assertion != nil,
						}).Debugf("Processing field from Match constraint")

						// Handle nested shapes by inferring their concrete types
						if v.Shape != nil {
							// Infer the nested shape type
							nestedTypes, err := tc.inferShapeType(*v.Shape)
							if err != nil {
								return nil, err
							}
							var nestedTypeIdent ast.TypeIdent
							if len(nestedTypes) > 0 {
								nestedTypeIdent = nestedTypes[0].Ident
							}
							// Create field with the concrete nested shape type
							mergedFields[k] = ast.ShapeFieldNode{
								Type:      &ast.TypeNode{Ident: nestedTypeIdent},
								Assertion: nil, // Always clear assertion for concrete type
								Shape:     nil, // Not a nested shape here
							}
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"type":     nestedTypeIdent,
							}).Debugf("Added nested shape field from Match constraint with registered shape type")
						} else {
							// For non-nested fields, copy the field node as-is (preserve its type, e.g., String)
							mergedFields[k] = v
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"type":     v.Type.Ident,
							}).Debugf("Added field from Match constraint with actual field type")
						}
					}
				}
			}
			continue // Skip normal type guard processing for Match constraint
		}

		// Get the type guard definition
		guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]
		if !exists {
			return nil, fmt.Errorf("type guard %s not found", constraint.Name)
		}

		// Push a new scope for the type guard's body
		var guardNode ast.TypeGuardNode
		if ptr, ok := guardDef.(*ast.TypeGuardNode); ok {
			guardNode = *ptr
		} else {
			guardNode = guardDef.(ast.TypeGuardNode)
		}

		tc.log.WithFields(logrus.Fields{
			"function":    "inferAssertionType",
			"subject":     guardNode.Subject.GetIdent(),
			"subjectType": guardNode.Subject.GetType().Ident,
			"parameters":  guardNode.Parameters(),
		}).Tracef("Subject parameter and additional parameters")

		// Map constraint arguments to parameters
		argMap := make(map[string]ast.Node)
		for i, arg := range constraint.Args {
			if i+1 < len(guardNode.Parameters()) {
				param := guardNode.Parameters()[i+1] // +1 to skip subject parameter
				argMap[param.GetIdent()] = arg
				tc.log.WithFields(logrus.Fields{
					"function": "inferAssertionType",
					"param":    param.GetIdent(),
					"arg":      fmt.Sprintf("%+v", arg),
				}).Tracef("Mapping parameter to argument")
			}
		}

		// For all constraints, process each parameter
		for _, param := range guardNode.Parameters() {
			if param.GetIdent() != guardNode.Subject.GetIdent() {
				tc.log.WithFields(logrus.Fields{
					"function":  "inferAssertionType",
					"param":     param.GetIdent(),
					"paramType": param.GetType().Ident,
				}).Tracef("Processing parameter")

				// If we have an argument for this parameter, use its concrete type
				if arg, ok := argMap[param.GetIdent()]; ok {
					tc.log.WithFields(logrus.Fields{
						"function": "inferAssertionType",
						"param":    param.GetIdent(),
						"arg":      fmt.Sprintf("%+v", arg),
					}).Tracef("Found argument for parameter")

					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							tc.log.WithFields(logrus.Fields{
								"function":    "inferAssertionType",
								"param":       param.GetIdent(),
								"shape":       argNode.Shape,
								"shapeFields": argNode.Shape.Fields,
							}).Debugf("Argument is a shape literal; merging its fields directly")
							// Ensure the shape type is inferred and registered
							if _, err := tc.inferShapeType(*argNode.Shape); err != nil {
								return nil, err
							}
							// Merge shape fields directly (do NOT add a field for the parameter itself)
							for k, v := range argNode.Shape.Fields {
								if v.Shape != nil {
									// Register the nested shape type for any field
									nestedTypes, err := tc.inferShapeType(*v.Shape)
									if err != nil {
										return nil, err
									}
									var nestedTypeIdent ast.TypeIdent
									if len(nestedTypes) > 0 {
										nestedTypeIdent = nestedTypes[0].Ident
									}
									mergedFields[k] = ast.ShapeFieldNode{
										Type:  &ast.TypeNode{Ident: nestedTypeIdent},
										Shape: v.Shape,
									}
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"field":    k,
										"type":     nestedTypeIdent,
									}).Debugf("Added nested shape field from shape literal argument with registered shape type")
								} else if v.Type != nil && v.Type.Ident == ast.TypeShape && v.Shape == nil && argNode.Shape != nil {
									// Register the provided shape and use its type
									nestedTypes, err := tc.inferShapeType(*argNode.Shape)
									if err != nil {
										return nil, err
									}
									var nestedTypeIdent ast.TypeIdent
									if len(nestedTypes) > 0 {
										nestedTypeIdent = nestedTypes[0].Ident
									}
									mergedFields[k] = ast.ShapeFieldNode{
										Type:  &ast.TypeNode{Ident: nestedTypeIdent},
										Shape: argNode.Shape,
									}
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"field":    k,
										"type":     nestedTypeIdent,
									}).Debugf("Fixed: Added nested shape field from argument-provided shape type")
									continue
								} else if v.Type != nil {
									// If the field already has a type, preserve it
									mergedFields[k] = v
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"field":    k,
										"type":     v.Type.Ident,
									}).Debugf("Added field from shape literal argument with existing type")
								} else {
									// If the field is a shape type and has an assertion, use the inferred type
									if v.Type != nil && v.Type.Ident == ast.TypeShape && v.Type.Assertion != nil {
										inferredTypes, err := tc.InferAssertionType(v.Type.Assertion, false)
										if err != nil {
											return nil, err
										}
										var inferredTypeIdent ast.TypeIdent
										if len(inferredTypes) > 0 {
											inferredTypeIdent = inferredTypes[0].Ident
										}
										mergedFields[k] = ast.ShapeFieldNode{
											Type:  &ast.TypeNode{Ident: inferredTypeIdent},
											Shape: nil,
										}
										tc.log.WithFields(logrus.Fields{
											"function": "inferAssertionType",
											"field":    k,
											"type":     inferredTypeIdent,
										}).Debugf("Fixed: Added nested shape field from argument field assertion type")
										continue
									}
									// Otherwise, just copy the field node
									mergedFields[k] = v
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"field":    k,
										"value":    v,
									}).Debugf("Added field from shape literal argument (preserving nested structure)")
								}
							}
							tc.log.WithFields(logrus.Fields{
								"function":     "inferAssertionType",
								"param":        param.GetIdent(),
								"mergedFields": mergedFields,
							}).Debugf("After merging shape literal fields")
							continue // skip adding a field for the parameter itself
						} else if argNode.Type != nil {
							tc.log.WithFields(logrus.Fields{
								"function":  "inferAssertionType",
								"param":     param.GetIdent(),
								"type":      argNode.Type.Ident,
								"assertion": argNode.Type.Assertion,
							}).Tracef("Argument is a type")
							// If the argument is a TypeNode with an Assertion (e.g., Match constraint with shape literal),
							if argNode.Type.Assertion != nil {
								inferredTypes, err := tc.InferAssertionType(argNode.Type.Assertion, false)
								if err != nil {
									return nil, err
								}
								var concreteType ast.TypeIdent
								if len(inferredTypes) > 0 {
									concreteType = inferredTypes[0].Ident
								}
								// Look up the inferred type definition
								if def, ok := tc.Defs[concreteType].(ast.TypeDefNode); ok {
									if shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr); ok {
										matchShape := (*ast.ShapeNode)(nil)
										for _, c := range argNode.Type.Assertion.Constraints {
											if c.Name == "Match" && len(c.Args) > 0 && c.Args[0].Shape != nil {
												matchShape = c.Args[0].Shape
												break
											}
										}
										for fieldName, fieldNode := range shapeExpr.Shape.Fields {
											if fieldNode.Type != nil && fieldNode.Type.Ident == ast.TypeShape && matchShape != nil {
												if argField, ok := matchShape.Fields[fieldName]; ok {
													if argField.Type != nil && argField.Type.Assertion != nil {
														inferredNestedTypes, err := tc.InferAssertionType(argField.Type.Assertion, false)
														if err != nil {
															return nil, err
														}
														var nestedTypeIdent ast.TypeIdent
														if len(inferredNestedTypes) > 0 {
															nestedTypeIdent = inferredNestedTypes[0].Ident
														}
														mergedFields[fieldName] = ast.ShapeFieldNode{
															Type:  &ast.TypeNode{Ident: nestedTypeIdent},
															Shape: nil,
														}
														tc.log.WithFields(logrus.Fields{
															"function": "inferAssertionType",
															"field":    fieldName,
															"type":     nestedTypeIdent,
														}).Debugf("Matched and inferred nested field from argument assertion")
														continue
													}
													// Otherwise, use the argument field's type directly
													mergedFields[fieldName] = argField
													tc.log.WithFields(logrus.Fields{
														"function": "inferAssertionType",
														"field":    fieldName,
														"type":     argField.Type.Ident,
													}).Debugf("Matched and used argument field type for nested field")
													continue
												}
											}
										}
										continue
									}
								}
								// Fallback: only set the parameter field if the inferred type is not a shape
								mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
									Type:      &ast.TypeNode{Ident: concreteType},
									Assertion: nil,
									Shape:     nil,
								}
								tc.log.WithFields(logrus.Fields{
									"function": "inferAssertionType",
									"field":    param.GetIdent(),
									"type":     concreteType,
								}).Debugf("Added field with concrete type from TypeNode assertion (fallback)")
							}
						} else {
							// If the argument is a TypeNode without assertion, try to merge fields from its type definition
							argType := argNode.Type.Ident
							tc.log.WithFields(logrus.Fields{
								"function":       "inferAssertionType",
								"param":          param.GetIdent(),
								"argType":        argType,
								"defsHasArgType": (tc.Defs[argType] != nil),
							}).Debugf("Checking for type definition for argument type")
							if def, ok := tc.Defs[argType].(ast.TypeDefNode); ok {
								if shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr); ok {
									// Merge fields from the type definition
									for fieldName, fieldNode := range shapeExpr.Shape.Fields {
										mergedFields[fieldName] = fieldNode
										tc.log.WithFields(logrus.Fields{
											"function": "inferAssertionType",
											"field":    fieldName,
											"type":     fieldNode.Type.Ident,
										}).Debugf("Merged field from type definition")
									}
									tc.log.WithFields(logrus.Fields{
										"function":     "inferAssertionType",
										"param":        param.GetIdent(),
										"mergedFields": mergedFields,
									}).Debugf("Merged fields from type definition for parameter")
									continue // skip adding a field for the parameter itself and fallback
								}
							}
							// Fallback: Add the parameter name as a field with the argument's type
							mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
								Type: &ast.TypeNode{Ident: argType},
							}
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    param.GetIdent(),
								"value":    argNode.Type.Ident,
							}).Tracef("Added field with concrete type")
						}
					}
				}
				// Only add a field for the parameter if no argument or not a shape
				paramType := param.GetType().Ident
				mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
					Assertion: &ast.AssertionNode{
						BaseType: &paramType,
					},
				}
			}
		}
	}

	// Use the structural hash for this assertion node
	hash, err := tc.Hasher.HashNode(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to hash assertion during inferAssertionType: %s", err)
	}
	typeIdent := hash.ToTypeIdent()
	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"assertion": assertion,
		"fields":    mergedFields,
		"typeIdent": typeIdent,
	}).Tracef("Stored shape type with fields")

	// Store the shape type
	tc.Defs[typeIdent] = ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: mergedFields,
			},
		},
	}

	shapeType := ast.TypeNode{Ident: typeIdent}
	tc.storeInferredType(assertion, []ast.TypeNode{shapeType})

	tc.log.WithFields(logrus.Fields{
		"function":     "inferAssertionType",
		"mergedFields": mergedFields,
	}).Debugf("Final merged fields before storing shape type")
	for k, v := range mergedFields {
		if v.Type != nil {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferAssertionType",
				"field":     k,
				"typeIdent": v.Type.Ident,
			}).Debugf("Field type ident after merging")
		}
	}

	return []ast.TypeNode{shapeType}, nil
}

func generateUniqueID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
