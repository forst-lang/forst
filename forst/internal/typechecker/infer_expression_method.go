package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionMethodCall(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.MethodCallNode:

		argTypes := make([][]ast.TypeNode, 0, len(e.Arguments))
		for _, arg := range e.Arguments {
			ts, err := tc.inferExpressionType(arg)
			if err != nil {
				return nil, true, err
			}
			argTypes = append(argTypes, ts)
		}
		if goRecv := tc.goTypeForExpression(e.Receiver); goRecv != nil {
			fc := ast.FunctionCallNode{Arguments: e.Arguments, CallSpan: e.CallSpan, ArgSpans: e.ArgSpans}
			ret, err := tc.checkGoMethodCall(goRecv, e.Method, fc, argTypes, true)
			if err != nil {
				return nil, true, err
			}
			span := e.CallSpan
			if !span.IsSet() {
				span = e.Method.Span
			}
			tc.invalidateReachableMutableArg(e.Receiver, span, dropByForeign)
			tc.storeInferredType(e, ret)
			return ret, true, nil
		}
		if vn, ok := e.Receiver.(ast.VariableNode); ok {
			recvTypes, err := tc.inferExpressionType(e.Receiver)
			if err != nil {
				return nil, true, err
			}
			fc := ast.FunctionCallNode{Function: ast.Ident{ID: ast.Identifier(string(vn.Ident.ID) + "." + string(e.Method.ID))}, Arguments: e.Arguments, CallSpan: e.CallSpan, ArgSpans: e.ArgSpans}
			ret, err := tc.inferMethodCallType(vn.Ident.ID, recvTypes, string(e.Method.ID), fc, argTypes)
			if err != nil {
				return nil, true, err
			}
			tc.storeInferredType(e, ret)
			return ret, true, nil
		}
		sp := e.Method.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, true, reportBodyf(sp, "undefined-identifier", "method %s on receiver type %T", e.Method.ID, e.Receiver)
	}
	return nil, false, nil
}
