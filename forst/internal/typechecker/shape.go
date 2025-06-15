package typechecker

import (
	"forst/internal/ast"
)

// resolveShapeFieldsFromAssertion recursively merges all shape fields from an assertion chain
func (tc *TypeChecker) resolveShapeFieldsFromAssertion(assertion *ast.AssertionNode) map[string]ast.ShapeFieldNode {
	merged := make(map[string]ast.ShapeFieldNode)
	if assertion == nil {
		tc.log.Debugf("[resolveShapeFieldsFromAssertion] Assertion is nil, returning empty map")
		return merged
	}

	tc.log.Debugf("[resolveShapeFieldsFromAssertion] Starting with assertion: %+v", assertion)
	tc.log.Debugf("[resolveShapeFieldsFromAssertion] BaseType: %v, Constraints: %+v", assertion.BaseType, assertion.Constraints)

	// First, handle the base type if it exists
	if assertion.BaseType != nil {
		tc.log.Debugf("[resolveShapeFieldsFromAssertion] Processing base type: %s", *assertion.BaseType)
		if def, exists := tc.Defs[*assertion.BaseType]; exists {
			tc.log.Debugf("[resolveShapeFieldsFromAssertion] Found base type definition: %T %+v", def, def)
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				tc.log.Debugf("[resolveShapeFieldsFromAssertion] Base type is TypeDefNode: %s, Expr: %T %+v",
					typeDef.Ident, typeDef.Expr, typeDef.Expr)

				// Handle both value and pointer types for TypeDefAssertionExpr
				var baseAssertionExpr ast.TypeDefAssertionExpr
				if expr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					baseAssertionExpr = expr
				} else if expr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
					baseAssertionExpr = *expr
				}

				if baseAssertionExpr.Assertion != nil {
					tc.log.Debugf("[resolveShapeFieldsFromAssertion] Recursively merging fields from base assertion: %+v",
						baseAssertionExpr.Assertion)
					baseFields := tc.resolveShapeFieldsFromAssertion(baseAssertionExpr.Assertion)
					tc.log.Debugf("[resolveShapeFieldsFromAssertion] Got base fields: %+v", baseFields)
					for k, v := range baseFields {
						merged[k] = v
						tc.log.Debugf("[resolveShapeFieldsFromAssertion] Added field from base: %s => %+v", k, v)
					}
				} else if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					// If the base type is a shape, add its fields directly
					for k, v := range shapeExpr.Shape.Fields {
						merged[k] = v
						tc.log.Debugf("[resolveShapeFieldsFromAssertion] Added field from base shape: %s => %+v", k, v)
					}
				}
			}
		}
	}

	// Then process each constraint
	for i, constraint := range assertion.Constraints {
		tc.log.Debugf("[resolveShapeFieldsFromAssertion] Processing constraint %d: %s with args: %+v", i, constraint.Name, constraint.Args)

		// Get the type guard definition
		guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]
		if !exists {
			tc.log.Debugf("[resolveShapeFieldsFromAssertion] Type guard %s not found", constraint.Name)
			continue
		}

		// Push a new scope for the type guard's body
		var guardNode ast.TypeGuardNode
		if ptr, ok := guardDef.(*ast.TypeGuardNode); ok {
			guardNode = *ptr
		} else {
			guardNode = guardDef.(ast.TypeGuardNode)
		}

		tc.log.Debugf("[resolveShapeFieldsFromAssertion] Type guard node: %+v", guardNode)
		tc.log.Debugf("[resolveShapeFieldsFromAssertion] Type guard parameters: %+v", guardNode.Parameters())

		// Map constraint arguments to parameters
		argMap := make(map[string]ast.Node)
		for i, arg := range constraint.Args {
			if i+1 < len(guardNode.Parameters()) {
				param := guardNode.Parameters()[i+1] // +1 to skip subject parameter
				argMap[param.GetIdent()] = arg
				tc.log.Debugf("[resolveShapeFieldsFromAssertion] Mapping parameter %s to argument %+v", param.GetIdent(), arg)
			}
		}

		// For each parameter in the type guard (except the subject), merge fields from argument shapes
		for _, param := range guardNode.Parameters() {
			if param.GetIdent() != guardNode.Subject.GetIdent() {
				// Add the field from the parameter
				paramType := param.GetType().Ident
				merged[param.GetIdent()] = ast.ShapeFieldNode{
					Assertion: &ast.AssertionNode{
						BaseType: &paramType,
					},
				}

				// If we have an argument for this parameter, use its concrete type or merge its shape fields
				if arg, ok := argMap[param.GetIdent()]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							// Merge all fields from the argument shape
							for k, v := range argNode.Shape.Fields {
								merged[k] = v
								tc.log.Debugf("[resolveShapeFieldsFromAssertion] Added field from shape argument: %s => %+v", k, v)
							}
						} else if argNode.Type != nil {
							// Use the concrete type from the argument instead of the generic Shape
							argType := argNode.Type.Ident
							merged[param.GetIdent()] = ast.ShapeFieldNode{
								Assertion: &ast.AssertionNode{
									BaseType: &argType,
								},
							}
							tc.log.Debugf("[resolveShapeFieldsFromAssertion] Added field with concrete type: %s => %s", param.GetIdent(), argNode.Type.Ident)
						}
					}
				}
			}
		}
	}

	tc.log.Debugf("[resolveShapeFieldsFromAssertion] Final merged fields: %+v", merged)
	return merged
}
