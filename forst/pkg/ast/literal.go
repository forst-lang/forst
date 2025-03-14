package ast

type LiteralNode interface {
	ExpressionNode
	isLiteral() // Marker method to identify literal nodes
}

// IntLiteralNode represents an integer literal
type IntLiteralNode struct {
	Value int64
}

// FloatLiteralNode represents a float literal
type FloatLiteralNode struct {
	Value float64
}

// StringLiteralNode represents a string literal
type StringLiteralNode struct {
	Value string
}

// BoolLiteralNode represents a boolean literal
type BoolLiteralNode struct {
	Value bool
}

// NodeType returns the type of this AST node
func (i IntLiteralNode) NodeType() string {
	return "IntLiteral"
}

func (f FloatLiteralNode) NodeType() string {
	return "FloatLiteral"
}

func (s StringLiteralNode) NodeType() string {
	return "StringLiteral"
}

func (b BoolLiteralNode) NodeType() string {
	return "BoolLiteral"
}

// Marker methods to satisfy LiteralNode interface
func (i IntLiteralNode) isLiteral()    {}
func (f FloatLiteralNode) isLiteral()  {}
func (s StringLiteralNode) isLiteral() {}
func (b BoolLiteralNode) isLiteral()   {}
