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
	switch expr.(type) {
	case ast.BinaryExpressionNode, ast.UnaryExpressionNode:
		types, _, err := tc.inferExpressionBinaryUnary(expr)
		return types, err
	case ast.IntLiteralNode, ast.FloatLiteralNode, ast.StringLiteralNode, ast.RuneLiteralNode,
		ast.BoolLiteralNode, ast.NilLiteralNode, ast.IotaLiteralNode:
		types, _, err := tc.inferExpressionLiterals(expr)
		return types, err
	case ast.ArrayLiteralNode:
		types, _, err := tc.inferExpressionArrayLiteral(expr)
		return types, err
	case ast.MapLiteralNode:
		types, _, err := tc.inferExpressionMapLiteral(expr)
		return types, err
	case ast.VariableNode:
		types, _, err := tc.inferExpressionVariable(expr)
		return types, err
	case ast.IndexExpressionNode:
		types, _, err := tc.inferExpressionIndex(expr)
		return types, err
	case ast.SliceExpressionNode:
		types, _, err := tc.inferExpressionSlice(expr)
		return types, err
	case ast.SpreadExpressionNode:
		types, _, err := tc.inferExpressionSpread(expr)
		return types, err
	case ast.FieldAccessNode:
		types, _, err := tc.inferExpressionFieldAccess(expr)
		return types, err
	case ast.MethodCallNode:
		types, _, err := tc.inferExpressionMethodCall(expr)
		return types, err
	case ast.FunctionCallNode:
		types, _, err := tc.inferExpressionFunctionCall(expr)
		return types, err
	case ast.ShapeNode, ast.AssertionNode:
		types, _, err := tc.inferExpressionShapeAssertion(expr)
		return types, err
	case ast.ReferenceNode, ast.DereferenceNode:
		types, _, err := tc.inferExpressionRefDeref(expr)
		return types, err
	case ast.FunctionLiteralNode, ast.TypeExpressionNode, ast.OkExprNode, ast.ErrExprNode:
		types, _, err := tc.inferExpressionMisc(expr)
		return types, err
	}
	tc.log.Tracef("Unhandled expression type: %T", expr)
	return nil, fmt.Errorf("cannot infer type for expression: %T", expr)
}
