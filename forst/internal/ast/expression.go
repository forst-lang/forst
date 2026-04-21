package ast

import (
	"fmt"
	"strings"
)

// ExpressionNode represents any expression in the AST
type ExpressionNode interface {
	Node

	isExpression()
	String() string
}

// UnaryExpressionNode represents a unary expression in the AST
type UnaryExpressionNode struct {
	Operator TokenIdent
	Operand  ExpressionNode
}

// BinaryExpressionNode represents a binary expression in the AST
type BinaryExpressionNode struct {
	Left     ExpressionNode
	Operator TokenIdent
	Right    ExpressionNode
}

// FunctionCallNode represents a function call expression
type FunctionCallNode struct {
	Function  Ident
	Arguments []ExpressionNode
	CallSpan  SourceSpan   // from '(' through ')' of this call; zero if unset
	ArgSpans  []SourceSpan // parallel to Arguments when set by parser
}

func (u UnaryExpressionNode) isExpression()  { _ = u }
func (b BinaryExpressionNode) isExpression() { _ = b }
func (f FunctionCallNode) isExpression()     { _ = f }

// IndexExpressionNode is a subscript expression: target[index] (slice, array, or map).
type IndexExpressionNode struct {
	Target ExpressionNode
	Index  ExpressionNode
}

func (IndexExpressionNode) isExpression() {}

// Kind returns the node kind for index expressions.
func (i IndexExpressionNode) Kind() NodeKind {
	return NodeKindIndexExpression
}

func (i IndexExpressionNode) String() string {
	return fmt.Sprintf("%s[%s]", i.Target.String(), i.Index.String())
}

// Kind returns the node kind for unary expressions.
func (u UnaryExpressionNode) Kind() NodeKind {
	return NodeKindUnaryExpression
}

// Kind returns the node kind for binary expressions
func (b BinaryExpressionNode) Kind() NodeKind {
	return NodeKindBinaryExpression
}

// Kind returns the node kind for function calls
func (f FunctionCallNode) Kind() NodeKind {
	return NodeKindFunctionCall
}

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
	return fmt.Sprintf("{%s}(%s)", f.Function.ID, strings.Join(args, ", "))
}
