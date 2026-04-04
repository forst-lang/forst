package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// Diagnostic is a type-check error with an optional source span for editors and LSP.
type Diagnostic struct {
	Msg  string
	Span ast.SourceSpan
	Code string
}

func (d *Diagnostic) Error() string {
	if d == nil {
		return ""
	}
	return d.Msg
}

// spanOfExpression returns the best-known span for an expression (parser may omit for some literals).
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

// spanForCallArg prefers ArgSpans[i], then spanOfExpression(args[i]), then callSpan.
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

func diagnosticf(span ast.SourceSpan, code, format string, a ...interface{}) error {
	return &Diagnostic{
		Msg:  fmt.Sprintf(format, a...),
		Span: span,
		Code: code,
	}
}
