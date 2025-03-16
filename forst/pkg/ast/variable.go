package ast

// VariableNode represents a variable in the AST
type VariableNode struct {
	ValueNode
	Ident        Ident
	ExplicitType TypeNode
}

func (v VariableNode) String() string {
	return v.Ident.String()
}

func (v VariableNode) NodeType() string {
	return "Variable"
}

// Implement ValueNode interface for VariableNode
func (v VariableNode) isValue() {}
