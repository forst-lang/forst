package ast

import "fmt"

// Ensure Node
type EnsureNode struct {
	Variable  string
	Assertion AssertionNode
	/// Is optional if we're in the main function of the main package
	Error *EnsureErrorNode
}

// EnsureErrorNode can be either a full error node with type and args,
// or just a variable reference
type EnsureErrorNode interface {
	String() string
}

type EnsureErrorCall struct {
	ErrorType string
	ErrorArgs []ExpressionNode
}

func (e EnsureErrorCall) String() string {
	return fmt.Sprintf("%s(%v)", e.ErrorType, e.ErrorArgs)
}

type EnsureErrorVar string

func (e EnsureErrorVar) String() string {
	return string(e)
}

// NodeType returns the type of this AST node
func (e EnsureNode) NodeType() string {
	return "Ensure"
}

func (e EnsureNode) String() string {
	if e.Error == nil {
		return fmt.Sprintf("Ensure(%s, %s)", e.Variable, e.Assertion)
	}
	return fmt.Sprintf("Ensure(%s, %s, %s)", e.Variable, e.Assertion, (*e.Error).String())
}
