package typechecker

import "forst/internal/ast"

// Test utilities for creating AST nodes in tests - now using centralized ast package

// typeIdentPtr creates a pointer to a TypeIdent
func typeIdentPtr(s string) *ast.TypeIdent {
	ti := ast.TypeIdent(s)
	return &ti
}
