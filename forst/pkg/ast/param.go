package ast

import "fmt"

// ParamNode represents a function parameter with a name and type
type ParamNode struct {
	Node
	Ident Ident    // Parameter name
	Type  TypeNode // Parameter type
}

// NodeType returns the type of this AST node
func (p ParamNode) Kind() NodeKind {
	return NodeKindParam
}

func (p ParamNode) String() string {
	return fmt.Sprintf("%s: %s", p.Ident.Id, p.Type.String())
}
