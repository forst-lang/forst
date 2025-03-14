package ast

// Return Node
type ReturnNode struct {
	Value string
	Type  TypeNode
}

// NodeType returns the type of this AST node
func (r ReturnNode) NodeType() string {
	return "Return"
} 