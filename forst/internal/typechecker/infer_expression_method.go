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
		recvTypes, err := tc.inferExpressionType(e.Receiver)
		if err != nil {
			return nil, true, err
		}
		recvID := ast.Identifier("")
		fnID := ast.Identifier(string(e.Method.ID))
		if vn, ok := e.Receiver.(ast.VariableNode); ok {
			recvID = vn.Ident.ID
			fnID = ast.Identifier(string(vn.Ident.ID) + "." + string(e.Method.ID))
		}
		fc := ast.FunctionCallNode{
			Function:  ast.Ident{ID: fnID, Span: e.Method.Span},
			Arguments: e.Arguments,
			CallSpan:  e.CallSpan,
			ArgSpans:  e.ArgSpans,
		}
		ret, err := tc.inferMethodCallType(recvID, recvTypes, string(e.Method.ID), fc, argTypes)
		if err != nil {
			return nil, true, err
		}
		tc.storeInferredType(e, ret)
		return ret, true, nil
	}
	return nil, false, nil
}
