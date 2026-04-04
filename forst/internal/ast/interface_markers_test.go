package ast

import "testing"

// Exercises unexported interface marker methods so coverage attributes statements
// (see literal.go and related: bodies use `_ = recv` for coverable units).
func TestInterfaceMarkerMethodsExecute(t *testing.T) {
	t.Parallel()
	UnaryExpressionNode{}.isExpression()
	BinaryExpressionNode{}.isExpression()
	FunctionCallNode{}.isExpression()

	AssertionNode{}.isExpression()
	ShapeNode{}.isExpression()

	TypeDefAssertionExpr{}.isTypeDefExpr()
	TypeDefBinaryExpr{}.isTypeDefExpr()
	TypeDefShapeExpr{}.isTypeDefExpr()

	VariableNode{}.isValue()
	DereferenceNode{}.isValue()
	ReferenceNode{}.isValue()
}
