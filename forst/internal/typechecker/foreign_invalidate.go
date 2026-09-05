package typechecker

import (
	"forst/internal/ast"
)

// invalidateAfterUntrustedGoCall drops facts on reachable mutable storage of
// arguments after a Go/reflect/unsafe/sibling-.go call (phase 4f).
// The call itself remains legal.
func (tc *TypeChecker) invalidateAfterUntrustedGoCall(e ast.FunctionCallNode) {
	if tc == nil {
		return
	}
	span := e.CallSpan
	if !span.IsSet() {
		span = e.Function.Span
	}
	for _, arg := range e.Arguments {
		tc.invalidateReachableMutableArg(arg, span, dropByForeign)
	}
	// Method-style: receiver is first "arg" via Function "recv.Method" — handled by MethodCallNode path.
}

// invalidateReachableMutableArg drops facts on arg's root (and &x inners) when storage may be mutated abroad.
func (tc *TypeChecker) invalidateReachableMutableArg(arg ast.ExpressionNode, span ast.SourceSpan, reason dropReason) {
	if arg == nil {
		return
	}
	// &x → invalidate x
	if ref, ok := arg.(ast.ReferenceNode); ok {
		inner := refValueExpr(ref)
		if inner != nil {
			tc.invalidateReachableMutableArg(inner, span, reason)
		}
		return
	}
	path := tc.accessPathForExpr(arg)
	if path == nil {
		return
	}
	root := tc.paths.Intern(AccessPath{Root: path.Root})
	// Scalars / pure strings: retain when proved pure.
	if types, ok := tc.lookupArgTypes(arg); ok && len(types) == 1 {
		t := types[0]
		if t.Ident == ast.TypeInt || t.Ident == ast.TypeFloat || t.Ident == ast.TypeBool || t.Ident == ast.TypeString {
			return
		}
		if !tc.TypeMayAlias(t) && t.Ident != ast.TypePointer && t.Ident != ast.TypeArray && t.Ident != ast.TypeMap && t.Ident != ast.TypeObject {
			return
		}
	}
	tc.invalidateOverlappingFactsWithReason(root, span, reason)
}

// lookupArgTypes resolves types for a call argument via scope (variable or dotted path).
func (tc *TypeChecker) lookupArgTypes(arg ast.ExpressionNode) ([]ast.TypeNode, bool) {
	if vn, ok := arg.(ast.VariableNode); ok {
		if ts, ok := tc.scopeStack.LookupVariableType(vn.Ident.ID); ok {
			return ts, true
		}
	}
	if id := dottedIdentFromExpr(arg); id != "" {
		if ts, ok := tc.scopeStack.LookupVariableType(ast.Identifier(id)); ok {
			return ts, true
		}
	}
	return nil, false
}

// treatUnresolvedQualifiedCallAsForeign handles pkg.Func when the package is not
// a loaded Forst/Go import — degrade: invalidate, do not reject (4f-16).
func (tc *TypeChecker) treatUnresolvedQualifiedCallAsForeign(e ast.FunctionCallNode) ([]ast.TypeNode, bool) {
	parts := splitQualified(string(e.Function.ID))
	if len(parts) != 2 {
		return nil, false
	}
	pkg := parts[0]
	// Do not swallow real local variable method mistakes — those are handled earlier.
	if _, exists := tc.scopeStack.LookupVariableType(ast.Identifier(pkg)); exists {
		return nil, false
	}
	tc.invalidateAfterUntrustedGoCall(e)
	return []ast.TypeNode{{Ident: ast.TypeVoid}}, true
}

// splitQualified splits a dotted/qualified identifier into path segments.
func splitQualified(id string) []string {
	for i := 0; i < len(id); i++ {
		if id[i] == '.' {
			return []string{id[:i], id[i+1:]}
		}
	}
	return []string{id}
}
