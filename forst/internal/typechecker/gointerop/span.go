package gointerop

import "forst/internal/ast"

func spanOfExpression(expr ast.ExpressionNode) ast.SourceSpan {
	if expr == nil {
		return ast.SourceSpan{}
	}
	switch e := expr.(type) {
	case ast.FunctionCallNode:
		if e.CallSpan.IsSet() {
			return e.CallSpan
		}
		if e.Function.Span.IsSet() {
			return e.Function.Span
		}
		for _, arg := range e.Arguments {
			if s := spanOfExpression(arg); s.IsSet() {
				return s
			}
		}
	case ast.VariableNode:
		if e.Ident.Span.IsSet() {
			return e.Ident.Span
		}
	case ast.IntLiteralNode:
		return e.Span
	case ast.FloatLiteralNode:
		return e.Span
	case ast.StringLiteralNode:
		return e.Span
	case ast.RuneLiteralNode:
		return e.Span
	case ast.BoolLiteralNode:
		return e.Span
	case ast.ArrayLiteralNode:
		if e.Span.IsSet() {
			return e.Span
		}
		for _, el := range e.Value {
			if s := spanOfExpression(el); s.IsSet() {
				return s
			}
		}
	case ast.NilLiteralNode:
		return e.Span
	case ast.UnaryExpressionNode:
		return spanOfExpression(e.Operand)
	case ast.BinaryExpressionNode:
		if s := spanOfExpression(e.Left); s.IsSet() {
			return s
		}
		return spanOfExpression(e.Right)
	case ast.IndexExpressionNode:
		if s := spanOfExpression(e.Target); s.IsSet() {
			return s
		}
		return spanOfExpression(e.Index)
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
