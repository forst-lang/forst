package ast

import "fmt"

// SwitchNode is a Go-faithful switch statement (tag or boolean form).
type SwitchNode struct {
	Init    Node           // optional init before tag (switch init; tag { ... })
	Tag     ExpressionNode // nil for boolean switch (switch { case cond: ... })
	Clauses []SwitchClauseNode
}

// SwitchClauseNode is one case or default clause in a switch.
type SwitchClauseNode struct {
	IsDefault bool
	Values    []ExpressionNode // case literals/expressions; empty when IsDefault
	Body      []Node
}

// FallthroughNode is an explicit fallthrough in a switch case body.
type FallthroughNode struct{}

func (s SwitchNode) Kind() NodeKind { return NodeKindSwitch }

func (s SwitchNode) String() string {
	if s.Tag != nil {
		return fmt.Sprintf("Switch(%v)", s.Tag)
	}
	return "Switch"
}

func (c SwitchClauseNode) Kind() NodeKind {
	if c.IsDefault {
		return NodeKindDefault
	}
	return NodeKindCase
}

func (c SwitchClauseNode) String() string {
	if c.IsDefault {
		return "Default"
	}
	return fmt.Sprintf("Case(%d)", len(c.Values))
}

func (FallthroughNode) Kind() NodeKind { return NodeKindFallthrough }

func (FallthroughNode) String() string { return "Fallthrough" }
