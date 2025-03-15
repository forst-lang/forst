package ast

// Function Node represents a function definition with optional parameters and an optional return type
type FunctionNode struct {
	Ident              Ident
	Params             []ParamNode
	ExplicitReturnType TypeNode
	Body               []Node
}

// NodeType returns the type of this AST node
func (f FunctionNode) NodeType() string {
	return "Function"
}

func (f FunctionNode) HasExplicitReturnType() bool {
	return f.ExplicitReturnType.IsExplicit()
}
