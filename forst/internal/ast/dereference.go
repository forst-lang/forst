package ast

import (
	"fmt"
)

// DereferenceNode represents a dereference of a variable or value in the AST
type DereferenceNode struct {
	ValueNode
	Value ValueNode // The value being referenced (can be a VariableNode or ShapeNode)
}

func (d DereferenceNode) String() string {
	return fmt.Sprintf("Deref(%s)", d.Value.String())
}

// Kind returns the node kind for a dereference
func (d DereferenceNode) Kind() NodeKind {
	return NodeKindDereference
}

// GetIdent returns the dereference identifier as a string
func (d DereferenceNode) GetIdent() string {
	return d.Value.String()
}

// Implement ValueNode interface for DereferenceNode
func (r DereferenceNode) isValue() {}
