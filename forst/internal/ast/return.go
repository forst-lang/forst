package ast

import "fmt"

// ReturnNode represents a return statement
type ReturnNode struct {
	Value ExpressionNode
	Type  TypeNode
}

// Kind returns the node kind for a return statement
func (r ReturnNode) Kind() NodeKind {
	return NodeKindReturn
}

// String returns a string representation of the return statement
func (r ReturnNode) String() string {
	return fmt.Sprintf("Return(%s)", r.Value.String())
}
