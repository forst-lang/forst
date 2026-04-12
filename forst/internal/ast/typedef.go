package ast

import (
	"fmt"
)

// TypeDefExpr is an interface for type definition expressions
type TypeDefExpr interface {
	isTypeDefExpr()
	String() string
}

// TypeDefAssertionExpr represents a type definition assertion expression
type TypeDefAssertionExpr struct {
	Assertion *AssertionNode
}

func (t TypeDefAssertionExpr) isTypeDefExpr() { _ = t }

// Kind returns the node kind for a type definition assertion expression
func (t TypeDefAssertionExpr) Kind() NodeKind {
	return NodeKindTypeDefAssertion
}

func (t TypeDefAssertionExpr) String() string {
	if t.Assertion != nil {
		return fmt.Sprintf("TypeDefAssertionExpr(%s)", t.Assertion)
	}
	return "TypeDefAssertionExpr(?)"
}

// TypeDefBinaryExpr represents a binary expression in a type definition
type TypeDefBinaryExpr struct {
	Left  TypeDefExpr
	Op    TokenIdent // TokenBitwiseAnd or TokenBitwiseOr (& or |)
	Right TypeDefExpr
}

// IsConjunction returns true if the binary expression is a conjunction
func (t TypeDefBinaryExpr) IsConjunction() bool {
	return t.Op == TokenBitwiseAnd
}

// IsDisjunction returns true if the binary expression is a disjunction
func (t TypeDefBinaryExpr) IsDisjunction() bool {
	return t.Op == TokenBitwiseOr
}

// Kind returns the node kind for a type definition binary expression
func (t TypeDefBinaryExpr) Kind() NodeKind {
	return NodeKindTypeDefBinaryExpr
}

func (t TypeDefBinaryExpr) String() string {
	return fmt.Sprintf("TypeDefBinaryExpr(%s %s %s)", t.Left, t.Op, t.Right)
}

// isTypeDefExpr marks TypeDefBinaryExpr as implementing TypeDefExpr
func (t TypeDefBinaryExpr) isTypeDefExpr() { _ = t }

// TypeDefShapeExpr represents a shape definition in a type definition
type TypeDefShapeExpr struct {
	Shape ShapeNode
}

func (t TypeDefShapeExpr) isTypeDefExpr() { _ = t }

// Kind returns the node kind for a type definition shape expression
func (t TypeDefShapeExpr) Kind() NodeKind {
	return NodeKindTypeDefShape
}

func (t TypeDefShapeExpr) String() string {
	return fmt.Sprintf("TypeDefShapeExpr(%s)", t.Shape)
}

// TypeDefErrorExpr is the body of `error Name { ... }` (RFC 02): a nominal error type wrapping a payload shape.
// It is distinct from TypeDefShapeExpr so errors are not "just shapes with a flag".
type TypeDefErrorExpr struct {
	Payload ShapeNode
}

func (t TypeDefErrorExpr) isTypeDefExpr() { _ = t }

// Kind returns the node kind for a nominal error type definition expression.
func (t TypeDefErrorExpr) Kind() NodeKind {
	return NodeKindTypeDefError
}

func (t TypeDefErrorExpr) String() string {
	return fmt.Sprintf("TypeDefErrorExpr(%s)", t.Payload)
}

// PayloadShape returns the backing shape for field types and lowering: ordinary shape types or error payloads.
func PayloadShape(expr TypeDefExpr) (*ShapeNode, bool) {
	switch e := expr.(type) {
	case TypeDefShapeExpr:
		return &e.Shape, true
	case TypeDefErrorExpr:
		return &e.Payload, true
	default:
		return nil, false
	}
}

// TypeDefNode represents a type definition node
type TypeDefNode struct {
	Node
	// Name of the type being defined
	Ident TypeIdent
	// Expression defining the type
	Expr TypeDefExpr
}

// Kind returns the node kind for a type definition
func (t TypeDefNode) Kind() NodeKind {
	return NodeKindTypeDef
}

// GetIdent returns the identifier for the type definition
func (t TypeDefNode) GetIdent() string {
	return string(t.Ident)
}

func (t TypeDefNode) String() string {
	return fmt.Sprintf("TypeDef(%s = %s)", t.Ident, t.Expr)
}
