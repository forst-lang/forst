package typechecker

import "forst/internal/ast"

// IsShapeType reports whether t is Shape or resolves to a shape typedef/alias.
func (tc *TypeChecker) IsShapeType(t ast.TypeNode) bool {
	if t.Ident == ast.TypeShape {
		return true
	}
	if def, ok := tc.Defs[t.Ident]; ok {
		if td, ok := def.(ast.TypeDefNode); ok {
			if _, ok := td.Expr.(ast.TypeDefShapeExpr); ok {
				return true
			}
			if ae, ok := td.Expr.(ast.TypeDefAssertionExpr); ok && ae.Assertion != nil && ae.Assertion.BaseType != nil {
				if *ae.Assertion.BaseType == ast.TypeShape {
					return true
				}
			}
		}
	}
	for _, link := range tc.GetTypeAliasChain(t) {
		if link.Ident == ast.TypeShape {
			return true
		}
	}
	return false
}
