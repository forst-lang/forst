package ast

import "fmt"

// ParamNode represents a function parameter with a name and type
type ParamNode interface {
	Node
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

// NodeType returns the type of this AST node
func (p SimpleParamNode) Kind() NodeKind {
	return NodeKindSimpleParam
}

func (p SimpleParamNode) String() string {
	return fmt.Sprintf("%s: %s", p.Ident.Id, p.Type.String())
}

func (p DestructuredParamNode) String() string {
	return fmt.Sprintf("{%v}: %s", p.Fields, p.Type.String())
}

func (p DestructuredParamNode) Kind() NodeKind {
	return NodeKindDestructuredParam
}
