package ast

import "fmt"

// OkExprNode is the AST for a parsed Ok(expr) call. The typechecker rejects it as a value
// constructor; Ok remains a built-in discriminant for is/ensure on Result.
type OkExprNode struct {
	Value ExpressionNode
}

func (OkExprNode) isExpression() {}

func (OkExprNode) Kind() NodeKind { return NodeKindOkExpr }

func (o OkExprNode) String() string {
	if o.Value == nil {
		return "Ok()"
	}
	return fmt.Sprintf("Ok(%s)", o.Value.String())
}

// ErrExprNode is the AST for a parsed Err(expr) call. The typechecker rejects it as a value
// constructor; Err remains a built-in discriminant for is/ensure on Result.
type ErrExprNode struct {
	Value ExpressionNode
}

func (ErrExprNode) isExpression() {}

func (ErrExprNode) Kind() NodeKind { return NodeKindErrExpr }

func (e ErrExprNode) String() string {
	if e.Value == nil {
		return "Err()"
	}
	return fmt.Sprintf("Err(%s)", e.Value.String())
}

func (t TypeNode) IsResultType() bool {
	return t.Ident == TypeResult && len(t.TypeParams) >= 2
}

func (t TypeNode) IsTupleType() bool {
	return t.Ident == TypeTuple && len(t.TypeParams) >= 1
}
