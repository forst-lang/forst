package ast

import (
	"fmt"
)

type TypeDefExpr interface {
	isTypeDefExpr()
	String() string
}

type TypeDefAssertionExpr struct {
	Assertion *AssertionNode
}

func (t TypeDefAssertionExpr) isTypeDefExpr() {}

type TypeDefBinaryExpr struct {
	Left  TypeDefExpr
	Op    TokenIdent // TokenBitwiseAnd or TokenBitwiseOr (& or |)
	Right TypeDefExpr
}

func (t TypeDefBinaryExpr) isTypeDefExpr() {}

func (t TypeDefBinaryExpr) IsConjunction() bool {
	return t.Op == TokenBitwiseAnd
}

func (t TypeDefBinaryExpr) IsDisjunction() bool {
	return t.Op == TokenBitwiseOr
}

type TypeDefNode struct {
	Node
	// Name of the type being defined
	Ident TypeIdent
	// Expression defining the type
	Expr TypeDefExpr
}

func (t TypeDefNode) Kind() NodeKind {
	return NodeKindTypeDef
}

func (t TypeDefNode) String() string {
	return fmt.Sprintf("TypeDefNode(%s)", t.Ident)
}

func (e TypeDefBinaryExpr) String() string {
	return fmt.Sprintf("TypeDefBinaryExpr(%s %s %s)", e.Left, e.Op, e.Right)
}

func (e TypeDefAssertionExpr) String() string {
	return fmt.Sprintf("TypeDefAssertionExpr(%s)", e.Assertion)
}
