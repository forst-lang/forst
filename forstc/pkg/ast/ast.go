package ast

// AST Node interface
type Node interface{}

// Function Definition Node
type FuncNode struct {
	Name       string
	Params     []string
	ReturnType string
	Body       []Node
}

// Assertion Node
type AssertNode struct {
	Condition string
	ErrorType string
}

// Return Node
type ReturnNode struct {
	Value string
}