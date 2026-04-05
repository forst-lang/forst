package typechecker

import (
	"forst/internal/ast"
)

// NarrowingTypeGuardsForVariableOccurrence returns type guard names recorded for this source
// occurrence when the binding came from `if subject is …` or ensure-successor narrowing.
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

func (tc *TypeChecker) typeGuardNamesFromAssertionNode(a *ast.AssertionNode) []string {
	if tc == nil || a == nil {
		return nil
	}
	var out []string
	seen := make(map[string]struct{})
	for _, c := range a.Constraints {
		if !tc.IsTypeGuardConstraint(c.Name) {
			continue
		}
		if _, ok := seen[c.Name]; ok {
			continue
		}
		seen[c.Name] = struct{}{}
		out = append(out, c.Name)
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
		if tc.IsTypeGuardConstraint(id) {
			return []string{id}
		}
	case *ast.FunctionCallNode:
		if r != nil {
			id := string(r.Function.ID)
			if tc.IsTypeGuardConstraint(id) {
				return []string{id}
			}
		}
	case ast.VariableNode:
		id := string(r.Ident.ID)
		if tc.IsTypeGuardConstraint(id) {
			return []string{id}
		}
	case *ast.VariableNode:
		if r != nil {
			id := string(r.Ident.ID)
			if tc.IsTypeGuardConstraint(id) {
				return []string{id}
			}
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
