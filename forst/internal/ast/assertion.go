// Package ast defines the Abstract Syntax Tree (AST) nodes and types used to represent
// the structure of Forst source code after parsing.
package ast

import (
	"fmt"
	"strings"
)

// AssertionNode represents an assertion, which may have a base type and constraints.
// Dot-chained Constraints are Meet. OrChains are Join alternatives after `or`
// (each alternative is itself a Meet chain without nested OrChains).
type AssertionNode struct {
	// Base type is optional when the type can be inferred from the value being checked
	BaseType    *TypeIdent
	Constraints []ConstraintNode
	// OrChains holds subsequent Join alternatives after `or` (same place).
	OrChains []AssertionNode
	// TypeParams holds element/key/value types when BaseType is a type constructor
	// (Array, Map, Pointer, Channel, etc.) — preserved from typedef RHS like `[]Byte`.
	TypeParams []TypeNode
	// ArrayLen is set for fixed-size arrays `[N]T` in typedef aliases.
	ArrayLen *int64
}

// ConstraintNode is a constraint on a type or value, with arguments
type ConstraintNode struct {
	Node
	Name string
	Args []ConstraintArgumentNode
	// Span of the constraint name (and call) for diagnostics.
	Span SourceSpan
}

func (c ConstraintNode) String() string {
	argStrings := make([]string, len(c.Args))
	for i, arg := range c.Args {
		argStrings[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", c.Name, strings.Join(argStrings, ", "))
}

// ConstraintArgumentNode is an argument to a constraint, can be a value, a shape, or a type
type ConstraintArgumentNode struct {
	Value *ValueNode
	Shape *ShapeNode
	Type  *TypeNode
}

// Kind returns the node kind for a constraint argument
func (c ConstraintArgumentNode) Kind() NodeKind {
	if c.Value != nil {
		return (*c.Value).Kind()
	}
	return c.Shape.Kind()
}

// String returns a string representation of the constraint argument
func (c ConstraintArgumentNode) String() string {
	if c.Value != nil {
		return (*c.Value).String()
	}
	if c.Shape != nil {
		return c.Shape.String()
	}
	if c.Type != nil {
		return c.Type.String()
	}
	return "?"
}

// ToTypeNode rebuilds a TypeNode from a typedef/assertion base that may carry constructor params.
func (a AssertionNode) ToTypeNode() (TypeNode, bool) {
	if a.BaseType == nil {
		return TypeNode{}, false
	}
	return TypeNode{
		Ident:      *a.BaseType,
		TypeParams: a.TypeParams,
		ArrayLen:   a.ArrayLen,
	}, true
}

// String returns a string representation of the assertion
func (a AssertionNode) String() string {
	return a.ToString(a.BaseType)
}

// ToString returns a string representation of the assertion with an optional base type
func (a AssertionNode) ToString(baseType *TypeIdent) string {
	constraints := make([]string, len(a.Constraints))
	for i, c := range a.Constraints {
		constraints[i] = c.String()
	}

	constraintsString := strings.Join(constraints, ".")

	var head string
	if baseType == nil {
		head = constraintsString
	} else if constraintsString == "" {
		if len(a.TypeParams) > 0 || a.ArrayLen != nil {
			head = a.mustTypeNodeString(baseType)
		} else {
			head = baseType.String()
		}
	} else {
		head = fmt.Sprintf("%s.%s", baseType.String(), constraintsString)
	}
	for _, alt := range a.OrChains {
		head += " or " + alt.ToString(alt.BaseType)
	}
	return head
}

func (a AssertionNode) mustTypeNodeString(baseType *TypeIdent) string {
	tn := TypeNode{Ident: *baseType, TypeParams: a.TypeParams, ArrayLen: a.ArrayLen}
	return tn.String()
}

// MeetChains returns this assertion's Meet chain plus each OrChains alternative
// as separate Meet-only assertions (OrChains cleared).
func (a AssertionNode) MeetChains() []AssertionNode {
	first := AssertionNode{
		BaseType:    a.BaseType,
		Constraints: a.Constraints,
		TypeParams:  a.TypeParams,
		ArrayLen:    a.ArrayLen,
	}
	if len(a.OrChains) == 0 {
		return []AssertionNode{first}
	}
	out := make([]AssertionNode, 0, 1+len(a.OrChains))
	out = append(out, first)
	out = append(out, a.OrChains...)
	return out
}

// IsBareTypeShape reports BaseType with no constraints and no Join (TypeTarget shape).
func (a AssertionNode) IsBareTypeShape() bool {
	return a.BaseType != nil && len(a.Constraints) == 0 && len(a.OrChains) == 0
}

// Kind returns the node kind for an assertion
func (a AssertionNode) Kind() NodeKind {
	return NodeKindAssertion
}

func (a AssertionNode) isExpression() { _ = a }
