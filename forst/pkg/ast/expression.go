package ast

// ExpressionNode represents any expression in the AST
type ExpressionNode interface {
	Node
	isExpression() // Marker method to identify expression nodes
	ImplicitType() TypeNode
}

// UnaryExpressionNode represents a unary expression in the AST
type UnaryExpressionNode struct {
	Operator TokenType
	Operand  ExpressionNode
	// The type of the expression
	Type TypeNode
}

// BinaryExpressionNode represents a binary expression in the AST
type BinaryExpressionNode struct {
	Left     ExpressionNode
	Operator TokenType
	Right    ExpressionNode
	// The type of the expression
	Type TypeNode
}

// Ensure LiteralNode implements ExpressionNode
func (i IntLiteralNode) isExpression()       {}
func (f FloatLiteralNode) isExpression()     {}
func (s StringLiteralNode) isExpression()    {}
func (b BoolLiteralNode) isExpression()      {}
func (u UnaryExpressionNode) isExpression()  {}
func (b BinaryExpressionNode) isExpression() {}

func (i IntLiteralNode) ImplicitType() TypeNode       { return TypeNode{Name: TypeInt} }
func (f FloatLiteralNode) ImplicitType() TypeNode     { return TypeNode{Name: TypeFloat} }
func (s StringLiteralNode) ImplicitType() TypeNode    { return TypeNode{Name: TypeString} }
func (b BoolLiteralNode) ImplicitType() TypeNode      { return TypeNode{Name: TypeBool} }
func (u UnaryExpressionNode) ImplicitType() TypeNode  { return u.Type }
func (b BinaryExpressionNode) ImplicitType() TypeNode { return b.Type }
