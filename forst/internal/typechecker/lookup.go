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
	log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, scope.String())

	// Split the identifier on dots to handle field access
	parts := strings.Split(string(variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])

	// Look up the base variable
	symbol, exists := scope.LookupVariable(baseIdent)
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

// mergeShapeFields merges fields from a shape into the target map
func (tc *TypeChecker) mergeShapeFields(target map[string]ast.ShapeFieldNode, shape ast.ShapeNode) {
	log.Debugf("[mergeShapeFields] Starting merge with target: %+v", target)
	log.Debugf("[mergeShapeFields] Merging shape fields: %+v", shape.Fields)

	for fieldName, field := range shape.Fields {
		log.Debugf("[mergeShapeFields] Processing field %s: %+v", fieldName, field)
		if existingField, exists := target[fieldName]; exists {
			log.Debugf("[mergeShapeFields] Field %s already exists with value: %+v", fieldName, existingField)
			// If both fields have assertions, merge them
			if field.Assertion != nil && existingField.Assertion != nil {
				log.Debugf("[mergeShapeFields] Both fields have assertions, merging...")
				log.Debugf("[mergeShapeFields] Existing assertion: %+v", existingField.Assertion)
				log.Debugf("[mergeShapeFields] New assertion: %+v", field.Assertion)
				// Create a new assertion that combines both
				mergedAssertion := &ast.AssertionNode{
					BaseType:    existingField.Assertion.BaseType,
					Constraints: append(existingField.Assertion.Constraints, field.Assertion.Constraints...),
				}
				target[fieldName] = ast.ShapeFieldNode{
					Assertion: mergedAssertion,
				}
				log.Debugf("[mergeShapeFields] Merged field %s with new assertion: %+v", fieldName, mergedAssertion)
			} else {
				// If one has an assertion and the other has a shape, prefer the assertion
				if field.Assertion != nil {
					log.Debugf("[mergeShapeFields] Using new field with assertion for %s", fieldName)
					target[fieldName] = field
				} else if existingField.Assertion != nil {
					log.Debugf("[mergeShapeFields] Keeping existing field with assertion for %s", fieldName)
					// Keep existing field
				} else {
					// If both are shapes, merge them recursively
					if field.Shape != nil && existingField.Shape != nil {
						log.Debugf("[mergeShapeFields] Both fields are shapes, merging recursively")
						tc.mergeShapeFields(target, *field.Shape)
					} else {
						log.Debugf("[mergeShapeFields] Using new field for %s", fieldName)
						target[fieldName] = field
					}
				}
			}
		} else {
			log.Debugf("[mergeShapeFields] Adding new field %s: %+v", fieldName, field)
			target[fieldName] = field
		}
	}
	log.Debugf("[mergeShapeFields] Final target after merge: %+v", target)
}

