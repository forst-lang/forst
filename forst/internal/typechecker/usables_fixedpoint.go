package typechecker

import (
	"forst/internal/ast"
	"forst/internal/usablesgraph"

	logrus "github.com/sirupsen/logrus"
)

// UsableRootIdentsFromSlots returns ordered root contract idents for discovery JSON / LSP.
func UsableRootIdentsFromSlots(slots []UsableSlot) []string {
	return usablesgraph.RootIdentsFromSlots(slots)
}

func orderUsableSlots(slots []UsableSlot) []UsableSlot {
	return usablesgraph.OrderSlots(slots)
}

func (tc *TypeChecker) slotsFromMap(m map[string]UsableSlot) []UsableSlot {
	return usablesgraph.SlotsFromDirectMap(m)
}

func (tc *TypeChecker) addUsableSlotToFunction(fn ast.Identifier, slot UsableSlot) bool {
	return usablesgraph.AddSlotToFunction(tc.FunctionUsables, fn, slot)
}

func (tc *TypeChecker) computeUsablesFixedPoint() {
	g := usablesgraph.New()
	for fn, direct := range tc.functionDirectUsables {
		g.SetDirect(fn, direct)
	}
	for caller, sites := range tc.functionCallSites {
		for _, site := range sites {
			g.AddIntraCall(caller, site.Callee, site.AmbientKeys)
		}
	}
	g.ComputeIntraFixedPoint(tc.ambientSatisfiesSlot)
	tc.FunctionUsables = g.AllSlots()

	tc.log.WithFields(logrus.Fields{
		"function": "computeUsablesFixedPoint",
		"count":    len(tc.FunctionUsables),
	}).Debug("Computed FunctionUsables fixed point")
}

func (tc *TypeChecker) callerForwardsUsables(caller, callee ast.Identifier) bool {
	calleeSlots := tc.FunctionUsables[callee]
	if len(calleeSlots) == 0 {
		return true
	}
	callerKeys := make(map[string]struct{})
	for _, slot := range tc.FunctionUsables[caller] {
		callerKeys[slot.Key] = struct{}{}
	}
	for _, slot := range calleeSlots {
		if _, ok := callerKeys[slot.Key]; !ok {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) isWiringRootIdent(id ast.Identifier) bool {
	return ast.IsUsablesWiringRoot(id, tc.paramTypesForIdent(id))
}

func (tc *TypeChecker) paramTypesForIdent(id ast.Identifier) []ast.TypeNode {
	sig, ok := tc.Functions[id]
	if !ok {
		return nil
	}
	types := make([]ast.TypeNode, 0, len(sig.Parameters))
	for _, param := range sig.Parameters {
		types = append(types, param.Type)
	}
	return types
}

func (tc *TypeChecker) validateCallSite(caller ast.Identifier, site callSiteRecord) error {
	if len(tc.FunctionUsables[site.Callee]) == 0 {
		return nil
	}
	if tc.ambientSatisfiesAllSlots(site.AmbientKeys, tc.FunctionUsables[site.Callee]) {
		return nil
	}
	if !tc.isWiringRootIdent(caller) && tc.callerForwardsUsables(caller, site.Callee) {
		return nil
	}
	return tc.checkCallUsablesSatisfied(caller, site.Callee, site.AmbientKeys, site.Span)
}

// UsablesGraph returns the intra-package Usables graph after inference (ADR-036).
func (tc *TypeChecker) UsablesGraph() *usablesgraph.Graph {
	g := usablesgraph.New()
	for fn, direct := range tc.functionDirectUsables {
		g.SetDirect(fn, direct)
	}
	for caller, sites := range tc.functionCallSites {
		for _, site := range sites {
			g.AddIntraCall(caller, site.Callee, site.AmbientKeys)
		}
	}
	g.ComputeIntraFixedPoint(tc.ambientSatisfiesSlot)
	return g
}
