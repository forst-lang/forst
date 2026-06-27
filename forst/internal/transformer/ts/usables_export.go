package transformerts

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
)

// ShouldEmitFunctionToTypeScript reports whether fn should appear in TS/sidecar artifacts (ADR-021).
// Public functions with outstanding Usables are omitted when typecheck is relaxed; strict generate errors instead.
func ShouldEmitFunctionToTypeScript(fn ast.FunctionNode, tc *typechecker.TypeChecker) bool {
	if fn.Receiver != nil || !ast.IsPublicExportIdent(fn.Ident.ID) {
		return false
	}
	if tc == nil {
		return true
	}
	return len(tc.FunctionUsables[fn.Ident.ID]) == 0
}
