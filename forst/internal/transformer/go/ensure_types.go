package transformergo

import (
	"fmt"
	"forst/internal/ast"
)

const (
	// MinConstraint is the built-in Min constraint in Forst
	MinConstraint = "Min"
	// MaxConstraint is the built-in Max constraint in Forst
	MaxConstraint = "Max"
	// LessThanConstraint is the built-in LessThan constraint in Forst
	LessThanConstraint = "LessThan"
	// GreaterThanConstraint is the built-in GreaterThan constraint in Forst
	GreaterThanConstraint = "GreaterThan"
	// HasPrefixConstraint is the built-in HasPrefix constraint in Forst
	HasPrefixConstraint = "HasPrefix"
	// TrueConstraint is the built-in True constraint in Forst
	TrueConstraint = "True"
	// FalseConstraint is the built-in False constraint in Forst
	FalseConstraint = "False"
	// NilConstraint is the built-in Nil constraint in Forst
	NilConstraint = "Nil"
)

const (
	// BoolConstantTrue is the true constant in Go
	BoolConstantTrue = "true"
	// BoolConstantFalse is the false constant in Go
	BoolConstantFalse = "false"
	// NilConstant is the nil constant in Go
	NilConstant = "nil"
)

// AssertionTransformer handles the transformation of assertions
type AssertionTransformer struct {
	transformer *Transformer
}

// NewAssertionTransformer creates a new AssertionTransformer
func NewAssertionTransformer(t *Transformer) *AssertionTransformer {
	return &AssertionTransformer{transformer: t}
}

// expectValue validates and returns a value node
func (at *AssertionTransformer) expectValue(arg *ast.ConstraintArgumentNode) (ast.ValueNode, error) {
	if arg == nil {
		return nil, fmt.Errorf("expected an argument")
	}

	if arg.Value == nil {
		return nil, fmt.Errorf("expected argument to be a value")
	}

	return *arg.Value, nil
}

// validateConstraintArgs validates the number of arguments for a constraint
func (at *AssertionTransformer) validateConstraintArgs(constraint ast.ConstraintNode, expectedArgs int) error {
	if len(constraint.Args) != expectedArgs {
		return fmt.Errorf("%s constraint requires %d argument(s)", constraint.Name, expectedArgs)
	}
	return nil
}
