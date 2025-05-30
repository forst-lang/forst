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

// Kind returns the node kind for a type guard
func (t TypeGuardNode) Kind() NodeKind {
	return NodeKindTypeGuard
}

// GetIdent returns the identifier for the type guard
func (t TypeGuardNode) GetIdent() string {
	return string(t.Ident)
}

// String returns a string representation of the type guard
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
