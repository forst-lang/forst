package ast

// Function Node represents a function definition with optional parameters and an optional return type
type FunctionNode struct {
	Name       string
	Params     []ParamNode
	ReturnType TypeNode
	Body       []Node
}
