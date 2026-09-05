package typechecker

import "forst/internal/ast"

// ConstraintMatch names the "Match" guard constraint whose sole argument is
// a nested Shape, used by shape-pattern narrowing (`is` guards) to bind and
// narrow a value against that shape rather than a plain named type guard.
const ConstraintMatch = "Match"

// IsBuiltinAssertionConstraintName reports built-in value/refinement constraints that are not
// user-defined TypeGuardNode entries in Defs (see internal/transformer/go/ensure_types.go).
func IsBuiltinAssertionConstraintName(name string) bool {
	return isBuiltinAssertionConstraintName(name)
}

func isBuiltinAssertionConstraintName(name string) bool {
	switch name {
	case "Min", "Max", "MinBytes", "MaxBytes", "LessThan", "GreaterThan",
		"HasPrefix", "HasSuffix", "Contains",
		"True", "False", "Nil", "Present", "NotEmpty", "Finite", "Router", ast.ValueConstraint:
		return true
	default:
		return false
	}
}
