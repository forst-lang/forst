package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
)

// BuiltinConstraint names a compiler-recognized ensure/guard constraint (Min,
// Max, ...) so it can be told apart from user-defined TypeGuardNode entries
// in TypeChecker.Defs.
type BuiltinConstraint string

const (
	// MinConstraint is the built-in Min constraint in Forst
	MinConstraint BuiltinConstraint = "Min"
	// MaxConstraint is the built-in Max constraint in Forst
	MaxConstraint BuiltinConstraint = "Max"
	// MinBytesConstraint is UTF-8 byte lower bound on String
	MinBytesConstraint BuiltinConstraint = "MinBytes"
	// MaxBytesConstraint is UTF-8 byte upper bound on String
	MaxBytesConstraint BuiltinConstraint = "MaxBytes"
	// LessThanConstraint is the built-in LessThan constraint in Forst
	LessThanConstraint BuiltinConstraint = "LessThan"
	// GreaterThanConstraint is the built-in GreaterThan constraint in Forst
	GreaterThanConstraint BuiltinConstraint = "GreaterThan"
	// HasPrefixConstraint is the built-in HasPrefix constraint in Forst
	HasPrefixConstraint BuiltinConstraint = "HasPrefix"
	// HasSuffixConstraint is the built-in HasSuffix constraint in Forst
	HasSuffixConstraint BuiltinConstraint = "HasSuffix"
	// ContainsConstraint is the built-in Contains constraint in Forst
	ContainsConstraint BuiltinConstraint = "Contains"
	// TrueConstraint is the built-in True constraint in Forst
	TrueConstraint BuiltinConstraint = "True"
	// FalseConstraint is the built-in False constraint in Forst
	FalseConstraint BuiltinConstraint = "False"
	// NilConstraint is the built-in Nil constraint in Forst
	NilConstraint BuiltinConstraint = "Nil"
	// PresentConstraint is the built-in NotNil constraint in Forst
	PresentConstraint BuiltinConstraint = "Present"
	// NotEmptyConstraint is the built-in NotEmpty constraint in Forst
	NotEmptyConstraint BuiltinConstraint = "NotEmpty"
	// FiniteConstraint is the built-in Finite constraint for Float
	FiniteConstraint BuiltinConstraint = "Finite"
	// ValueConstraint is the built-in Value constraint in Forst
	ValueConstraint BuiltinConstraint = ast.ValueConstraint
)

// BuiltinConstraintNames is the closed set of value/refinement builtin names
// (excluding structural Match and plugin markers such as Router).
var BuiltinConstraintNames = []string{
	string(MinConstraint),
	string(MaxConstraint),
	string(MinBytesConstraint),
	string(MaxBytesConstraint),
	string(LessThanConstraint),
	string(GreaterThanConstraint),
	string(HasPrefixConstraint),
	string(HasSuffixConstraint),
	string(ContainsConstraint),
	string(TrueConstraint),
	string(FalseConstraint),
	string(NilConstraint),
	string(PresentConstraint),
	string(NotEmptyConstraint),
	string(FiniteConstraint),
	string(ValueConstraint),
}

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

// constraintArgAsExpr lowers a constraint argument (literal value or param ident parsed as Type).
func (at *AssertionTransformer) constraintArgAsExpr(arg ast.ConstraintArgumentNode) (goast.Expr, error) {
	if arg.Value != nil {
		return at.transformer.transformExpression((*arg.Value).(ast.ExpressionNode))
	}
	if arg.Type != nil && arg.Shape == nil {
		return goast.NewIdent(string(arg.Type.Ident)), nil
	}
	return nil, fmt.Errorf("expected argument to be a value")
}
