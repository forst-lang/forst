package ast

import (
	"fmt"
)

// VariableNode represents a variable in the AST
type VariableNode struct {
	ValueNode
	Ident        Ident
	ExplicitType TypeNode
}

func (v VariableNode) String() string {
	return fmt.Sprintf("Variable(%s)", v.Ident.ID)
}

// Kind returns the node kind for a variable
func (v VariableNode) Kind() NodeKind {
	return NodeKindVariable
}

// GetIdent returns the variable identifier as a string
func (v VariableNode) GetIdent() string {
	return string(v.Ident.ID)
}

// Implement ValueNode interface for VariableNode
func (v VariableNode) isValue() {}
