package typechecker

import (
	"forst/internal/ast"
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
type ModuleCrossCall struct {
	CallerPkg   string
	CallerFn    ast.Identifier
	TargetPkg   string
	TargetFn    ast.Identifier
	AmbientKeys map[string]ast.TypeNode
}

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
			AmbientKeys: site.AmbientKeys,
		})
	}
	return out
}

func ambientKeysSatisfySlot(ambient map[string]ast.TypeNode, slot UsableSlot) bool {
	if ambient == nil {
		return false
	}
	_, ok := ambient[string(slot.RootIdent)]
	return ok
}

func addSlotToFunctionMap(m map[ast.Identifier][]UsableSlot, fn ast.Identifier, slot UsableSlot) bool {
	if slot.Key == "" {
		return false
	}
	existing := m[fn]
	for _, s := range existing {
		if s.Key == slot.Key {
			return false
		}
	}
	m[fn] = append(existing, slot)
	return true
}

// PropagateModuleUsablesFixedPoint merges Usables across Forst packages until stable.
func PropagateModuleUsablesFixedPoint(perPkg map[string]map[ast.Identifier][]UsableSlot, calls []ModuleCrossCall) {
	if len(calls) == 0 {
		return
	}
	changed := true
	for changed {
		changed = false
		for _, call := range calls {
			calleeSlots := perPkg[call.TargetPkg][call.TargetFn]
			if len(calleeSlots) == 0 {
				continue
			}
			callerMap := perPkg[call.CallerPkg]
			if callerMap == nil {
				callerMap = make(map[ast.Identifier][]UsableSlot)
				perPkg[call.CallerPkg] = callerMap
			}
			for _, slot := range calleeSlots {
				if ambientKeysSatisfySlot(call.AmbientKeys, slot) {
					continue
				}
				if addSlotToFunctionMap(callerMap, call.CallerFn, slot) {
					changed = true
				}
			}
		}
	}
	for pkg, fnMap := range perPkg {
		for fn, slots := range fnMap {
			perPkg[pkg][fn] = orderUsableSlots(slots)
		}
	}
}
