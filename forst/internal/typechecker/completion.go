package typechecker

import (
	"sort"
	"strings"

	"forst/internal/ast"
)

// VisibleVariableLikeSymbols returns variable and parameter identifiers visible from the
// current scope after RestoreScope, inner scopes shadowing outer names (first occurrence wins).
func (tc *TypeChecker) VisibleVariableLikeSymbols() []ast.Identifier {
	seen := make(map[string]bool)
	var out []ast.Identifier

	scope := tc.CurrentScope()
	for scope != nil {
		for name, sym := range scope.Symbols {
			key := string(name)
			if seen[key] {
				continue
			}
			switch sym.Kind {
			case SymbolVariable:
				seen[key] = true
				out = append(out, name)
			case SymbolParameter:
				if scope.Parent != nil && (scope.IsFunction() || scope.IsTypeGuard()) {
					seen[key] = true
					out = append(out, name)
				}
			default:
				// skip types, functions at file scope, etc.
			}
		}
		scope = scope.Parent
	}

	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// ListFieldNamesForType returns field and method names suitable for member completion after `.`
// for the given type (may be a pointer or assertion-backed type).
func (tc *TypeChecker) ListFieldNamesForType(baseType ast.TypeNode) []string {
	seen := make(map[string]bool)
	var names []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		names = append(names, s)
	}

	t := baseType
	if t.Ident == ast.TypePointer && len(t.TypeParams) > 0 {
		t = t.TypeParams[0]
	}

	t = tc.resolveAliasedType(t)

	if bt, ok := BuiltinTypes[t.Ident]; ok {
		for m := range bt.Methods {
			add(m)
		}
		sort.Strings(names)
		return names
	}

	def, ok := tc.Defs[t.Ident]
	if !ok {
		sort.Strings(names)
		return names
	}

	switch d := def.(type) {
	case ast.TypeDefNode:
		if shapeExpr, ok := d.Expr.(ast.TypeDefShapeExpr); ok {
			for k := range shapeExpr.Shape.Fields {
				add(k)
			}
		}
		if assertionExpr, ok := d.Expr.(ast.TypeDefAssertionExpr); ok && assertionExpr.Assertion != nil {
			merged := tc.resolveShapeFieldsFromAssertion(assertionExpr.Assertion)
			for k := range merged {
				add(k)
			}
		}
	case ast.TypeGuardNode:
		for _, param := range d.Parameters() {
			add(param.GetIdent())
		}
	case *ast.TypeGuardNode:
		for _, param := range d.Parameters() {
			add(param.GetIdent())
		}
	}

	if baseType.Assertion != nil {
		merged := tc.resolveShapeFieldsFromAssertion(baseType.Assertion)
		for k := range merged {
			add(k)
		}
	}

	sort.Strings(names)
	return names
}

// InferExpressionTypeForCompletion runs expression type inference using the current scope chain.
// Used by the LSP after RestoreScope to resolve member completion types.
func (tc *TypeChecker) InferExpressionTypeForCompletion(expr ast.ExpressionNode) ([]ast.TypeNode, error) {
	return tc.inferExpressionType(expr)
}
