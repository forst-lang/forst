package typechecker

import (
	"forst/internal/ast"
	"maps"
)

// MergeShapeFieldsFromConstraints merges all fields from all shapes in all constraints.
// If multiple constraints define the same field, the last one wins.
func MergeShapeFieldsFromConstraints(constraints []ast.ConstraintNode) map[string]ast.ShapeFieldNode {
	merged := make(map[string]ast.ShapeFieldNode)
	for _, constraint := range constraints {
		for _, arg := range constraint.Args {
			if arg.Shape != nil {
				maps.Copy(merged, arg.Shape.Fields)
			}
		}
	}
	return merged
}

// shapeFromShapeFieldNode returns an inline nested shape for a shape-type field, from either
// Shape or assertion constraints (Match / legacy "Shape" constraint name).
func shapeFromShapeFieldNode(f ast.ShapeFieldNode) *ast.ShapeNode {
	if f.Shape != nil {
		return f.Shape
	}
	if f.Type != nil {
		return shapeFromTypeAssertion(f.Type.Assertion)
	}
	return nil
}

// shapeFromTypeAssertion extracts an inline shape from assertion constraint args.
func shapeFromTypeAssertion(assertion *ast.AssertionNode) *ast.ShapeNode {
	if assertion == nil {
		return nil
	}
	for _, c := range assertion.Constraints {
		for _, arg := range c.Args {
			if arg.Shape != nil {
				return arg.Shape
			}
		}
	}
	return nil
}

// resolveMatchConstraintFieldType infers concrete types for nested shapes and Array inline
// element shapes when merging Match constraint fields.
func (tc *TypeChecker) resolveMatchConstraintFieldType(v ast.ShapeFieldNode) (ast.ShapeFieldNode, bool) {
	if nestedShape := shapeFromShapeFieldNode(v); nestedShape != nil {
		shapeCopy := *nestedShape
		shapeCopy.BaseType = nil
		nestedType, err := tc.inferShapeType(shapeCopy, nil)
		if err != nil {
			return ast.ShapeFieldNode{}, false
		}
		return ast.ShapeFieldNode{
			Type:  &ast.TypeNode{Ident: nestedType.Ident},
			Shape: nestedShape,
		}, true
	}
	if v.Type != nil && v.Type.Ident == ast.TypeArray && len(v.Type.TypeParams) == 1 {
		elem := v.Type.TypeParams[0]
		if elem.Assertion != nil {
			inferred, err := tc.InferAssertionType(elem.Assertion, false, "", nil)
			if err == nil && len(inferred) == 1 {
				return ast.ShapeFieldNode{
					Type: &ast.TypeNode{
						Ident:      ast.TypeArray,
						TypeParams: inferred,
					},
				}, true
			}
		}
	}
	return ast.ShapeFieldNode{}, false
}
