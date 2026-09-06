package semantic

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
)

func projectConstraints(tc *typechecker.TypeChecker, baseKind string, a *ast.AssertionNode) []Constraint {
	if a == nil || len(a.Constraints) == 0 {
		return nil
	}
	out := make([]Constraint, 0, len(a.Constraints))
	for _, c := range a.Constraints {
		if c.Name == typechecker.ConstraintMatch {
			continue
		}
		origin := "typeGuard"
		if isKnownBuiltinName(c.Name) {
			origin = "builtin"
		} else if tc != nil && tc.IsTypeGuardConstraint(c.Name) {
			origin = "typeGuard"
		}
		cons := Constraint{
			Name:   c.Name,
			Args:   constraintArgs(c.Args),
			Origin: origin,
		}
		if applies := constraintApplies(baseKind, c.Name); applies != "" {
			cons.Applies = applies
		}
		out = append(out, cons)
	}
	return out
}

// IsKnownBuiltinConstraintName reports whether name is a closed builtin constraint atom.
func IsKnownBuiltinConstraintName(name string) bool {
	return isKnownBuiltinName(name)
}

func isKnownBuiltinName(name string) bool {
	switch name {
	case "Min", "Max", "MinBytes", "MaxBytes", "LessThan", "GreaterThan",
		"HasPrefix", "HasSuffix", "Contains",
		"True", "False", "Nil", "Present", "NotEmpty", "Finite", "Router", ast.ValueConstraint:
		return true
	default:
		return false
	}
}

func constraintApplies(baseKind, name string) string {
	switch baseKind {
	case "string":
		switch name {
		case "Min", "Max":
			return "length"
		case "MinBytes", "MaxBytes":
			return "bytes"
		}
	case "array":
		if name == "Min" || name == "Max" || name == "NotEmpty" {
			return "items"
		}
	case "map":
		if name == "Min" || name == "Max" || name == "NotEmpty" {
			return "value"
		}
	case "bytes":
		if name == "Min" || name == "Max" || name == "NotEmpty" {
			return "items"
		}
	}
	return ""
}

func constraintArgs(args []ast.ConstraintArgumentNode) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, 0, len(args))
	for _, arg := range args {
		if v := literalArgValue(arg); v != nil {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func literalArgValue(arg ast.ConstraintArgumentNode) any {
	if arg.Value != nil {
		switch v := (*arg.Value).(type) {
		case ast.IntLiteralNode:
			return v.Value
		case ast.FloatLiteralNode:
			return v.Value
		case ast.StringLiteralNode:
			return v.Value
		case ast.BoolLiteralNode:
			return v.Value
		case ast.RuneLiteralNode:
			return v.Value
		}
	}
	return nil
}
