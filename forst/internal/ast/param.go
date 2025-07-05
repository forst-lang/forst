package ast

import "fmt"

// ParamNode is the interface for parameter nodes
type ParamNode interface {
	Node
	GetIdent() string
	GetType() TypeNode
}

// SimpleParamNode represents a basic named parameter
type SimpleParamNode struct {
	ParamNode
	Ident Ident    // Parameter name
	Type  TypeNode // Parameter type
}

// DestructuredParamNode represents a parameter that destructures an object
type DestructuredParamNode struct {
	ParamNode
	Fields []string // Field names to destructure
	Type   TypeNode // Parameter type
}

// Kind returns the node kind for a simple parameter
func (p SimpleParamNode) Kind() NodeKind {
	return NodeKindSimpleParam
}

func (p SimpleParamNode) String() string {
	return fmt.Sprintf("%s: %s", p.Ident.ID, p.Type.String())
}

// GetIdent returns the identifier for the simple parameter
func (p SimpleParamNode) GetIdent() string {
	return string(p.Ident.ID)
}

func (p DestructuredParamNode) String() string {
	return fmt.Sprintf("{%v}: %s", p.Fields, p.Type.String())
}

// Kind returns the node kind for a destructured parameter
func (p DestructuredParamNode) Kind() NodeKind {
	return NodeKindDestructuredParam
}

// GetType returns the type of the parameter
func (p SimpleParamNode) GetType() TypeNode {
	return p.Type
}

// GetType returns the type of the destructured parameter
func (p DestructuredParamNode) GetType() TypeNode {
	return p.Type
}
