package ast

import "fmt"

// EnsureNode represents an ensure statement in the AST
type EnsureNode struct {
	Variable  VariableNode
	Assertion AssertionNode
	/// Is optional if we're in the main function of the main package
	Error *EnsureErrorNode
	Block *EnsureBlockNode
}

// EnsureBlockNode represents a block of statements for an ensure statement
type EnsureBlockNode struct {
	Body []Node
}

// EnsureErrorNode represents an error node for an ensure statement, can be a call or a variable
type EnsureErrorNode interface {
	String() string
}

// EnsureErrorCall represents an error call for an ensure statement
type EnsureErrorCall struct {
	ErrorType string
	ErrorArgs []ExpressionNode
}

func (e EnsureErrorCall) String() string {
	return fmt.Sprintf("%s(%v)", e.ErrorType, e.ErrorArgs)
}

// EnsureErrorVar represents an error variable for an ensure statement
type EnsureErrorVar string

func (e EnsureErrorVar) String() string {
	return string(e)
}

// Kind returns the node kind for an ensure statement
func (e EnsureNode) Kind() NodeKind {
	return NodeKindEnsure
}

func (e EnsureNode) String() string {
	if e.Error == nil {
		return fmt.Sprintf("Ensure(%s, %s)", e.Variable, e.Assertion)
	}
	return fmt.Sprintf("Ensure(%s, %s, %s)", e.Variable, e.Assertion, (*e.Error).String())
}

// String returns a string representation of the ensure block
func (e EnsureBlockNode) String() string {
	return "EnsureBlock"
}

// Kind returns the node kind for an ensure block
func (e EnsureBlockNode) Kind() NodeKind {
	return NodeKindEnsureBlock
}
