package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"time"

	log "github.com/sirupsen/logrus"
)

// TODO: Improve assertion type inference
// This should handle:
// 1. Complex constraints
// 2. Nested assertions
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) inferAssertionType(assertion *ast.AssertionNode, isTypeGuard bool) ([]ast.TypeNode, error) {
	if assertion == nil {
		return nil, nil
	}

	log.Debugf("[inferAssertionType] Inferring type for assertion: %+v", assertion)

	// First, get the base type if it exists
	var baseType ast.TypeIdent
	if assertion.BaseType != nil {
		baseType = *assertion.BaseType
		log.Debugf("[inferAssertionType] Base type: %s", baseType)
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
						log.Debugf("[inferAssertionType] Added field from base type: %s => %+v", k, v)
					}
				} else if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					if assertionExpr.Assertion != nil {
						baseFields := tc.resolveMergedShapeFields(assertionExpr.Assertion)
						for k, v := range baseFields {
							mergedFields[k] = v
							log.Debugf("[inferAssertionType] Added field from base assertion: %s => %+v", k, v)
						}
					}
				}
			}
		}
	}

	// Process each constraint
	for _, constraint := range assertion.Constraints {
		log.Debugf("[inferAssertionType] Processing constraint: %s", constraint.Name)

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

		log.Debugf("[inferAssertionType] Subject parameter: %s: %s", guardNode.Subject.GetIdent(), guardNode.Subject.GetType().Ident)
		log.Debugf("[inferAssertionType] Additional parameters: %+v", guardNode.Parameters())

		// Map constraint arguments to parameters
		argMap := make(map[string]ast.Node)
		for i, arg := range constraint.Args {
			if i+1 < len(guardNode.Parameters()) {
				param := guardNode.Parameters()[i+1] // +1 to skip subject parameter
				argMap[param.GetIdent()] = arg
				log.Debugf("[inferAssertionType] Mapping parameter %s to argument %+v", param.GetIdent(), arg)
			}
		}

		// Special handling for mutation types
		if constraint.Name == "Input" {
			// For Input constraint, we want to keep existing fields and add new input fields
			// First, preserve the ctx field if it exists
			if ctxField, hasCtx := mergedFields["ctx"]; hasCtx {
				// Create a temporary map to store fields
				tempFields := make(map[string]ast.ShapeFieldNode)
				tempFields["ctx"] = ctxField

				// Add the input field from the argument
				if arg, ok := argMap["input"]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							for k, v := range argNode.Shape.Fields {
								tempFields[k] = v
								log.Debugf("[inferAssertionType] Added field from mutation input: %s => %+v", k, v)
							}
						}
					}
				}

				// Replace mergedFields with our new map that preserves ctx
				mergedFields = tempFields
			} else {
				// If no ctx field exists, just add the input fields
				if arg, ok := argMap["input"]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							for k, v := range argNode.Shape.Fields {
								mergedFields[k] = v
								log.Debugf("[inferAssertionType] Added field from mutation input: %s => %+v", k, v)
							}
						}
					}
				}
			}
			continue
		}

		// For other constraints, process each parameter
		for _, param := range guardNode.Parameters() {
			if param.GetIdent() != guardNode.Subject.GetIdent() {
				// Add the field from the parameter
				paramType := param.GetType().Ident
				mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
					Assertion: &ast.AssertionNode{
						BaseType: &paramType,
					},
				}

				// If we have an argument for this parameter, use its concrete type
				if arg, ok := argMap[param.GetIdent()]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							// If it's a shape, merge its fields
							for k, v := range argNode.Shape.Fields {
								mergedFields[k] = v
								log.Debugf("[inferAssertionType] Added field from shape argument: %s => %+v", k, v)
							}
						} else if argNode.Type != nil {
							// Use the concrete type from the argument instead of the generic Shape
							argType := argNode.Type.Ident
							mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
								Assertion: &ast.AssertionNode{
									BaseType: &argType,
								},
							}
							log.Debugf("[inferAssertionType] Added field with concrete type: %s => %s", param.GetIdent(), argNode.Type.Ident)
						}
					}
				}
			}
		}
	}

	// Create a unique type identifier for this shape
	typeIdent := ast.TypeIdent(fmt.Sprintf("T_%s", generateUniqueID()))
	log.Debugf("[inferAssertionType] Stored shape type with fields assertion=%+v fields=%+v typeIdent=%s",
		assertion, mergedFields, typeIdent)

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
	return []ast.TypeNode{shapeType}, nil
}

func generateUniqueID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
