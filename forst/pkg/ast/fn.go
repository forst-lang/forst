package ast

// Function Node represents a function definition with optional parameters and an optional return type
type FunctionNode struct {
	Name               string
	Params             []ParamNode
	ExplicitReturnType TypeNode
	ImplicitReturnType TypeNode
	Body               []Node
}

// NodeType returns the type of this AST node
func (f FunctionNode) NodeType() string {
	return "Function"
}
