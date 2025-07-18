package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"strings"
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
func (tc *TypeChecker) InferAssertionType(assertion *ast.AssertionNode, isFunctionParam bool, fieldName string, expectedType *ast.TypeNode) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"assertion":       assertion,
		"isFunctionParam": isFunctionParam,
		"fieldName":       fieldName,
		"function":        "inferAssertionType",
	}).Trace("Inferring type for assertion")

	// If this assertion has a base type, start with that
	mergedFields := make(map[string]ast.ShapeFieldNode)
	if assertion.BaseType != nil {
		// Look up the base type definition
		baseTypeDef, exists := tc.Defs[*assertion.BaseType]
		if !exists {
			return nil, fmt.Errorf("base type %s not found", *assertion.BaseType)
		}

		// Extract fields from the base type
		if shapeExpr, ok := baseTypeDef.(ast.TypeDefNode); ok {
			if shapeDef, ok := shapeExpr.Expr.(ast.TypeDefShapeExpr); ok {
				for name, field := range shapeDef.Shape.Fields {
					mergedFields[name] = field
				}
			}
		}
	}

	// Special handling for Value constraints
	if len(assertion.Constraints) == 1 && assertion.Constraints[0].Name == ast.ValueConstraint {
		resolvedType, err := tc.inferValueConstraintType(assertion.Constraints[0], fieldName, expectedType)
		if err != nil {
			return nil, err
		}
		return []ast.TypeNode{resolvedType}, nil
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

					// Merge shape fields directly into the mergedFields (preserving base type fields)
					for k, v := range arg.Shape.Fields {
						if v.Shape != nil {
							// Register the nested shape type for any field
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"shape":    fmt.Sprintf("%+v", *v.Shape),
							}).Debugf("Calling inferShapeType with shape node")
							nestedType, err := tc.inferShapeType(*v.Shape, nil)
							if err != nil {
								return nil, fmt.Errorf("failed to infer nested shape type: %w", err)
							}
							mergedFields[k] = ast.ShapeFieldNode{
								Type:  &ast.TypeNode{Ident: nestedType.Ident},
								Shape: v.Shape,
							}
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"type":     nestedType.Ident,
							}).Debugf("Added nested shape field from Match constraint with registered shape type")
						} else if v.Type != nil {
							// If the field already has a type, preserve it
							mergedFields[k] = v
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"type":     v.Type.Ident,
							}).Debugf("Added field from Match constraint shape with existing type")
						} else {
							// Otherwise, just copy the field node
							mergedFields[k] = v
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"value":    v,
							}).Debugf("Added field from Match constraint shape (preserving structure)")
						}
					}
					tc.log.WithFields(logrus.Fields{
						"function":     "inferAssertionType",
						"constraint":   constraint.Name,
						"mergedFields": mergedFields,
					}).Debugf("After merging Match constraint shape fields")
					continue // Skip normal type guard processing for Match constraint
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
							if _, err := tc.inferShapeType(*argNode.Shape, nil); err != nil {
								return nil, fmt.Errorf("failed to infer shape type: %w", err)
							}
							// Merge shape fields directly (do NOT add a field for the parameter itself)
							for k, v := range argNode.Shape.Fields {
								if v.Shape != nil {
									// Register the nested shape type for any field
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"shape":    fmt.Sprintf("%+v", *v.Shape),
									}).Debugf("Calling inferShapeType with shape node")
									nestedType, err := tc.inferShapeType(*v.Shape, nil)
									if err != nil {
										return nil, fmt.Errorf("failed to infer nested shape type: %w", err)
									}
									mergedFields[k] = ast.ShapeFieldNode{
										Type:  &ast.TypeNode{Ident: nestedType.Ident},
										Shape: v.Shape,
									}
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"field":    k,
										"type":     nestedType.Ident,
									}).Debugf("Added nested shape field from shape literal argument with registered shape type")
								} else if v.Type != nil && v.Type.Ident == ast.TypeShape && v.Shape == nil && argNode.Shape != nil {
									// Register the provided shape and use its type
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"shape":    fmt.Sprintf("%+v", *argNode.Shape),
									}).Debugf("Calling inferShapeType with shape node")
									nestedType, err := tc.inferShapeType(*argNode.Shape, nil)
									if err != nil {
										return nil, fmt.Errorf("failed to infer nested shape type: %w", err)
									}
									mergedFields[k] = ast.ShapeFieldNode{
										Type:  &ast.TypeNode{Ident: nestedType.Ident},
										Shape: argNode.Shape,
									}
									tc.log.WithFields(logrus.Fields{
										"function": "inferAssertionType",
										"field":    k,
										"type":     nestedType.Ident,
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
										inferredTypes, err := tc.InferAssertionType(v.Type.Assertion, false, fieldName, nil)
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
								inferredTypes, err := tc.InferAssertionType(argNode.Type.Assertion, false, fieldName, nil)
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
														inferredNestedTypes, err := tc.InferAssertionType(argField.Type.Assertion, false, fieldName, nil)
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

	// Register the hash-based type and its fields
	RegisterHashBasedType(tc, typeIdent, mergedFields)

	// Robust named type preservation: If BaseType is a named type and compatible, use it
	if assertion.BaseType != nil && !strings.HasPrefix(string(*assertion.BaseType), "T_") {
		baseTypeIdent := *assertion.BaseType
		// Check structural compatibility using the full shape merging logic
		if tc.IsShapeCompatibleWithNamedType(ast.ShapeNode{Fields: mergedFields}, baseTypeIdent) {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferAssertionType",
				"baseType":  baseTypeIdent,
				"typeIdent": typeIdent,
				"note":      "Preserving named type (compatible with full shape logic)",
			}).Warn("[PINPOINT] inferAssertionType: Preserving named type (compatible with full shape logic)")
			return []ast.TypeNode{{Ident: baseTypeIdent}}, nil
		}
		// If not compatible, log and fall back to hash-based type
		tc.log.WithFields(logrus.Fields{
			"function":  "inferAssertionType",
			"baseType":  baseTypeIdent,
			"typeIdent": typeIdent,
			"note":      "BaseType not compatible with full shape logic, using hash-based type",
		}).Warn("[PINPOINT] inferAssertionType: BaseType not compatible, using hash-based type")
	}

	// Default: return the hash-based type
	return []ast.TypeNode{{Ident: typeIdent}}, nil
}

func generateUniqueID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
