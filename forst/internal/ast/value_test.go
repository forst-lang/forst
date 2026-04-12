package ast

import "testing"

func TestValueNode_literals_satisfy_interface(_ *testing.T) {
	var _ ValueNode = IntLiteralNode{Value: 1}
	var _ ValueNode = StringLiteralNode{Value: "x"}
	var _ ValueNode = VariableNode{Ident: Ident{ID: "v"}}
}
