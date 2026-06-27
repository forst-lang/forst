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
	return providersgraph.AddSlotToFunction(tc.FunctionProviders, fn, slot)
}

func (tc *TypeChecker) computeProvidersFixedPoint() {
	g := providersgraph.New()
	for fn, direct := range tc.functionDirectProviders {
		g.SetDirect(fn, direct)
	}
	for caller, sites := range tc.functionCallSites {
		for _, site := range sites {
			g.AddIntraCall(caller, site.Callee, site.ScopeKeys)
		}
	}
	g.ComputeIntraFixedPoint(tc.scopeSatisfiesSlot)
	tc.FunctionProviders = g.AllSlots()

	tc.log.WithFields(logrus.Fields{
		"function": "computeProvidersFixedPoint",
		"count":    len(tc.FunctionProviders),
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

func (tc *TypeChecker) validateCallSite(caller ast.Identifier, site callSiteRecord) error {
	if len(tc.FunctionProviders[site.Callee]) == 0 {
		return nil
	}
	if tc.scopeSatisfiesAllSlots(site.ScopeKeys, tc.FunctionProviders[site.Callee]) {
		return nil
	}
	if !tc.isWiringRootIdent(caller) && tc.callerForwardsProviders(caller, site.Callee) {
		return nil
	}
	return tc.checkCallProvidersSatisfied(caller, site.Callee, site.ScopeKeys, site.Span)
}

// ProvidersGraph returns the intra-package Providers graph after inference (ADR-036).
func (tc *TypeChecker) ProvidersGraph() *providersgraph.Graph {
	g := providersgraph.New()
	for fn, direct := range tc.functionDirectProviders {
		g.SetDirect(fn, direct)
	}
	for caller, sites := range tc.functionCallSites {
		for _, site := range sites {
			g.AddIntraCall(caller, site.Callee, site.ScopeKeys)
		}
	}
	g.ComputeIntraFixedPoint(tc.scopeSatisfiesSlot)
	return g
}
