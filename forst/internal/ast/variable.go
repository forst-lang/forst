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

func (v VariableNode) Kind() NodeKind {
	return NodeKindVariable
}

func (v VariableNode) Id() string {
	return string(v.Ident.Id)
}

// Implement ValueNode interface for VariableNode
func (v VariableNode) isValue() {}
