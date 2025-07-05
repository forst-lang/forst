package ast

import (
	"fmt"
	"strings"
)

// LiteralNode represents a literal value in the AST
type LiteralNode interface {
	ValueNode
	isLiteral() // Marker method to identify literal nodes
}

// IntLiteralNode represents an integer literal
type IntLiteralNode struct {
	Value int64
	Type  TypeNode
}

// FloatLiteralNode represents a float literal
type FloatLiteralNode struct {
	Value float64
	Type  TypeNode
}

// StringLiteralNode represents a string literal
type StringLiteralNode struct {
	Value string
	Type  TypeNode
}

// BoolLiteralNode represents a boolean literal
type BoolLiteralNode struct {
	Value bool
	Type  TypeNode
}

// ArrayLiteralNode represents an array literal
type ArrayLiteralNode struct {
	Value []LiteralNode
	Type  TypeNode
}

// MapEntryNode represents a key-value pair in a map literal
type MapEntryNode struct {
	Key   ValueNode
	Value ValueNode
}

// MapLiteralNode represents a map literal
type MapLiteralNode struct {
	Entries []MapEntryNode
	Type    TypeNode
}

// NilLiteralNode represents the nil literal
// This is used instead of VariableNode with Ident 'nil'
type NilLiteralNode struct{}

// Kind returns the node kind for an integer literal
func (i IntLiteralNode) Kind() NodeKind {
	return NodeKindIntLiteral
}

// Kind returns the node kind for a float literal
func (f FloatLiteralNode) Kind() NodeKind {
	return NodeKindFloatLiteral
}

// Kind returns the node kind for a string literal
func (s StringLiteralNode) Kind() NodeKind {
	return NodeKindStringLiteral
}

// Kind returns the node kind for a boolean literal
func (b BoolLiteralNode) Kind() NodeKind {
	return NodeKindBoolLiteral
}

// Kind returns the node kind for an array literal
func (a ArrayLiteralNode) Kind() NodeKind {
	return NodeKindArrayLiteral
}

// Kind returns the node kind for a map literal
func (m MapLiteralNode) Kind() NodeKind {
	return NodeKindMapLiteral
}

// Kind returns the node kind for nil literal
func (n NilLiteralNode) Kind() NodeKind {
	return NodeKindNilLiteral
}

// Marker methods to satisfy LiteralNode interface
func (i IntLiteralNode) isLiteral()    {}
func (f FloatLiteralNode) isLiteral()  {}
func (s StringLiteralNode) isLiteral() {}
func (b BoolLiteralNode) isLiteral()   {}
func (a ArrayLiteralNode) isLiteral()  {}
func (m MapLiteralNode) isLiteral()    {}
func (n NilLiteralNode) isLiteral()    {}

// Implement ValueNode interface for all literal nodes
func (i IntLiteralNode) isValue()    {}
func (f FloatLiteralNode) isValue()  {}
func (s StringLiteralNode) isValue() {}
func (b BoolLiteralNode) isValue()   {}
func (a ArrayLiteralNode) isValue()  {}
func (m MapLiteralNode) isValue()    {}
func (n NilLiteralNode) isValue()    {}

// Ensures LiteralNode implements ExpressionNode
func (i IntLiteralNode) isExpression()    {}
func (f FloatLiteralNode) isExpression()  {}
func (s StringLiteralNode) isExpression() {}
func (b BoolLiteralNode) isExpression()   {}
func (a ArrayLiteralNode) isExpression()  {}
func (m MapLiteralNode) isExpression()    {}
func (n NilLiteralNode) isExpression()    {}

func (i IntLiteralNode) String() string {
	return fmt.Sprintf("%d", i.Value)
}

func (f FloatLiteralNode) String() string {
	return fmt.Sprintf("%g", f.Value)
}

func (s StringLiteralNode) String() string {
	return fmt.Sprintf("\"%s\"", s.Value)
}

func (b BoolLiteralNode) String() string {
	return fmt.Sprintf("%t", b.Value)
}

func (a ArrayLiteralNode) String() string {
	items := []string{}
	for _, item := range a.Value {
		items = append(items, item.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

func (m MapLiteralNode) String() string {
	entries := make([]string, len(m.Entries))
	for i, entry := range m.Entries {
		entries[i] = fmt.Sprintf("%s: %s", entry.Key.String(), entry.Value.String())
	}
	return fmt.Sprintf("map[%s]%s{%s}", m.Type.TypeParams[0].String(), m.Type.TypeParams[1].String(), strings.Join(entries, ", "))
}

func (n NilLiteralNode) String() string {
	return "nil"
}
