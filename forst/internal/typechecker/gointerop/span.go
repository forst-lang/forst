package gointerop

import "forst/internal/ast"

func spanOfExpression(expr ast.ExpressionNode) ast.SourceSpan {
	switch e := expr.(type) {
	case ast.FunctionCallNode:
		if e.CallSpan.IsSet() {
			return e.CallSpan
		}
		if e.Function.Span.IsSet() {
			return e.Function.Span
		}
	case ast.VariableNode:
		if e.Ident.Span.IsSet() {
			return e.Ident.Span
		}
	}
	return ast.SourceSpan{}
}

func spanForCallArg(argSpans []ast.SourceSpan, i int, args []ast.ExpressionNode, callSpan ast.SourceSpan) ast.SourceSpan {
	if i < len(argSpans) && argSpans[i].IsSet() {
		return argSpans[i]
	}
	if i < len(args) {
		if s := spanOfExpression(args[i]); s.IsSet() {
			return s
		}
	}
	if callSpan.IsSet() {
		return callSpan
	}
	return ast.SourceSpan{}
}
