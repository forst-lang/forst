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
func (m MethodCallNode) isExpression()      { _ = m }

// MethodCallNode is a method call on an arbitrary receiver expression: recv.Method(args).
type MethodCallNode struct {
	Receiver  ExpressionNode
	Method    Ident
	Arguments []ExpressionNode
	CallSpan  SourceSpan
	ArgSpans  []SourceSpan
}

// IndexExpressionNode is a subscript expression: target[index] (slice, array, or map).
type IndexExpressionNode struct {
	Target ExpressionNode
	Index  ExpressionNode
}

func (i IndexExpressionNode) isExpression() { _ = i }

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

func (m MethodCallNode) Kind() NodeKind {
	return NodeKindMethodCall
}

func (m MethodCallNode) String() string {
	var b strings.Builder
	b.WriteString(m.Receiver.String())
	b.WriteByte('.')
	b.WriteString(string(m.Method.ID))
	b.WriteByte('(')
	for i, a := range m.Arguments {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(a.String())
	}
	b.WriteByte(')')
	return b.String()
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
