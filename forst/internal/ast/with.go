package ast

import "fmt"

// WithNode wires Usable implementations for a block.
// Wiring is a shape literal `{ Logger: x, ... }` or an expression such as `ciUserApiServices()`.
type WithNode struct {
	Wiring ExpressionNode
	Body   []Node
	Span   SourceSpan
}

func (w WithNode) Kind() NodeKind {
	return NodeKindWith
}

func (w WithNode) String() string {
	return fmt.Sprintf("With(%s)", w.Wiring)
}
