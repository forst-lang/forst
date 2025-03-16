package ast

import "fmt"

// Return Node
type ReturnNode struct {
	Value ExpressionNode
	Type  TypeNode
}

// NodeType returns the type of this AST node
func (r ReturnNode) NodeType() string {
	return "Return"
}

func (r ReturnNode) String() string {
	return fmt.Sprintf("Return(%s)", r.Value.String())
}
