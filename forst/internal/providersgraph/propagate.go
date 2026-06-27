package providersgraph

import (
	"forst/internal/ast"
)

// IntraPackageEdge is one intra-package call edge with scope at the call site.
type IntraPackageEdge struct {
	Caller  ast.Identifier
	Callee  ast.Identifier
	ProviderScope ProviderScopeSnapshot
}

// ModuleCallEdge links a caller function to an exported callee in another Forst package.
type ModuleCallEdge struct {
	CallerPkg string
	CallerFn  ast.Identifier
	TargetPkg string
	TargetFn  ast.Identifier
	ProviderScope   ProviderScopeSnapshot
}

// SatisfiesProviderScope decides whether scope at a call site satisfies a callee Provider slot.
type SatisfiesProviderScope func(slot Slot, scope map[string]ast.TypeNode) bool

// PropagateIntraPackageFixedPoint merges direct Providers and call edges until stable.
func PropagateIntraPackageFixedPoint(
	slots map[ast.Identifier][]Slot,
	direct map[ast.Identifier]map[string]Slot,
	edges []IntraPackageEdge,
	satisfies SatisfiesProviderScope,
) {
	if slots == nil {
		slots = make(map[ast.Identifier][]Slot)
	}
	for fn, m := range direct {
		slots[fn] = SlotsFromDirectMap(m)
	}

	changed := true
	for changed {
		changed = false
		for _, edge := range edges {
			for _, slot := range slots[edge.Callee] {
				if satisfies != nil && satisfies(slot, edge.ProviderScope) {
					continue
				}
				if AddSlotToFunction(slots, edge.Caller, slot) {
					changed = true
				}
			}
		}
	}

	for fn, fnSlots := range slots {
		slots[fn] = OrderSlots(fnSlots)
	}
}

// PropagateModuleFixedPoint merges Providers across Forst packages until stable.
func PropagateModuleFixedPoint(
	perPkg map[string]map[ast.Identifier][]Slot,
	edges []ModuleCallEdge,
	satisfies SatisfiesProviderScope,
) {
	if len(edges) == 0 {
		return
	}
	if satisfies == nil {
		satisfies = ProviderScopeKeyPresent
	}

	changed := true
	for changed {
		changed = false
		for _, call := range edges {
			calleeSlots := perPkg[call.TargetPkg][call.TargetFn]
			if len(calleeSlots) == 0 {
				continue
			}
			callerMap := perPkg[call.CallerPkg]
			if callerMap == nil {
				callerMap = make(map[ast.Identifier][]Slot)
				perPkg[call.CallerPkg] = callerMap
			}
			for _, slot := range calleeSlots {
				if satisfies(slot, call.ProviderScope) {
					continue
				}
				if AddSlotToFunction(callerMap, call.CallerFn, slot) {
					changed = true
				}
			}
		}
	}

	for pkg, fnMap := range perPkg {
		for fn, fnSlots := range fnMap {
			perPkg[pkg][fn] = OrderSlots(fnSlots)
		}
	}
}
