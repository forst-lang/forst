package ast

import "fmt"

// ForNode is a for-loop: either a classic C-style loop or a range loop.
// Classic: IsRange false, optional Init / Cond / Post, Body.
// Range: IsRange true, RangeX required; RangeKey/RangeValue optional (nil means omitted, as in `for range ch`).
// Use Ident ID "_" for blank identifiers.
type ForNode struct {
	Init  Node
	Cond  ExpressionNode
	Post  Node

	IsRange    bool
	RangeX     ExpressionNode
	RangeKey   *Ident
	RangeValue *Ident
	RangeShort bool // := vs =

	Body []Node
}

func (f ForNode) Kind() NodeKind {
	return NodeKindFor
}

func (f ForNode) String() string {
	if f.IsRange {
		return fmt.Sprintf("For(range %v)", f.RangeX)
	}
	return "For(...)"
}

// BreakNode is a break statement. Label is reserved for future labeled breaks.
type BreakNode struct {
	Label *Ident
}

func (b BreakNode) Kind() NodeKind {
	return NodeKindBreak
}

func (b BreakNode) String() string {
	if b.Label != nil {
		return fmt.Sprintf("Break(%s)", b.Label.ID)
	}
	return "Break"
}

// ContinueNode is a continue statement. Label is reserved for future labeled continue.
type ContinueNode struct {
	Label *Ident
}

func (c ContinueNode) Kind() NodeKind {
	return NodeKindContinue
}

func (c ContinueNode) String() string {
	if c.Label != nil {
		return fmt.Sprintf("Continue(%s)", c.Label.ID)
	}
	return "Continue"
}

// DeferNode is a defer statement: `defer <call>` (expression must be a function or method call).
type DeferNode struct {
	Call ExpressionNode
}

func (d DeferNode) Kind() NodeKind {
	return NodeKindDefer
}

func (d DeferNode) String() string {
	return "Defer(...)"
}

// GoStmtNode is a go statement: `go <call>` (starts a goroutine; expression must be a function or method call).
type GoStmtNode struct {
	Call ExpressionNode
}

func (g GoStmtNode) Kind() NodeKind {
	return NodeKindGoStmt
}

func (g GoStmtNode) String() string {
	return "Go(...)"
}
