package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
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
	tc.log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, scope.String())

	// Split the identifier on dots to handle field access
	parts := strings.Split(string(variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])

	// Look up the base variable
	symbol, exists := scope.LookupVariable(baseIdent, true)
	if !exists {
		err := fmt.Errorf("undefined symbol: %s [scope: %s]", parts[0], scope.String())
		tc.log.WithError(err).Error("lookup symbol failed")
		return ast.TypeNode{}, err
	}

	if len(symbol.Types) != 1 {
		err := fmt.Errorf("expected single type for variable %s but got %d types", parts[0], len(symbol.Types))
		tc.log.WithError(err).Error("lookup symbol failed")
		return ast.TypeNode{}, err
	}

	// If there are no field accesses, return the base type
	if len(parts) == 1 {
		return symbol.Types[0], nil
	}

	// Handle field access by looking up each field in the type
	currentType := symbol.Types[0]
	// Track the original assertion chain from the base type
	originalAssertion := currentType.Assertion
	for i := 1; i < len(parts); i++ {
		// Always pass the original assertion chain for all field accesses
		fieldType, err := tc.lookupFieldType(currentType, ast.Ident{ID: ast.Identifier(parts[i])}, originalAssertion)
		if err != nil {
			return ast.TypeNode{}, err
		}
		currentType = fieldType
	}

	return currentType, nil
}

// mergeShapeFields merges fields from a shape into the target map
func (tc *TypeChecker) mergeShapeFields(target map[string]ast.ShapeFieldNode, shape ast.ShapeNode) {
	tc.log.Debugf("[mergeShapeFields] Starting merge with target: %+v", target)
	tc.log.Debugf("[mergeShapeFields] Merging shape fields: %+v", shape.Fields)

	for fieldName, field := range shape.Fields {
		tc.log.Debugf("[mergeShapeFields] Processing field %s: %+v", fieldName, field)
		if existingField, exists := target[fieldName]; exists {
			tc.log.Debugf("[mergeShapeFields] Field %s already exists with value: %+v", fieldName, existingField)
			// If both fields have assertions, merge them
			if field.Assertion != nil && existingField.Assertion != nil {
				tc.log.Debugf("[mergeShapeFields] Both fields have assertions, merging...")
				tc.log.Debugf("[mergeShapeFields] Existing assertion: %+v", existingField.Assertion)
				tc.log.Debugf("[mergeShapeFields] New assertion: %+v", field.Assertion)
				// Create a new assertion that combines both
				mergedAssertion := &ast.AssertionNode{
					BaseType:    existingField.Assertion.BaseType,
					Constraints: append(existingField.Assertion.Constraints, field.Assertion.Constraints...),
				}
				target[fieldName] = ast.ShapeFieldNode{
					Assertion: mergedAssertion,
				}
				tc.log.Debugf("[mergeShapeFields] Merged field %s with new assertion: %+v", fieldName, mergedAssertion)
			} else {
				// If one has an assertion and the other has a shape, prefer the assertion
				if field.Assertion != nil {
					tc.log.Debugf("[mergeShapeFields] Using new field with assertion for %s", fieldName)
					target[fieldName] = field
				} else if existingField.Assertion != nil {
					tc.log.Debugf("[mergeShapeFields] Keeping existing field with assertion for %s", fieldName)
					// Keep existing field
				} else {
					// If both are shapes, merge them recursively
					if field.Shape != nil && existingField.Shape != nil {
						tc.log.Debugf("[mergeShapeFields] Both fields are shapes, merging recursively")
						tc.mergeShapeFields(target, *field.Shape)
					} else {
						tc.log.Debugf("[mergeShapeFields] Using new field for %s", fieldName)
						target[fieldName] = field
					}
				}
			}
		} else {
			tc.log.Debugf("[mergeShapeFields] Adding new field %s: %+v", fieldName, field)
			target[fieldName] = field
		}
	}
	tc.log.Debugf("[mergeShapeFields] Final target after merge: %+v", target)
}

