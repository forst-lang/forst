package ast

// ValueNode represents a value in the AST, which can be either a variable or a literal
type ValueNode interface {
	ExpressionNode
	isValue() // Marker method to identify value nodes
	String() string
}
