// Package ast defines the Abstract Syntax Tree (AST) nodes and types used to represent
// the structure of Forst source code after parsing.
package ast

import (
	"fmt"
	"strings"
)

// AssertionNode represents an assertion, which may have a base type and constraints
type AssertionNode struct {
	// Base type is optional when the type can be inferred from the value being checked
	BaseType    *TypeIdent
	Constraints []ConstraintNode
}

// ConstraintNode is a constraint on a type or value, with arguments
type ConstraintNode struct {
	Node
	Name string
	Args []ConstraintArgumentNode
}

// ConstraintArgumentNode is an argument to a constraint, can be a value or a shape
type ConstraintArgumentNode struct {
	Value *ValueNode
	Shape *ShapeNode
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
	return c.Shape.String()
}

// String returns a string representation of the assertion
func (a AssertionNode) String() string {
	constraints := make([]string, len(a.Constraints))
	for i, c := range a.Constraints {
		argStrings := make([]string, len(c.Args))
		for j, arg := range c.Args {
			argStrings[j] = arg.String()
		}
		constraints[i] = fmt.Sprintf("%s(%s)", c.Name, strings.Join(argStrings, ", "))
	}

	constraintsString := strings.Join(constraints, ".")
	if a.BaseType == nil {
		return constraintsString
	}
	if constraintsString == "" {
		return string(*a.BaseType)
	}
	return fmt.Sprintf("%s.%s", *a.BaseType, constraintsString)
}

// Kind returns the node kind for an assertion
func (a AssertionNode) Kind() NodeKind {
	return NodeKindAssertion
}
