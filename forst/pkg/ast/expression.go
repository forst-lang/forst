package ast

import (
	"fmt"
	"strings"
)

// ExpressionNode represents any expression in the AST
type ExpressionNode interface {
	Node
	isExpression() // Marker method to identify expression nodes
	ImplicitType() TypeNode
	String() string
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

type FunctionCallNode struct {
	Function  string
	Arguments []ExpressionNode
	Type      TypeNode
}

// Ensure LiteralNode implements ExpressionNode
func (i IntLiteralNode) isExpression()       {}
func (f FloatLiteralNode) isExpression()     {}
func (s StringLiteralNode) isExpression()    {}
func (b BoolLiteralNode) isExpression()      {}
func (u UnaryExpressionNode) isExpression()  {}
func (b BinaryExpressionNode) isExpression() {}
func (f FunctionCallNode) isExpression()     {}

func (i IntLiteralNode) ImplicitType() TypeNode       { return TypeNode{Name: TypeInt} }
func (f FloatLiteralNode) ImplicitType() TypeNode     { return TypeNode{Name: TypeFloat} }
func (s StringLiteralNode) ImplicitType() TypeNode    { return TypeNode{Name: TypeString} }
func (b BoolLiteralNode) ImplicitType() TypeNode      { return TypeNode{Name: TypeBool} }
func (u UnaryExpressionNode) ImplicitType() TypeNode  { return u.Type }
func (b BinaryExpressionNode) ImplicitType() TypeNode { return b.Type }
func (f FunctionCallNode) ImplicitType() TypeNode     { return f.Type }

func (u UnaryExpressionNode) String() string {
	return fmt.Sprintf("%s %s", u.Operator, u.Operand.String())
}

func (b BinaryExpressionNode) String() string {
	return fmt.Sprintf("%s %s %s", b.Left.String(), b.Operator, b.Right.String())
}

func (f FunctionCallNode) String() string {
	args := make([]string, len(f.Arguments))
	for i, arg := range f.Arguments {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", f.Function, strings.Join(args, ", "))
}
