package typechecker

import (
	"forst/internal/ast"
	"forst/internal/providersgraph"
)

// ModuleCrossCall links a caller function to an exported callee in another Forst package.
type ModuleCrossCall = providersgraph.ModuleCallEdge

func (tc *TypeChecker) recordCrossPackageCall(importLocal string, callee ast.Identifier, span ast.SourceSpan) {
	fn := tc.currentFunctionIdent()
	if fn == "" || importLocal == "" || callee == "" {
		return
	}
	eng := tc.providersEngine()
	eng.CallEdges = append(eng.CallEdges, providersgraph.CallEdge{
		CallerFn:    fn,
		CalleeFn:    callee,
		ImportLocal: importLocal,
		Scope:       tc.currentMergedScope(),
		Span:        span,
	})
}

// BuildModuleCrossCalls resolves recorded cross-package import calls to Forst package callees.
func BuildModuleCrossCalls(callerForstPkg string, tc *TypeChecker, importPathToForstPkg map[string]string) []ModuleCrossCall {
	if tc == nil || tc.providers == nil {
		return nil
	}
	var out []ModuleCrossCall
	for _, edge := range tc.providers.CallEdges {
		if edge.ImportLocal == "" {
			continue
		}
		importPath := tc.importPathForLocal(edge.ImportLocal)
		if importPath == "" {
			continue
		}
		targetPkg := importPathToForstPkg[importPath]
		if targetPkg == "" || targetPkg == callerForstPkg {
			continue
		}
		out = append(out, ModuleCrossCall{
			CallerPkg:     callerForstPkg,
			CallerFn:      edge.CallerFn,
			TargetPkg:     targetPkg,
			TargetFn:      edge.CalleeFn,
			ProviderScope: edge.Scope,
		})
	}
	return out
}

// PropagateModuleProvidersFixedPoint merges Providers across Forst packages until stable.
func PropagateModuleProvidersFixedPoint(perPkg map[string]map[ast.Identifier][]ProviderSlot, calls []ModuleCrossCall, satisfies providersgraph.SatisfiesProviderScope) {
	providersgraph.PropagateModuleFixedPoint(perPkg, calls, satisfies)
}

// ModuleSatisfiesFromTypeChecker returns a type-aware satisfaction predicate for module propagation.
func ModuleSatisfiesFromTypeChecker(tc *TypeChecker) providersgraph.SatisfiesProviderScope {
	if tc == nil {
		return providersgraph.ProviderScopeKeyPresent
	}
	return tc.scopeSatisfiesSlot
}

// CrossPackageCallEdges returns cross-package call edges with resolved target packages.
func CrossPackageCallEdges(callerForstPkg string, tc *TypeChecker, importPathToForstPkg map[string]string) []providersgraph.CallEdge {
	if tc == nil || tc.providers == nil {
		return nil
	}
	var out []providersgraph.CallEdge
	for _, edge := range tc.providers.CallEdges {
		if edge.ImportLocal == "" {
			continue
		}
		importPath := tc.importPathForLocal(edge.ImportLocal)
		if importPath == "" {
			continue
		}
		targetPkg := importPathToForstPkg[importPath]
		if targetPkg == "" || targetPkg == callerForstPkg {
			continue
		}
		out = append(out, providersgraph.CallEdge{
			CallerPkg: callerForstPkg,
			CallerFn:  edge.CallerFn,
			CalleePkg: targetPkg,
			CalleeFn:  edge.CalleeFn,
			Scope:     edge.Scope,
			Span:      edge.Span,
		})
	}
	return out
}