// resolveMergedShapeFields recursively merges all shape fields from an assertion chain
func (tc *TypeChecker) resolveMergedShapeFields(assertion *ast.AssertionNode) map[string]ast.ShapeFieldNode {
	merged := make(map[string]ast.ShapeFieldNode)
	if assertion == nil {
		log.Debugf("[resolveMergedShapeFields] Assertion is nil, returning empty map")
		return merged
	}

	log.Debugf("[resolveMergedShapeFields] Starting with assertion: %+v", assertion)
	log.Debugf("[resolveMergedShapeFields] BaseType: %v, Constraints: %+v", assertion.BaseType, assertion.Constraints)

	// First, handle the base type if it exists
	if assertion.BaseType != nil {
		log.Debugf("[resolveMergedShapeFields] Processing base type: %s", *assertion.BaseType)
		if def, exists := tc.Defs[*assertion.BaseType]; exists {
			log.Debugf("[resolveMergedShapeFields] Found base type definition: %T %+v", def, def)
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				log.Debugf("[resolveMergedShapeFields] Base type is TypeDefNode: %s, Expr: %T %+v",
					typeDef.Ident, typeDef.Expr, typeDef.Expr)

				// Handle both value and pointer types for TypeDefAssertionExpr
				var baseAssertionExpr ast.TypeDefAssertionExpr
				if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					baseAssertionExpr = expr
				} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
					baseAssertionExpr = *expr
				}

				if baseAssertionExpr.Assertion != nil {
					log.Debugf("[resolveMergedShapeFields] Recursively merging fields from base assertion: %+v",
						baseAssertionExpr.Assertion)
					baseFields := tc.resolveMergedShapeFields(baseAssertionExpr.Assertion)
					log.Debugf("[resolveMergedShapeFields] Got base fields: %+v", baseFields)
					for k, v := range baseFields {
						merged[k] = v
						log.Debugf("[resolveMergedShapeFields] Added field from base: %s => %+v", k, v)
					}
				}
			}
		}
	}

	// Then process each constraint
	for i, constraint := range assertion.Constraints {
		log.Debugf("[resolveMergedShapeFields] Processing constraint %d: %s with args: %+v", i, constraint.Name, constraint.Args)

		// Special handling for mutation types
		if assertion.BaseType != nil && (*assertion.BaseType == "trpc.Mutation" || *assertion.BaseType == "trpc.Query") {
			if constraint.Name == "Input" && len(constraint.Args) > 0 {
				if arg := constraint.Args[0]; arg.Shape != nil {
					log.Debugf("[resolveMergedShapeFields] Processing mutation input shape: %+v", arg.Shape)
					for k, v := range arg.Shape.Fields {
						merged[k] = v
						log.Debugf("[resolveMergedShapeFields] Added field from mutation input: %s => %+v", k, v)
					}
				}
				continue
			}
		}

		if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
			if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
				log.Debugf("[resolveMergedShapeFields] Type guard node: %+v", guardNode)
				log.Debugf("[resolveMergedShapeFields] Type guard parameters: %+v", guardNode.Parameters())

				// Add fields from type guard parameters
				for _, param := range guardNode.Parameters() {
					if param.GetIdent() != guardNode.Subject.GetIdent() {
						log.Debugf("[resolveMergedShapeFields] Adding field from parameter: %s", param.GetIdent())
						typeIdent := param.GetType().Ident
						if _, exists := merged[param.GetIdent()]; !exists {
							merged[param.GetIdent()] = ast.ShapeFieldNode{
								Assertion: &ast.AssertionNode{
									BaseType: &typeIdent,
								},
							}
							log.Debugf("[resolveMergedShapeFields] Added new field from parameter: %s => %+v", param.GetIdent(), merged[param.GetIdent()])
						} else {
							log.Debugf("[resolveMergedShapeFields] Field already exists from parameter: %s", param.GetIdent())
						}
					}
				}

				// Process type guard arguments
				for j, arg := range constraint.Args {
					log.Debugf("[resolveMergedShapeFields] Processing type guard arg %d: %+v", j, arg)

					if arg.Shape != nil {
						log.Debugf("[resolveMergedShapeFields] Processing shape argument: %+v", arg.Shape)
						for k, v := range arg.Shape.Fields {
							merged[k] = v
							log.Debugf("[resolveMergedShapeFields] Added field from shape argument: %s => %+v", k, v)
						}
					} else if arg.Type != nil {
						log.Debugf("[resolveMergedShapeFields] Processing type argument: %+v", arg.Type)
						if def, exists := tc.Defs[arg.Type.Ident]; exists {
							if typeDef, ok := def.(ast.TypeDefNode); ok {
								if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
									for k, v := range shapeExpr.Shape.Fields {
										merged[k] = v
										log.Debugf("[resolveMergedShapeFields] Added field from type argument shape: %s => %+v", k, v)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	log.Debugf("[resolveMergedShapeFields] Final merged fields: %+v", merged)
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
	log.Debugf("[lookupFieldType] Starting lookup for field %s in type %s", fieldName, baseType.Ident)
	log.Debugf("[lookupFieldType] Base type details: %+v", baseType)

	// If the base type is an assertion type, recursively merge all fields from the assertion chain
	if baseType.Assertion != nil {
		log.Debugf("[lookupFieldType] Base type has assertion: %+v", baseType.Assertion)
		mergedFields := tc.resolveMergedShapeFields(baseType.Assertion)
		log.Debugf("[lookupFieldType] Merged fields from assertion chain: %+v", mergedFields)
		if field, exists := mergedFields[string(fieldName.ID)]; exists {
			log.Debugf("[lookupFieldType] Found field %s in merged fields: %+v", fieldName.ID, field)
			if field.Assertion != nil && field.Assertion.BaseType != nil {
				return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
			}
			if field.Shape != nil {
				log.Debugf("[lookupFieldType] Field has nested shape: %+v", field.Shape)
				return tc.lookupNestedField(field.Shape, fieldName)
			}
			return ast.TypeNode{Ident: ast.TypeShape}, nil
		}
		log.Debugf("[lookupFieldType] Field %s not found in merged fields", fieldName.ID)
	}

	// If the base type is an assertion type, first try to resolve its base type to a shape
	if baseType.Assertion != nil && baseType.Assertion.BaseType != nil {
		log.Debugf("[lookupFieldType] Checking assertion base type: %s", *baseType.Assertion.BaseType)
		// Look for the base type in VariableTypes
		var baseTypeIdent = ast.Identifier(*baseType.Assertion.BaseType)
		if varTypes, exists := tc.VariableTypes[baseTypeIdent]; exists {
			log.Debugf("[lookupFieldType] Found %d types for base type %s: %+v", len(varTypes), *baseType.Assertion.BaseType, varTypes)
			// Look for a shape type in the variable's types
			for _, varType := range varTypes {
				log.Debugf("[lookupFieldType] Checking variable type: %+v", varType)
				if varType.Ident == ast.TypeShape && varType.Assertion != nil {
					log.Debugf("[lookupFieldType] Found shape type with assertion: %+v", varType.Assertion)
					// Extract fields from the shape type's constraints
					for _, constraint := range varType.Assertion.Constraints {
						log.Debugf("[lookupFieldType] Checking constraint: %s with %d args", constraint.Name, len(constraint.Args))
						if constraint.Name == "Match" && len(constraint.Args) > 0 {
							if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
								log.Debugf("[lookupFieldType] Found shape fields in variable type: %+v", shapeArg.Fields)
								if field, exists := shapeArg.Fields[string(fieldName.ID)]; exists {
									log.Debugf("[lookupFieldType] Found field %s with type %+v", fieldName.ID, field)
									if field.Assertion != nil && field.Assertion.BaseType != nil {
										return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
									}
								} else {
									log.Debugf("[lookupFieldType] Field %s not found in shape fields", fieldName.ID)
								}
							}
						}
					}
				}
			}
		}

		// If we didn't find it in VariableTypes, try to resolve the base type's definition
		if def, exists := tc.Defs[*baseType.Assertion.BaseType]; exists {
			log.Debugf("[lookupFieldType] Found type definition for base type %s: %+v", baseTypeIdent, def)
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				log.Debugf("[lookupFieldType] Type definition is TypeDefNode: %+v", typeDef)
				// Try both value and pointer types for TypeDefAssertionExpr
				var typeDefExpr ast.TypeDefAssertionExpr
				if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					typeDefExpr = expr
				} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
					typeDefExpr = *expr
				} else {
					log.Debugf("[lookupFieldType] Type definition expression is not TypeDefAssertionExpr: %T", typeDef.Expr)
					return ast.TypeNode{}, fmt.Errorf("field %s not found in type %s", fieldName, baseType.Ident)
				}

				// Now look for the assertion in the base type's definition
				if typeDefExpr.Assertion != nil {
					log.Debugf("[lookupFieldType] Processing %d constraints from type definition: %+v",
						len(typeDefExpr.Assertion.Constraints), typeDefExpr.Assertion.Constraints)
					// Collect fields from constraints
					mergedFields := MergeShapeFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
					log.Debugf("[lookupFieldType] Merged fields from constraints: %+v", mergedFields)
					if field, exists := mergedFields[string(fieldName.ID)]; exists {
						log.Debugf("[lookupFieldType] Found field %s in merged fields: %+v", fieldName.ID, field)
						if field.Assertion != nil && field.Assertion.BaseType != nil {
							return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
						}
					} else {
						log.Debugf("[lookupFieldType] Field %s not found in merged fields", fieldName.ID)
					}
				}
			}
		}
	}

	// If we didn't find the field in the base type's shape, try looking in the assertion's shape
	if baseType.Assertion != nil {
		log.Debugf("[lookupFieldType] Checking assertion type: %+v", baseType.Assertion)
		// Look for a Match constraint with a shape
		for _, constraint := range baseType.Assertion.Constraints {
			if constraint.Name == "Match" && len(constraint.Args) > 0 {
				if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
					log.Debugf("[lookupFieldType] Found shape in Match constraint: %+v", shapeArg)
					if field, exists := shapeArg.Fields[string(fieldName.ID)]; exists {
						log.Debugf("[lookupFieldType] Found field %s in assertion shape: %+v", fieldName.ID, field)
						if field.Assertion != nil && field.Assertion.BaseType != nil {
							return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
						}
					}
				}
			}
		}
	}

	// First try to get the shape type from VariableTypes
	log.Debugf("[lookupFieldType] Looking up variable type for %s", baseType.Ident)
	if varTypes, exists := tc.VariableTypes[fieldName.ID]; exists {
		log.Debugf("[lookupFieldType] Found %d types for variable %s: %+v", len(varTypes), baseType.Ident, varTypes)
		// Look for a shape type in the variable's types
		for _, varType := range varTypes {
			log.Debugf("[lookupFieldType] Checking variable type: %+v", varType)
			if varType.Ident == ast.TypeShape && varType.Assertion != nil {
				log.Debugf("[lookupFieldType] Found shape type with assertion: %+v", varType.Assertion)
				// Extract fields from the shape type's constraints
				for _, constraint := range varType.Assertion.Constraints {
					log.Debugf("[lookupFieldType] Checking constraint: %s with %d args", constraint.Name, len(constraint.Args))
					if constraint.Name == "Match" && len(constraint.Args) > 0 {
						if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
							log.Debugf("[lookupFieldType] Found shape fields in variable type: %+v", shapeArg.Fields)
							if field, exists := shapeArg.Fields[string(fieldName.ID)]; exists {
								log.Debugf("[lookupFieldType] Found field %s with type %+v", fieldName.ID, field)
								if field.Assertion != nil && field.Assertion.BaseType != nil {
									return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
								}
							} else {
								log.Debugf("[lookupFieldType] Field %s not found in shape fields", fieldName.ID)
							}
						}
					}
				}
			}
		}
	} else {
		log.Debugf("[lookupFieldType] No types found in VariableTypes for %s", fieldName.ID)
	}

	// If we didn't find the field in VariableTypes, try to look it up in the type definition
	def, exists := tc.Defs[baseType.Ident]
	if !exists {
		log.Debugf("[lookupFieldType] No type definition found for %s", baseType.Ident)
		return ast.TypeNode{}, fmt.Errorf("base type %s for field %s not found", baseType.Ident, fieldName.ID)
	}
	log.Debugf("[lookupFieldType] Found type definition for %s: %+v", baseType.Ident, def)

	if typeDef, ok := def.(ast.TypeDefNode); ok {
		log.Debugf("[lookupFieldType] Type definition is TypeDefNode: %+v", typeDef)
		// Try both value and pointer types for TypeDefAssertionExpr
		var typeDefExpr ast.TypeDefAssertionExpr
		if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
			typeDefExpr = expr
		} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
			typeDefExpr = *expr
		} else if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
			// Handle direct shape definition
			log.Debugf("[lookupFieldType] Processing direct shape definition: %+v", shapeExpr.Shape)
			if field, exists := shapeExpr.Shape.Fields[string(fieldName.ID)]; exists {
				log.Debugf("[lookupFieldType] Found field %s in shape: %+v", fieldName.ID, field)
				if field.Assertion != nil && field.Assertion.BaseType != nil {
					return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
				}
			} else {
				log.Debugf("[lookupFieldType] Field %s not found in shape", fieldName.ID)
			}
			return ast.TypeNode{}, fmt.Errorf("field %s not found in shape def type %s", fieldName, baseType.Ident)
		} else {
			log.Debugf("[lookupFieldType] Type definition expression is not TypeDefAssertionExpr or TypeDefShapeExpr: %T", typeDef.Expr)
			return ast.TypeNode{}, fmt.Errorf("field %s not found in unknown type %s", fieldName, baseType.Ident)
		}

		log.Debugf("[lookupFieldType] Type definition expression is TypeDefAssertionExpr: %+v", typeDefExpr)
		if typeDefExpr.Assertion != nil {
			log.Debugf("[lookupFieldType] Processing %d constraints from type definition: %+v",
				len(typeDefExpr.Assertion.Constraints), typeDefExpr.Assertion.Constraints)
			// Collect fields from constraints
			mergedFields := MergeShapeFieldsFromConstraints(typeDefExpr.Assertion.Constraints)
			log.Debugf("[lookupFieldType] Merged fields from constraints: %+v", mergedFields)
			if field, exists := mergedFields[string(fieldName.ID)]; exists {
				log.Debugf("[lookupFieldType] Found field %s in merged fields: %+v", fieldName.ID, field)
				if field.Assertion != nil && field.Assertion.BaseType != nil {
					return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
				}
			} else {
				log.Debugf("[lookupFieldType] Field %s not found in merged fields", fieldName.ID)
			}
		}
	} else if shape, ok := def.(ast.ShapeNode); ok {
		// Handle direct shape node
		log.Debugf("[lookupFieldType] Type definition is ShapeNode: %+v", shape)
		if field, exists := shape.Fields[string(fieldName.ID)]; exists {
			log.Debugf("[lookupFieldType] Found field %s in shape: %+v", fieldName.ID, field)
			if field.Assertion != nil && field.Assertion.BaseType != nil {
				return ast.TypeNode{Ident: *field.Assertion.BaseType}, nil
			}
		} else {
			log.Debugf("[lookupFieldType] Field %s not found in shape", fieldName.ID)
		}
	} else {
		log.Debugf("[lookupFieldType] Type definition is not TypeDefNode or ShapeNode: %T", def)
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
