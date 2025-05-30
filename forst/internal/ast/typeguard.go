package ast

import (
	"fmt"
)

// TypeGuardNode represents a type guard declaration
type TypeGuardNode struct {
	Node
	// Name of the type guard
	Ident Identifier
	// Subject parameter - the value being validated
	SubjectParam ParamNode
	// Additional parameters used in validation
	AdditionalParams []ParamNode
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

// Parameters returns all parameters in order (subject first, then additional)
func (t TypeGuardNode) Parameters() []ParamNode {
	params := make([]ParamNode, 0, 1+len(t.AdditionalParams))
	params = append(params, t.SubjectParam)
	params = append(params, t.AdditionalParams...)
	return params
}
