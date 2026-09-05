package ast

// ExpressionSpanStart returns the best-known start span for an expression node
// already carrying location (literals, idents, calls, nested index/slice/shape/etc.).
func ExpressionSpanStart(expr ExpressionNode) SourceSpan {
	if expr == nil {
		return SourceSpan{}
	}
	switch e := expr.(type) {
	case IntLiteralNode:
		return e.Span
	case FloatLiteralNode:
		return e.Span
	case StringLiteralNode:
		return e.Span
	case RuneLiteralNode:
		return e.Span
	case BoolLiteralNode:
		return e.Span
	case ArrayLiteralNode:
		return e.Span
	case MapLiteralNode:
		return e.Span
	case NilLiteralNode:
		return e.Span
	case IotaLiteralNode:
		return e.Span
	case VariableNode:
		return e.Ident.Span
	case FunctionCallNode:
		if e.CallSpan.IsSet() {
			return e.CallSpan
		}
		return e.Function.Span
	case MethodCallNode:
		if e.CallSpan.IsSet() {
			return e.CallSpan
		}
		if e.Method.Span.IsSet() {
			return e.Method.Span
		}
		return ExpressionSpanStart(e.Receiver)
	case FieldAccessNode:
		return ExpressionSpanStart(e.Target)
	case IndexExpressionNode:
		if e.Span.IsSet() {
			return e.Span
		}
		return ExpressionSpanStart(e.Target)
	case SliceExpressionNode:
		if e.Span.IsSet() {
			return e.Span
		}
		return ExpressionSpanStart(e.Target)
	case ShapeNode:
		return e.Span
	case FunctionLiteralNode:
		return e.Span
	case TypeExpressionNode:
		return e.Span
	case OkExprNode:
		if e.Span.IsSet() {
			return e.Span
		}
		return ExpressionSpanStart(e.Value)
	case ErrExprNode:
		if e.Span.IsSet() {
			return e.Span
		}
		return ExpressionSpanStart(e.Value)
	case UnaryExpressionNode:
		return ExpressionSpanStart(e.Operand)
	case BinaryExpressionNode:
		return ExpressionSpanStart(e.Left)
	case SpreadExpressionNode:
		return ExpressionSpanStart(e.Expr)
	case ReferenceNode:
		if e.Value != nil {
			return ExpressionSpanStart(e.Value)
		}
	case DereferenceNode:
		if e.Value != nil {
			return ExpressionSpanStart(e.Value)
		}
	}
	return SourceSpan{}
}
