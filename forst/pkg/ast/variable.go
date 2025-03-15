package ast

// VariableNode represents a variable in the AST
type VariableNode struct {
	ValueNode
	Name string
	Type TypeNode
}

func (v VariableNode) String() string {
	return v.Name
}

func (v VariableNode) NodeType() string {
	return "Variable"
}

func (v VariableNode) ImplicitType() TypeNode { return v.Type }

// Implement ValueNode interface for VariableNode
func (v VariableNode) isValue() {}
