package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferExpressionType(expr ast.Node) ([]ast.TypeNode, error) {
	if tc.log.IsLevelEnabled(logrus.DebugLevel) {
		tc.log.WithFields(logrus.Fields{
			"function": "inferExpressionType",
			"expr":     expr,
		}).Debugf("Starting type inference for expression")
	}
	if isLiteralExpression(expr) {
		if cached, ok, err := tc.lookupCachedExpressionTypes(expr); err != nil {
			return nil, err
		} else if ok {
			tc.storeInferredType(expr, cached)
			return cached, nil
		}
	}
	handlers := []func(ast.Node) ([]ast.TypeNode, bool, error){
		tc.inferExpressionBinaryUnary,
		tc.inferExpressionLiterals,
		tc.inferExpressionArrayLiteral,
		tc.inferExpressionMapLiteral,
		tc.inferExpressionVariable,
		tc.inferExpressionIndex,
		tc.inferExpressionSlice,
		tc.inferExpressionSpread,
		tc.inferExpressionFieldAccess,
		tc.inferExpressionMethodCall,
		tc.inferExpressionFunctionCall,
		tc.inferExpressionShapeAssertion,
		tc.inferExpressionRefDeref,
		tc.inferExpressionMisc,
	}
	for _, h := range handlers {
		types, ok, err := h(expr)
		if ok {
			return types, err
		}
	}
	tc.log.Tracef("Unhandled expression type: %T", expr)
	return nil, fmt.Errorf("cannot infer type for expression: %T", expr)
}
