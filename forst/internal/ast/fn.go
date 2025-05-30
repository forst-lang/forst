package ast

import "fmt"

// FunctionNode represents a function definition with optional parameters and an optional return type
type FunctionNode struct {
	Ident       Ident
	Params      []ParamNode
	ReturnTypes []TypeNode
	Body        []Node
}

// Kind returns the node kind for functions
func (f FunctionNode) Kind() NodeKind {
	return NodeKindFunction
}

// HasExplicitReturnType returns whether the function has an explicit return type
func (f FunctionNode) HasExplicitReturnType() bool {
	return len(f.ReturnTypes) > 0
}

func (f FunctionNode) String() string {
	return fmt.Sprintf("Function(%s)", f.Ident.ID)
}

// GetIdent returns the function's identifier
func (f FunctionNode) GetIdent() string {
	return string(f.Ident.ID)
}

// HasMainFunctionName returns whether this is the main function
func (f FunctionNode) HasMainFunctionName() bool {
	return f.Ident.ID == "main"
}
