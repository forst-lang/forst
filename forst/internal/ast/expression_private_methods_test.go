package ast

import "testing"

func TestExpressionMarkerMethods_andTypeDefErrorExprString(t *testing.T) {
	t.Parallel()
	u := UnaryExpressionNode{}
	b := BinaryExpressionNode{}
	f := FunctionCallNode{}
	i := IndexExpressionNode{}
	okExpr := OkExprNode{}
	errExpr := ErrExprNode{}
	// Call private marker methods to ensure they are exercised.
	u.isExpression()
	b.isExpression()
	f.isExpression()
	i.isExpression()
	okExpr.isExpression()
	errExpr.isExpression()

	td := TypeDefErrorExpr{Payload: ShapeNode{Fields: map[string]ShapeFieldNode{}}}
	if td.String() == "" {
		t.Fatal("expected TypeDefErrorExpr.String() output")
	}
}

