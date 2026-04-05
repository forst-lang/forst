package typechecker

import "forst/internal/ast"

// Test utilities for creating AST nodes in tests - now using centralized ast package

// typeIdentPtr creates a pointer to a TypeIdent
func typeIdentPtr(s string) *ast.TypeIdent {
	ti := ast.TypeIdent(s)
	return &ti
}

// registerTypeGuardExpectsInt installs a minimal type guard definition expecting an Int subject (for ensure tests).
func registerTypeGuardExpectsInt(tc *TypeChecker, name ast.TypeIdent) {
	tc.Defs[name] = ast.TypeGuardNode{
		Ident: ast.Identifier(name),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "n"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{},
	}
}
