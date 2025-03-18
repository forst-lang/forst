package ast

import (
	"fmt"
	"strings"
)

type AssertionNode struct {
	// Base type is optional as the type can be inferred from the value being checked
	BaseType    *string
	Constraints []ConstraintNode
}

type ConstraintNode struct {
	Name string
	Args []ValueNode
}

func (a AssertionNode) String() string {
	constraints := make([]string, len(a.Constraints))
	for i, c := range a.Constraints {
		if len(c.Args) > 0 {
			argStrings := make([]string, len(c.Args))
			for j, arg := range c.Args {
				argStrings[j] = arg.String()
			}
			constraints[i] = fmt.Sprintf("%s(%s)", c.Name, strings.Join(argStrings, ", "))
		} else {
			constraints[i] = c.Name
		}
	}

	constraintsString := strings.Join(constraints, ".")
	if a.BaseType == nil {
		return constraintsString
	}
	return fmt.Sprintf("%s.%s", *a.BaseType, constraintsString)
}

func (a AssertionNode) Kind() NodeKind {
	return NodeKindAssertion
}
