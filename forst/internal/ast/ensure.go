package ast

import "fmt"

// Represents an ensure statement in the AST
type EnsureNode struct {
	Variable  VariableNode
	Assertion AssertionNode
	/// Is optional if we're in the main function of the main package
	Error *EnsureErrorNode
	Block *EnsureBlockNode
}

// Block of statements for ensure
type EnsureBlockNode struct {
	Body []Node
}

// Error node for ensure, can be a call or a variable
type EnsureErrorNode interface {
	String() string
}

// Error call for ensure
type EnsureErrorCall struct {
	ErrorType string
	ErrorArgs []ExpressionNode
}

func (e EnsureErrorCall) String() string {
	return fmt.Sprintf("%s(%v)", e.ErrorType, e.ErrorArgs)
}

// Error variable for ensure
type EnsureErrorVar string

func (e EnsureErrorVar) String() string {
	return string(e)
}

func (e EnsureNode) Kind() NodeKind {
	return NodeKindEnsure
}

func (e EnsureNode) String() string {
	if e.Error == nil {
		return fmt.Sprintf("Ensure(%s, %s)", e.Variable, e.Assertion)
	}
	return fmt.Sprintf("Ensure(%s, %s, %s)", e.Variable, e.Assertion, (*e.Error).String())
}

func (e EnsureBlockNode) String() string {
	return "EnsureBlock"
}

func (e EnsureBlockNode) Kind() NodeKind {
	return NodeKindEnsureBlock
}
