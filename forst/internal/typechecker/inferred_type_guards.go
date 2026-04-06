package typechecker

import (
	"forst/internal/ast"
)

// NarrowingTypeGuardsForVariableOccurrence returns assertion predicate names (Min, Equals,
// user-defined guards, …) recorded for this source occurrence from `if subject is …` or
// ensure-successor narrowing.
func (tc *TypeChecker) NarrowingTypeGuardsForVariableOccurrence(vn ast.VariableNode) []string {
	if tc == nil || !vn.Ident.Span.IsSet() {
		return nil
	}
	k := variableOccurrenceKey{ident: vn.Ident.ID, span: vn.Ident.Span}
	if g := tc.variableOccurrenceNarrowingGuards[k]; len(g) > 0 {
		return append([]string(nil), g...)
	}
	return nil
}

// NarrowingPredicateDisplayForVariableOccurrence returns the dotted predicate suffix from the
// narrowing RHS when available (e.g. `MyStr().Min(12)`), for LSP hover. Does not include the static base type.
func (tc *TypeChecker) NarrowingPredicateDisplayForVariableOccurrence(vn ast.VariableNode) string {
	if tc == nil || !vn.Ident.Span.IsSet() {
		return ""
	}
	k := variableOccurrenceKey{ident: vn.Ident.ID, span: vn.Ident.Span}
	return tc.variableOccurrenceNarrowingPredicateDisplay[k]
}

// typeGuardNamesFromAssertionNode returns labels for hover and NarrowingTypeGuards: every
// assertion constraint (Min, Equals, Match, user-defined guards, …) except internal sentinels.
// When there are no constraints but BaseType names the predicate (e.g. `is MyStr`), the base
// name is included unless it is only a built-in type token (avoids `String.String` for `is String`).
func (tc *TypeChecker) typeGuardNamesFromAssertionNode(a *ast.AssertionNode) []string {
	if tc == nil || a == nil {
		return nil
	}
	var out []string
	seen := make(map[string]struct{})
	add := func(name string) {
		if name == "" || name == ast.ValueConstraint || name == ConstraintMatch {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, c := range a.Constraints {
		add(c.Name)
	}
	if len(a.Constraints) == 0 && a.BaseType != nil {
		b := string(*a.BaseType)
		if tc.IsTypeGuardConstraint(b) || !tc.isBuiltinType(ast.TypeIdent(b)) {
			add(b)
		}
	}
	return out
}

func (tc *TypeChecker) typeGuardNamesFromIsRHS(right ast.Node) []string {
	if tc == nil || right == nil {
		return nil
	}
	switch r := right.(type) {
	case *ast.AssertionNode:
		return tc.typeGuardNamesFromAssertionNode(r)
	case ast.AssertionNode:
		return tc.typeGuardNamesFromAssertionNode(&r)
	case ast.TypeDefAssertionExpr:
		if r.Assertion != nil {
			return tc.typeGuardNamesFromAssertionNode(r.Assertion)
		}
	case *ast.TypeDefAssertionExpr:
		if r != nil && r.Assertion != nil {
			return tc.typeGuardNamesFromAssertionNode(r.Assertion)
		}
	case ast.FunctionCallNode:
		id := string(r.Function.ID)
		if id != "" {
			return []string{id}
		}
	case *ast.FunctionCallNode:
		if r != nil {
			id := string(r.Function.ID)
			if id != "" {
				return []string{id}
			}
		}
	case ast.VariableNode:
		id := string(r.Ident.ID)
		if id == "" || tc.isBuiltinType(ast.TypeIdent(id)) {
			return nil
		}
		return []string{id}
	case *ast.VariableNode:
		if r != nil {
			id := string(r.Ident.ID)
			if id == "" || tc.isBuiltinType(ast.TypeIdent(id)) {
				return nil
			}
			return []string{id}
		}
	}
	return nil
}

// TypeGuardConstraintNamesForInferredType returns user-defined type guard names that appear as
// assertion constraints for this inferred type, in constraint order. Used by LSP hover to show
// doc comments attached to each applicable guard.
func (tc *TypeChecker) TypeGuardConstraintNamesForInferredType(tn ast.TypeNode) []string {
	if tc == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(name string) {
		if name == "" || name == ast.ValueConstraint || name == ConstraintMatch {
			return
		}
		if !tc.IsTypeGuardConstraint(name) {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	walk := func(a *ast.AssertionNode) {
		if a == nil {
			return
		}
		for _, c := range a.Constraints {
			add(c.Name)
		}
	}
	walk(tn.Assertion)
	if def, ok := tc.Defs[tn.Ident].(ast.TypeDefNode); ok {
		if ae, ok := typeDefAssertionFromExpr(def.Expr); ok {
			walk(ae.Assertion)
		}
	}
	return out
}
