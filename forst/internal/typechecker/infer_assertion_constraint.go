package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) mergeAssertionMatchConstraint(constraint ast.ConstraintNode, mergedFields *map[string]ast.ShapeFieldNode) {
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

			for k, v := range arg.Shape.Fields {
				if resolved, ok := tc.resolveMatchConstraintFieldType(v); ok {
					(*mergedFields)[k] = resolved
					tc.log.WithFields(logrus.Fields{
						"function": "inferAssertionType",
						"field":    k,
						"type":     resolved.Type.Ident,
					}).Debugf("Added resolved field from Match constraint shape")
				} else if v.Type != nil {
					(*mergedFields)[k] = v
					tc.log.WithFields(logrus.Fields{
						"function": "inferAssertionType",
						"field":    k,
						"type":     v.Type.Ident,
					}).Debugf("Added field from Match constraint shape with existing type")
				} else {
					(*mergedFields)[k] = v
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
				"mergedFields": *mergedFields,
			}).Debugf("After merging Match constraint shape fields")
			continue
		}
	}
}
