package typechecker

import (
	"forst/internal/ast"
	"forst/internal/usablesgraph"
)

// crossPackageCallRecord is a qualified import call recorded during single-package check.
type crossPackageCallRecord struct {
	CallerFn    ast.Identifier
	ImportLocal string
	CalleeFn    ast.Identifier
	AmbientKeys map[string]ast.TypeNode
	Span        ast.SourceSpan
}

// ModuleCrossCall links a caller function to an exported callee in another Forst package.
type ModuleCrossCall = usablesgraph.ModuleCallEdge

func (tc *TypeChecker) recordCrossPackageCall(importLocal string, callee ast.Identifier, span ast.SourceSpan) {
	fn := tc.currentFunctionIdent()
	if fn == "" || importLocal == "" || callee == "" {
		return
	}
	tc.crossPackageCallSites = append(tc.crossPackageCallSites, crossPackageCallRecord{
		CallerFn:    fn,
		ImportLocal: importLocal,
		CalleeFn:    callee,
		AmbientKeys: tc.currentMergedAmbient(),
		Span:        span,
	})
}

// BuildModuleCrossCalls resolves recorded import calls to Forst package callees.
func BuildModuleCrossCalls(callerForstPkg string, tc *TypeChecker, importPathToForstPkg map[string]string) []ModuleCrossCall {
	if tc == nil || len(tc.crossPackageCallSites) == 0 {
		return nil
	}
	var out []ModuleCrossCall
	for _, site := range tc.crossPackageCallSites {
		if tc.importPathByLocal == nil {
			continue
		}
		importPath := tc.importPathByLocal[site.ImportLocal]
		targetPkg := importPathToForstPkg[importPath]
		if targetPkg == "" || targetPkg == callerForstPkg {
			continue
		}
		out = append(out, ModuleCrossCall{
			CallerPkg:   callerForstPkg,
			CallerFn:    site.CallerFn,
			TargetPkg:   targetPkg,
			TargetFn:    site.CalleeFn,
			Ambient:     site.AmbientKeys,
		})
	}
	return out
}

// PropagateModuleUsablesFixedPoint merges Usables across Forst packages until stable.
func PropagateModuleUsablesFixedPoint(perPkg map[string]map[ast.Identifier][]UsableSlot, calls []ModuleCrossCall) {
	usablesgraph.PropagateModuleFixedPoint(perPkg, calls, usablesgraph.AmbientKeyPresent)
}
