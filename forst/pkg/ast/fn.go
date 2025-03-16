package ast

import "fmt"

// Function Node represents a function definition with optional parameters and an optional return type
type FunctionNode struct {
	Ident      Ident
	Params     []ParamNode
	ReturnType TypeNode
	Body       []Node
}

// NodeType returns the type of this AST node
func (f FunctionNode) NodeType() string {
	return "Function"
}

func (f FunctionNode) HasExplicitReturnType() bool {
	return f.ReturnType.IsExplicit()
}

func (f FunctionNode) String() string {
	return fmt.Sprintf("Function(%s)", f.Ident.Id)
}

func (f FunctionNode) Id() Identifier {
	return f.Ident.Id
}
