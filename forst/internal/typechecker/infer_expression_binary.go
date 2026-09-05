package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionBinaryUnary(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:

		if e.Operator == ast.TokenArrow {
			// channel <- value
			if _, err := tc.inferExpressionType(e.Left); err != nil {
				return nil, true, err
			}
			if _, err := tc.inferExpressionType(e.Right); err != nil {
				return nil, true, err
			}
			span := ast.SourceSpan{}
			if vn, ok := e.Left.(ast.VariableNode); ok {
				span = vn.Ident.Span
			}
			tc.invalidateAfterChannelSend(e.Right, span)
			tc.storeInferredType(e, []ast.TypeNode{{Ident: ast.TypeVoid}})
			return []ast.TypeNode{{Ident: ast.TypeVoid}}, true, nil
		}
		inferredType, err := tc.unifyTypes(e.Left, e.Right, e.Operator)
		if err != nil {
			return nil, true, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, true, nil
	case ast.UnaryExpressionNode:

		inferredType, err := tc.unifyTypes(e.Operand, nil, e.Operator)
		if err != nil {
			return nil, true, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, true, nil
	}
	return nil, false, nil
}
