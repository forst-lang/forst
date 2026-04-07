package ast

import "fmt"

// OkExprNode is the language construct Ok(expr) for Result success values.
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

// ErrExprNode is the language construct Err(expr) for Result failure values.
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
