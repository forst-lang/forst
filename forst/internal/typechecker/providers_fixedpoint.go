package typechecker

import (
	"forst/internal/ast"
	"forst/internal/providersgraph"

	logrus "github.com/sirupsen/logrus"
)

// ProviderRootIdentsFromSlots returns ordered root contract idents for discovery JSON / LSP.
func ProviderRootIdentsFromSlots(slots []ProviderSlot) []string {
	return providersgraph.RootIdentsFromSlots(slots)
}

func orderProviderSlots(slots []ProviderSlot) []ProviderSlot {
	return providersgraph.OrderSlots(slots)
}

func (tc *TypeChecker) slotsFromMap(m map[string]ProviderSlot) []ProviderSlot {
	return providersgraph.SlotsFromDirectMap(m)
}

func (tc *TypeChecker) addProviderSlotToFunction(fn ast.Identifier, slot ProviderSlot) bool {
	return providersgraph.AddSlotToFunction(tc.providersEngine().Slots, fn, slot)
}

func (tc *TypeChecker) computeProvidersFixedPoint() {
	eng := tc.providersEngine()
	g := providersgraph.New()
	for fn, direct := range eng.Direct {
		g.SetDirect(fn, direct)
	}
	for _, edge := range providersgraph.IntraEdgesFromCallEdges(eng.CallEdges) {
		g.AddIntraCall(edge.Caller, edge.Callee, edge.ProviderScope)
	}
	g.ComputeIntraFixedPoint(tc.scopeSatisfiesSlot)
	eng.Slots = g.AllSlots()
	tc.FunctionProviders = eng.Slots

	tc.log.WithFields(logrus.Fields{
		"function": "computeProvidersFixedPoint",
		"count":    len(eng.Slots),
	}).Debug("Computed FunctionProviders fixed point")
}

func (tc *TypeChecker) callerForwardsProviders(caller, callee ast.Identifier) bool {
	calleeSlots := tc.FunctionProviders[callee]
	if len(calleeSlots) == 0 {
		return true
	}
	callerKeys := make(map[string]struct{})
	for _, slot := range tc.FunctionProviders[caller] {
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
	return ast.IsProvidersWiringRoot(id, tc.paramTypesForIdent(id))
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

func (tc *TypeChecker) validateCallSite(caller ast.Identifier, edge providersgraph.CallEdge) error {
	if len(tc.FunctionProviders[edge.CalleeFn]) == 0 {
		return nil
	}
	if tc.scopeSatisfiesAllSlots(edge.Scope, tc.FunctionProviders[edge.CalleeFn]) {
		return nil
	}
	if !tc.isWiringRootIdent(caller) && tc.callerForwardsProviders(caller, edge.CalleeFn) {
		return nil
	}
	return tc.checkCallProvidersSatisfied(caller, edge.CalleeFn, edge.Scope, edge.Span)
}

func (tc *TypeChecker) validateIntraCallSites() error {
	if tc.providers == nil {
		return nil
	}
	for _, edge := range tc.providers.CallEdges {
		if edge.ImportLocal != "" {
			continue
		}
		if err := tc.validateCallSite(edge.CallerFn, edge); err != nil {
			return err
		}
	}
	return nil
}

// BuildGraph returns the intra-package Providers graph from engine state.
func (tc *TypeChecker) BuildGraph() *providersgraph.Graph {
	eng := tc.providersEngine()
	g := providersgraph.New()
	for fn, direct := range eng.Direct {
		g.SetDirect(fn, direct)
	}
	for _, edge := range providersgraph.IntraEdgesFromCallEdges(eng.CallEdges) {
		g.AddIntraCall(edge.Caller, edge.Callee, edge.ProviderScope)
	}
	return g
}
