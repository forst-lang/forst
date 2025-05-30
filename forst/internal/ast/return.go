package ast

import "fmt"

// Return Node
type ReturnNode struct {
	Value ExpressionNode
	Type  TypeNode
}

func (r ReturnNode) Kind() NodeKind {
	return NodeKindReturn
}

func (r ReturnNode) String() string {
	return fmt.Sprintf("Return(%s)", r.Value.String())
}
