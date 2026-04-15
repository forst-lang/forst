package discovery

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
)

// typeToString converts an AST type to a string representation
func (d *Discoverer) typeToString(t ast.TypeNode) string {
	return t.String()
}

func (d *Discoverer) applyResultDiscoveryMetadata(fn *FunctionInfo, rt ast.TypeNode, tc *typechecker.TypeChecker) {
	if !rt.IsResultType() || len(rt.TypeParams) < 2 {
		return
	}
	fn.IsResult = true
	fn.ResultSuccessType = d.resolveTypeName(rt.TypeParams[0], tc)
	fn.ResultFailureType = d.resolveTypeName(rt.TypeParams[1], tc)
	// Result(S, F) is one Forst return slot but lowers to (S, error) in Go; the executor wrapper
	// must use `result, err := fn(...)` like any multi-return Go function.
	fn.HasMultipleReturns = true
}

// resolveTypeName converts an AST type to a string representation using the typechecker
func (d *Discoverer) resolveTypeName(t ast.TypeNode, tc *typechecker.TypeChecker) string {
	if tc != nil {
		name, err := tc.GetAliasedTypeName(t, typechecker.GetAliasedTypeNameOptions{AllowStructuralAlias: true})
		if err == nil {
			return name
		}
	}
	return t.String()
}

// determineInputType determines the input type for API purposes
func (d *Discoverer) determineInputType(params []ParameterInfo) string {
	if len(params) == 0 {
		return "void"
	}
	if len(params) == 1 {
		return params[0].Type
	}
	return "json" // Multiple parameters use JSON
}
