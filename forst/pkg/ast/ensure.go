package ast

import "fmt"

// Ensure Node
type EnsureNode struct {
	Variable  string
	Assertion AssertionNode
	/// Is optional if we're in the main function of the main package
	ErrorType *string
	ErrorArgs []ExpressionNode
}

// NodeType returns the type of this AST node
func (e EnsureNode) NodeType() string {
	return "Ensure"
}

func (e EnsureNode) String() string {
	if e.ErrorType == nil {
		return fmt.Sprintf("Ensure(%s, %s)", e.Variable, e.Assertion)
	}
	return fmt.Sprintf("Ensure(%s, %s, %s)", e.Variable, e.Assertion, *e.ErrorType)
}
