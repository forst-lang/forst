package ast

// ParamNode represents a function parameter with a name and type
type ParamNode struct {
	Ident Ident    // Parameter name
	Type  TypeNode // Parameter type
}

// NodeType returns the type of this AST node
func (p ParamNode) NodeType() NodeType {
	return NodeTypeParam
}
