package ast

import (
	"fmt"
)

// ReferenceNode represents a reference to a variable or value in the AST
type ReferenceNode struct {
	ValueNode
	Value ValueNode // The value being referenced (can be a VariableNode or ShapeNode)
}

func (r ReferenceNode) String() string {
	return fmt.Sprintf("Ref(%s)", r.Value.String())
}

// Kind returns the node kind for a reference
func (r ReferenceNode) Kind() NodeKind {
	return NodeKindReference
}

// GetIdent returns the reference identifier as a string
func (r ReferenceNode) GetIdent() string {
	return r.Value.String()
}

// Implement ValueNode interface for ReferenceNode
func (r ReferenceNode) isValue() {}
