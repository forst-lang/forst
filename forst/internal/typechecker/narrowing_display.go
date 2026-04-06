package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// narrowingPredicateDisplayFromIsRHS builds a dotted "call" chain for hover, e.g. `MyStr().Min(12)`,
// from the RHS of `x is …` or the assertion in `ensure x is …`. Empty string means nothing to show.
func (tc *TypeChecker) narrowingPredicateDisplayFromIsRHS(right ast.Node) string {
	segs := tc.narrowingPredicateSegmentsFromIsRHS(right)
	if len(segs) == 0 {
		return ""
	}
	return strings.Join(segs, ".")
}

func (tc *TypeChecker) narrowingPredicateSegmentsFromIsRHS(right ast.Node) []string {
	if tc == nil || right == nil {
		return nil
	}
	switch r := right.(type) {
	case ast.TypeDefAssertionExpr:
		if r.Assertion != nil {
			return tc.narrowingPredicateSegmentsFromAssertion(r.Assertion)
		}
	case *ast.TypeDefAssertionExpr:
		if r != nil && r.Assertion != nil {
			return tc.narrowingPredicateSegmentsFromAssertion(r.Assertion)
		}
	case ast.AssertionNode:
		return tc.narrowingPredicateSegmentsFromAssertion(&r)
	case *ast.AssertionNode:
		if r != nil {
			return tc.narrowingPredicateSegmentsFromAssertion(r)
		}
	case ast.FunctionCallNode:
		return []string{formatPredicateCallDisplay(r)}
	case *ast.FunctionCallNode:
		if r != nil {
			return []string{formatPredicateCallDisplay(*r)}
		}
	case ast.VariableNode:
		return tc.narrowingPredicateSegmentFromNamedTypeVar(&r)
	case *ast.VariableNode:
		if r != nil {
			return tc.narrowingPredicateSegmentFromNamedTypeVar(r)
		}
	}
	return nil
}

func (tc *TypeChecker) narrowingPredicateSegmentFromNamedTypeVar(v *ast.VariableNode) []string {
	if v == nil {
		return nil
	}
	id := string(v.Ident.ID)
	if id == "" || tc.isBuiltinType(ast.TypeIdent(id)) {
		return nil
	}
	return []string{id + "()"}
}

// narrowingPredicateSegmentsFromAssertion mirrors assertion structure for display: non-builtin base
// types with constraints become `Base().Constraint(…)`; built-in bases only contribute constraints.
func (tc *TypeChecker) narrowingPredicateSegmentsFromAssertion(a *ast.AssertionNode) []string {
	if a == nil {
		return nil
	}
	var segs []string
	if a.BaseType != nil && len(a.Constraints) > 0 {
		b := string(*a.BaseType)
		if tc.IsTypeGuardConstraint(b) || !tc.isBuiltinType(ast.TypeIdent(b)) {
			segs = append(segs, b+"()")
		}
	}
	for _, c := range a.Constraints {
		segs = append(segs, c.String())
	}
	if len(a.Constraints) == 0 && a.BaseType != nil {
		b := string(*a.BaseType)
		if tc.IsTypeGuardConstraint(b) || !tc.isBuiltinType(ast.TypeIdent(b)) {
			segs = append(segs, b+"()")
		}
	}
	return segs
}

func formatPredicateCallDisplay(f ast.FunctionCallNode) string {
	args := make([]string, len(f.Arguments))
	for i, arg := range f.Arguments {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", f.Function.ID, strings.Join(args, ", "))
}
