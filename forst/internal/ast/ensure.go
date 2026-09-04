package ast

import "fmt"

// EnsureNode represents an ensure statement in the AST.
// Typed failure is Error (from `else`); FailureBlock is Block (from `{ … }`).
// Error and Block are mutually exclusive (XOR).
type EnsureNode struct {
	Variable VariableNode
	// Target is the RHS of `is`: TypeTarget (bare type name) or AssertionTarget
	// (constraint chain(s), possibly Join via `or`). Nil for `ensure !err` sugar.
	Target RefinementTarget
	// Assertion is the primary assertion view for AssertionTarget (first Meet chain,
	// with OrChains for Join). For TypeTarget it holds BaseType only (compat).
	Assertion AssertionNode
	/// Is optional if we're in the main function of the main package
	Error *EnsureErrorNode
	// Block is the failure block (alias FailureBlock); runs when the assertion fails.
	Block *EnsureBlockNode
}

// FailureBlock is the ensure failure block (AST name from analyzable-refinements phase 1).
type FailureBlock = EnsureBlockNode

// RefinementTarget is the RHS of `ensure … is` / `if … is`.
type RefinementTarget interface {
	refinementTarget()
	String() string
}

// TypeTarget is a bare type name after `is` (no parentheses).
type TypeTarget struct {
	Name TypeIdent // named type / literal-union domain
}

func (TypeTarget) refinementTarget() {}

func (t TypeTarget) String() string { return string(t.Name) }

// AssertionTarget is one or more constraint chains joined by `or` (Any / Join).
type AssertionTarget struct {
	Chains []AssertionNode // Meet chains; index 0 is primary, rest are Join alts
}

func (AssertionTarget) refinementTarget() {}

func (a AssertionTarget) String() string {
	if len(a.Chains) == 0 {
		return ""
	}
	s := a.Chains[0].String()
	for i := 1; i < len(a.Chains); i++ {
		s += " or " + a.Chains[i].String()
	}
	return s
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
	target := e.Assertion.String()
	if e.Target != nil {
		target = e.Target.String()
	}
	if e.Error == nil {
		return fmt.Sprintf("Ensure(%s, %s)", e.Variable, target)
	}
	return fmt.Sprintf("Ensure(%s, %s, %s)", e.Variable, target, (*e.Error).String())
}

// String returns a string representation of the ensure block
func (e EnsureBlockNode) String() string {
	return "EnsureBlock"
}

// Kind returns the node kind for an ensure block
func (e EnsureBlockNode) Kind() NodeKind {
	return NodeKindEnsureBlock
}
