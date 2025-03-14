package ast

// ExpressionNode represents any expression in the AST
type ExpressionNode interface {
	Node
	isExpression() // Marker method to identify expression nodes
}

// Ensure LiteralNode implements ExpressionNode
func (i IntLiteralNode) isExpression()    {}
func (f FloatLiteralNode) isExpression()  {}
func (s StringLiteralNode) isExpression() {}
func (b BoolLiteralNode) isExpression()   {}
