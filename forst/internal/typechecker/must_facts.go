package typechecker

import (
	"strings"

	"forst/internal/ast"
)

// MustAtoms returns assertion atoms true on every successful path of a
// (must(All)=union of children, must(Any)=intersection of children).
func MustAtoms(a Assertion) []Atom {
	if a == nil {
		return nil
	}
	switch v := a.(type) {
	case Atom:
		return []Atom{v}
	case All:
		var out []Atom
		for _, c := range v.Children {
			out = append(out, MustAtoms(c)...)
		}
		return out
	case Any:
		if len(v.Children) == 0 {
			return nil
		}
		inter := MustAtoms(v.Children[0])
		for i := 1; i < len(v.Children); i++ {
			inter = intersectAtoms(inter, MustAtoms(v.Children[i]))
		}
		return inter
	default:
		return nil
	}
}

// atomKey is a stable identity string for must()/intersection of atoms.
func atomKey(a Atom) string {
	return a.String()
}

// intersectAtoms implements must(Any)=intersection of atom sets.
func intersectAtoms(a, b []Atom) []Atom {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := make(map[string]Atom, len(b))
	for _, x := range b {
		set[atomKey(x)] = x
	}
	var out []Atom
	seen := map[string]struct{}{}
	for _, x := range a {
		k := atomKey(x)
		if _, ok := set[k]; !ok {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, x)
	}
	return out
}

// exportMustFromNamedGuard proves must()-exported conjuncts from a named type guard
// body onto the call-site subject (e.g. LoggedIn → Present(ctx.session), Present(ctx.user)).
func (tc *TypeChecker) exportMustFromNamedGuard(subject ast.VariableNode, guardName string) {
	if tc == nil || guardName == "" {
		return
	}
	def, ok := tc.Defs[ast.TypeIdent(guardName)]
	if !ok {
		return
	}
	var gn ast.TypeGuardNode
	switch d := def.(type) {
	case *ast.TypeGuardNode:
		gn = *d
	case ast.TypeGuardNode:
		gn = d
	default:
		return
	}
	subjIdent := string(gn.Subject.GetIdent())
	callRoot := string(subject.Ident.ID)

	for _, node := range gn.Body {
		ens, ok := ensureStmt(node)
		if !ok {
			continue
		}
		// Only top-level sequential ensures contribute to must(All). Nested if/or
		// bodies are handled via MustAtoms on the guard IR when needed.
		pathID := rebaseGuardEnsurePath(subjIdent, callRoot, string(ens.Variable.Ident.ID))
		if pathID == "" {
			continue
		}
		guards := tc.typeGuardNamesFromAssertionNode(&ens.Assertion)
		if len(ens.Assertion.OrChains) > 0 {
			// must(Any) — only shared conjuncts; individual disjuncts are not exported.
			continue
		}
		vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(pathID), Span: subject.Ident.Span}}
		if len(guards) > 0 {
			tc.recordCompoundNarrowingIdentifier(vn.Ident.ID, guards, strings.Join(guards, "."))
			// Also register a shadow binding when the path is dotted so LookupVariable finds it.
			if typ, err := tc.LookupVariableType(&vn, tc.CurrentScope()); err == nil {
				tc.scopeStack.currentScope().RegisterSymbolWithNarrowing(
					vn.Ident.ID, []ast.TypeNode{typ}, SymbolVariable, guards, strings.Join(guards, "."))
			}
		}
		if tc.predicates != nil {
			ir, _ := LowerRefinementTarget(ens.Target, ens.Assertion)
			if ir != nil {
				path := tc.AccessPathForVariable(&vn)
				tc.CurrentRefinementContext().Prove(path, tc.predicates.FromAssertion(ir))
			}
		}
	}
}

// ensureStmt unwraps EnsureNode / *EnsureNode from a body statement.
func ensureStmt(node ast.Node) (ast.EnsureNode, bool) {
	switch n := node.(type) {
	case ast.EnsureNode:
		return n, true
	case *ast.EnsureNode:
		if n == nil {
			return ast.EnsureNode{}, false
		}
		return *n, true
	default:
		return ast.EnsureNode{}, false
	}
}

// rebaseGuardEnsurePath maps guard-body subject paths onto the call-site root.
// e.g. guard subject "ctx", ensure "ctx.session", call root "ctx" → "ctx.session"
// call root "op.ctx" → "op.ctx.session"
func rebaseGuardEnsurePath(guardSubject, callRoot, ensurePath string) string {
	if ensurePath == "" || guardSubject == "" {
		return ""
	}
	if ensurePath == guardSubject {
		return callRoot
	}
	prefix := guardSubject + "."
	if !strings.HasPrefix(ensurePath, prefix) {
		return ""
	}
	suffix := ensurePath[len(prefix):]
	if callRoot == "" {
		return suffix
	}
	return callRoot + "." + suffix
}

// exportMustFromAssertion exports must-facts for each named type-guard constraint
// on the ensure/if assertion (including Meet chains; skipping Or disjuncts).
func (tc *TypeChecker) exportMustFromAssertion(subject ast.VariableNode, a *ast.AssertionNode) {
	if tc == nil || a == nil {
		return
	}
	if len(a.OrChains) > 0 {
		return
	}
	for _, c := range a.Constraints {
		if tc.IsTypeGuardConstraint(c.Name) {
			tc.exportMustFromNamedGuard(subject, c.Name)
		}
	}
	if len(a.Constraints) == 0 && a.BaseType != nil {
		name := string(*a.BaseType)
		if tc.IsTypeGuardConstraint(name) {
			tc.exportMustFromNamedGuard(subject, name)
		}
	}
}
