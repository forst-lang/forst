package usablesgraph

import (
	"forst/internal/ast"
)

// IntraPackageEdge is one intra-package call edge with ambient at the call site.
type IntraPackageEdge struct {
	Caller  ast.Identifier
	Callee  ast.Identifier
	Ambient AmbientSnapshot
}

// ModuleCallEdge links a caller function to an exported callee in another Forst package.
type ModuleCallEdge struct {
	CallerPkg string
	CallerFn  ast.Identifier
	TargetPkg string
	TargetFn  ast.Identifier
	Ambient   AmbientSnapshot
}

// SatisfiesAmbient decides whether ambient at a call site satisfies a callee Usable slot.
type SatisfiesAmbient func(slot Slot, ambient map[string]ast.TypeNode) bool

// PropagateIntraPackageFixedPoint merges direct Usables and call edges until stable.
func PropagateIntraPackageFixedPoint(
	slots map[ast.Identifier][]Slot,
	direct map[ast.Identifier]map[string]Slot,
	edges []IntraPackageEdge,
	satisfies SatisfiesAmbient,
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
				if satisfies != nil && satisfies(slot, edge.Ambient) {
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

// PropagateModuleFixedPoint merges Usables across Forst packages until stable.
func PropagateModuleFixedPoint(
	perPkg map[string]map[ast.Identifier][]Slot,
	edges []ModuleCallEdge,
	satisfies SatisfiesAmbient,
) {
	if len(edges) == 0 {
		return
	}
	if satisfies == nil {
		satisfies = AmbientKeyPresent
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
				if satisfies(slot, call.Ambient) {
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