// resolveMergedShapeFields recursively merges all shape fields from an assertion chain
func (tc *TypeChecker) resolveMergedShapeFields(assertion *ast.AssertionNode) map[string]ast.ShapeFieldNode {
	merged := make(map[string]ast.ShapeFieldNode)
	if assertion == nil {
		tc.log.Debugf("[resolveMergedShapeFields] Assertion is nil, returning empty map")
		return merged
	}

	tc.log.Debugf("[resolveMergedShapeFields] Starting with assertion: %+v", assertion)
	tc.log.Debugf("[resolveMergedShapeFields] BaseType: %v, Constraints: %+v", assertion.BaseType, assertion.Constraints)

	// First, handle the base type if it exists
	if assertion.BaseType != nil {
		tc.log.Debugf("[resolveMergedShapeFields] Processing base type: %s", *assertion.BaseType)
		if def, exists := tc.Defs[*assertion.BaseType]; exists {
			tc.log.Debugf("[resolveMergedShapeFields] Found base type definition: %T %+v", def, def)
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				tc.log.Debugf("[resolveMergedShapeFields] Base type is TypeDefNode: %s, Expr: %T %+v",
					typeDef.Ident, typeDef.Expr, typeDef.Expr)

				// Handle both value and pointer types for TypeDefAssertionExpr
				var baseAssertionExpr ast.TypeDefAssertionExpr
				if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					baseAssertionExpr = expr
				} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
					baseAssertionExpr = *expr
				}

				if baseAssertionExpr.Assertion != nil {
					tc.log.Debugf("[resolveMergedShapeFields] Recursively merging fields from base assertion: %+v",
						baseAssertionExpr.Assertion)
					baseFields := tc.resolveMergedShapeFields(baseAssertionExpr.Assertion)
					tc.log.Debugf("[resolveMergedShapeFields] Got base fields: %+v", baseFields)
					for k, v := range baseFields {
						merged[k] = v
						tc.log.Debugf("[resolveMergedShapeFields] Added field from base: %s => %+v", k, v)
					}
				}
			}
		}
	}

	// Then process each constraint
	for i, constraint := range assertion.Constraints {
		tc.log.Debugf("[resolveMergedShapeFields] Processing constraint %d: %s with args: %+v", i, constraint.Name, constraint.Args)

		// Special handling for mutation types
		if assertion.BaseType != nil && (*assertion.BaseType == "trpc.Mutation" || *assertion.BaseType == "trpc.Query") {
			if constraint.Name == "Input" && len(constraint.Args) > 0 {
				if arg := constraint.Args[0]; arg.Shape != nil {
					tc.log.Debugf("[resolveMergedShapeFields] Processing mutation input shape: %+v", arg.Shape)
					for k, v := range arg.Shape.Fields {
						merged[k] = v
						tc.log.Debugf("[resolveMergedShapeFields] Added field from mutation input: %s => %+v", k, v)
					}
				}
				continue
			}
		}

		if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
			if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
				tc.log.Debugf("[resolveMergedShapeFields] Type guard node: %+v", guardNode)
				tc.log.Debugf("[resolveMergedShapeFields] Type guard parameters: %+v", guardNode.Parameters())

				// Add fields from type guard parameters
				for _, param := range guardNode.Parameters() {
					if param.GetIdent() != guardNode.Subject.GetIdent() {
						tc.log.Debugf("[resolveMergedShapeFields] Adding field from parameter: %s", param.GetIdent())
						typeIdent := param.GetType().Ident
						if _, exists := merged[param.GetIdent()]; !exists {
							merged[param.GetIdent()] = ast.ShapeFieldNode{
								Assertion: &ast.AssertionNode{
									BaseType: &typeIdent,
								},
							}
							tc.log.Debugf("[resolveMergedShapeFields] Added new field from parameter: %s => %+v", param.GetIdent(), merged[param.GetIdent()])
						} else {
							tc.log.Debugf("[resolveMergedShapeFields] Field already exists from parameter: %s", param.GetIdent())
						}
					}
				}

				// Process type guard arguments
				for j, arg := range constraint.Args {
					tc.log.Debugf("[resolveMergedShapeFields] Processing type guard arg %d: %+v", j, arg)

					if arg.Shape != nil {
						tc.log.Debugf("[resolveMergedShapeFields] Processing shape argument: %+v", arg.Shape)
						for k, v := range arg.Shape.Fields {
							merged[k] = v
							tc.log.Debugf("[resolveMergedShapeFields] Added field from shape argument: %s => %+v", k, v)
						}
					} else if arg.Type != nil {
						tc.log.Debugf("[resolveMergedShapeFields] Processing type argument: %+v", arg.Type)
						if def, exists := tc.Defs[arg.Type.Ident]; exists {
							if typeDef, ok := def.(ast.TypeDefNode); ok {
								if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
									tc.log.Debugf("[resolveMergedShapeFields] Found TypeDefShapeExpr for %s with fields: %+v", arg.Type.Ident, shapeExpr.Shape.Fields)
									for k, v := range shapeExpr.Shape.Fields {
										merged[k] = v
										tc.log.Debugf("[resolveMergedShapeFields] Added field from type argument shape: %s => %+v", k, v)
									}
								} else if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
									tc.log.Debugf("[resolveMergedShapeFields] Found TypeDefAssertionExpr for %s", arg.Type.Ident)
									if assertionExpr.Assertion != nil && assertionExpr.Assertion.BaseType != nil && *assertionExpr.Assertion.BaseType == ast.TypeShape {
										tc.log.Debugf("[resolveMergedShapeFields] Recursively merging fields from assertion-based shape alias: %s", arg.Type.Ident)
										baseFields := tc.resolveMergedShapeFields(assertionExpr.Assertion)
										for k, v := range baseFields {
											merged[k] = v
											tc.log.Debugf("[resolveMergedShapeFields] Added field from type argument assertion: %s => %+v", k, v)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	tc.log.Debugf("[resolveMergedShapeFields] Final merged fields: %+v", merged)
	return merged
}

// lookupNestedField recursively looks up a field in a nested shape
func (tc *TypeChecker) lookupNestedField(shape *ast.ShapeNode, fieldName ast.Ident) (ast.TypeNode, error) {
	if shape == nil {
		return ast.TypeNode{}, fmt.Errorf("shape is nil")
	}
	if field, exists := shape.Fields[string(fieldName.ID)]; exists {
		tc.log.Tracef("[lookupNestedField] Found field %s: %v", fieldName.ID, field)
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
func (tc *TypeChecker) lookupFieldType(baseType ast.TypeNode, fieldName ast.Ident, parentAssertion *ast.AssertionNode) (ast.TypeNode, error) {
	tc.log.Debugf("[lookupFieldType] Starting lookup for field {%s} in type %s", fieldName, baseType.Ident)

	// Get the type definition
	def, exists := tc.Defs[baseType.Ident]
	if !exists {
		tc.log.Debugf("[lookupFieldType] No type definition found for %s", baseType.Ident)
		// Don't throw error for Shape type
		if baseType.Ident == "Shape" {
			// Try to look up the field in the merged fields from the assertion chain
			assertion := baseType.Assertion
			if assertion == nil {
				assertion = parentAssertion
			}
			if assertion != nil {
				mergedFields := tc.resolveMergedShapeFields(assertion)
				if field, exists := mergedFields[string(fieldName.ID)]; exists {
					tc.log.Debugf("[lookupFieldType] Found field %s in merged fields for Shape: %+v", fieldName.ID, field)
					if field.Assertion != nil && field.Assertion.BaseType != nil {
						return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
					}
					if field.Shape != nil {
						return tc.lookupNestedField(field.Shape, fieldName)
					}
					return ast.TypeNode{Ident: ast.TypeShape}, nil
				}
			}
			return ast.TypeNode{}, nil
		}
		return ast.TypeNode{}, fmt.Errorf("base type %s for field %s not found", baseType.Ident, fieldName)
	}

	tc.log.Debugf("[lookupFieldType] Base type details: %+v", baseType)

	// If the base type is an assertion type, recursively merge all fields from the assertion chain
	if baseType.Assertion != nil {
		tc.log.Debugf("[lookupFieldType] Base type has assertion: %+v", baseType.Assertion)
		mergedFields := tc.resolveMergedShapeFields(baseType.Assertion)
		tc.log.Debugf("[lookupFieldType] Merged fields from assertion chain: %+v", mergedFields)
		if field, exists := mergedFields[string(fieldName.ID)]; exists {
			tc.log.Debugf("[lookupFieldType] Found field %s in merged fields: %+v", fieldName.ID, field)
			if field.Assertion != nil && field.Assertion.BaseType != nil {
				// Handle pointer types
				if *field.Assertion.BaseType == ast.TypePointer {
					if len(field.Assertion.Constraints) > 0 {
						for _, constraint := range field.Assertion.Constraints {
							if constraint.Name == "Value" && len(constraint.Args) > 0 {
								if arg := constraint.Args[0]; arg.Type != nil {
									return ast.TypeNode{Ident: arg.Type.Ident}, nil
								}
							}
						}
					}
				}
				return ast.TypeNode{Ident: *field.Assertion.BaseType, Assertion: field.Assertion}, nil
			}
			if field.Shape != nil {
				tc.log.Debugf("[lookupFieldType] Field has nested shape: %+v", field.Shape)
				// Always propagate the assertion chain from the parent if the current type is Shape
				var assertion *ast.AssertionNode = baseType.Assertion
				if baseType.Ident == ast.TypeShape && baseType.Assertion == nil {
					assertion = parentAssertion
				}
				return tc.lookupFieldType(ast.TypeNode{Ident: ast.TypeShape, Assertion: assertion}, fieldName, assertion)
			}
			return ast.TypeNode{Ident: ast.TypeShape, Assertion: baseType.Assertion}, nil
		}
		tc.log.Debugf("[lookupFieldType] Field %s not found in merged fields", fieldName.ID)
	}

	// If we have a shape type definition, look up the field in the shape
	if shapeDef, ok := def.(ast.TypeDefNode); ok {
		if shapeExpr, ok := shapeDef.Expr.(ast.TypeDefShapeExpr); ok {
			tc.log.Debugf("[lookupFieldType] Processing direct shape definition: %+v", shapeExpr)
			if field, exists := shapeExpr.Shape.Fields[string(fieldName.ID)]; exists {
				tc.log.Debugf("[lookupFieldType] Found field %s in shape: %+v", fieldName.ID, field)
				if field.Assertion != nil && field.Assertion.BaseType != nil {
					// Handle pointer types
					if *field.Assertion.BaseType == ast.TypePointer {
						// For pointer types, we need to look up the base type
						if len(field.Assertion.Constraints) > 0 {
							for _, constraint := range field.Assertion.Constraints {
								if constraint.Name == "Value" && len(constraint.Args) > 0 {
									if arg := constraint.Args[0]; arg.Type != nil {
										return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
									}
								}
							}
						}
					}
					return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
				}
				if field.Shape != nil {
					tc.log.Debugf("[lookupFieldType] Field has nested shape: %+v", field.Shape)
					// Always propagate the assertion chain from the parent if the current type is Shape
					var assertion *ast.AssertionNode = baseType.Assertion
					if baseType.Ident == ast.TypeShape && baseType.Assertion == nil {
						assertion = parentAssertion
					}
					return tc.lookupFieldType(ast.TypeNode{Ident: ast.TypeShape, Assertion: assertion}, fieldName, assertion)
				}
				return ast.TypeNode{Ident: ast.TypeShape}, nil
			}
		}
	}

	// If we have a type guard, look up the field in the type guard's parameter
	if guardDef, ok := def.(ast.TypeGuardNode); ok {
		tc.log.Debugf("[lookupFieldType] Found type guard: %+v", guardDef)
		for _, param := range guardDef.Parameters() {
			if param.GetIdent() == string(fieldName.ID) {
				tc.log.Debugf("[lookupFieldType] Found field %s in type guard parameter: %+v", fieldName.ID, param)
				return param.GetType(), nil
			}
		}
	}

	return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
}

// LookupFunctionReturnType looks up the return type of a function node
func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Ident.ID]
	if !exists {
		err := fmt.Errorf("undefined function: %s", function.Ident)
		tc.log.WithError(err).Error("lookup function return type failed")
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
			tc.log.WithError(err).Error("lookup assertion type failed")
			return nil, err
		}
		tc.log.Trace(fmt.Sprintf("existingType: %s", existingType))
		return &existingType[0], nil
	}
	typeNode := &ast.TypeNode{
		Ident:     hash.ToTypeIdent(),
		Assertion: assertion,
	}
	tc.log.Trace(fmt.Sprintf("Storing new looked up assertion type: %s", typeNode.Ident))
	tc.storeInferredType(assertion, []ast.TypeNode{*typeNode})
	return typeNode, nil
}
