package ast

import (
	"fmt"
)

// TypeGuardNode represents a type guard declaration
type TypeGuardNode struct {
	Node
	// Name of the type guard
	Ident Identifier
	// Parameters of the type guard
	Parameters []ParamNode
	// Body of the type guard
	Body []Node
}

func (t TypeGuardNode) Kind() NodeKind {
	return NodeKindTypeGuard
}

func (t TypeGuardNode) Id() string {
	return string(t.Ident)
}

func (t TypeGuardNode) String() string {
	return fmt.Sprintf("TypeGuardNode(%s)", t.Ident)
}
