package ast

// Ensure Node
type EnsureNode struct {
	Condition string
	ErrorType string
} 

// NodeType returns the type of this AST node
func (e EnsureNode) NodeType() string {
	return "Ensure"
} 