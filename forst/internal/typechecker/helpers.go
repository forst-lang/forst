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
