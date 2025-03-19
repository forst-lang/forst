package ast

import (
	"fmt"
	"strings"
)

type AssertionNode struct {
	// Base type is optional as the type can be inferred from the value being checked
	BaseType    *TypeIdent
	Constraints []ConstraintNode
}

type ConstraintNode struct {
	Name string
	Args []ConstraintArgumentNode
}

type ConstraintArgumentNode struct {
	Value *ValueNode
	Shape *ShapeNode
}

func (c ConstraintArgumentNode) String() string {
	if c.Value != nil {
		return (*c.Value).String()
	}
	return c.Shape.String()
}

func (c ConstraintArgumentNode) Kind() NodeKind {
	if c.Value != nil {
		return (*c.Value).Kind()
	}
	return c.Shape.Kind()
}

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

func (a AssertionNode) Kind() NodeKind {
	return NodeKindAssertion
}
