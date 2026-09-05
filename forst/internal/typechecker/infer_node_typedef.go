package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferNodeTypeDef(n ast.TypeDefNode) ([]ast.TypeNode, bool, error) {
	assertionExpr, ok := n.Expr.(ast.TypeDefAssertionExpr)
	if !ok || assertionExpr.Assertion == nil {
		return nil, true, nil
	}
	tc.log.WithFields(logrus.Fields{
		"ident":    n.Ident,
		"function": "inferNodeType",
	}).Debug("Merging fields for type")

	mergedFields := tc.resolveShapeFieldsFromAssertion(assertionExpr.Assertion)
	tc.log.WithFields(logrus.Fields{
		"ident":        n.Ident,
		"mergedFields": mergedFields,
	}).Debug("Merged fields for type")

	if len(mergedFields) == 0 && assertionExpr.Assertion.BaseType != nil {
		base := *assertionExpr.Assertion.BaseType
		if tc.isBuiltinType(base) {
			return nil, true, nil
		}
		if tc.underlyingBuiltinTypeOfAliasAssertion(base) != "" {
			return nil, true, nil
		}
	}

	if len(assertionExpr.Assertion.Constraints) == 0 && assertionExpr.Assertion.BaseType != nil {
		base := *assertionExpr.Assertion.BaseType
		if !tc.isBuiltinType(base) && tc.underlyingBuiltinTypeOfAliasAssertion(base) == "" {
			if _, ok := tc.Defs[base]; ok {
				return nil, true, nil
			}
		}
	}

	shape := ast.ShapeNode{Fields: mergedFields}
	tc.log.WithFields(logrus.Fields{
		"ident":    n.Ident,
		"shape":    shape,
		"function": "inferNodeType",
	}).Debug("Registering merged shape for type")

	tc.registerShapeType(n.Ident, shape)
	return nil, true, nil
}
