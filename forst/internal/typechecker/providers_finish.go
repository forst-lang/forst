package typechecker

import (
	"strings"

	"forst/internal/ast"
	"forst/internal/providersgraph"
)

func (tc *TypeChecker) validateSidecarExportable(nodes []ast.Node) error {
	isMain := false
	for _, node := range nodes {
		if p, ok := node.(ast.PackageNode); ok {
			isMain = p.IsMainPackage()
			break
		}
	}
	if !isMain {
		return nil
	}
	for _, node := range nodes {
		fn, ok := node.(ast.FunctionNode)
		if !ok {
			continue
		}
		if fn.Receiver != nil || !ast.IsPublicExportIdent(fn.Ident.ID) {
			continue
		}
		slots := tc.FunctionProviders[fn.Ident.ID]
		if len(slots) == 0 {
			continue
		}
		roots := ProviderRootIdentsFromSlots(slots)
		return diagnosticf(fn.Ident.Span, "providers-sidecar-export",
			"cannot export %s to TypeScript/sidecar: requires %s; wire at a host entry point first",
			fn.Ident.ID, strings.Join(roots, ", "))
	}
	return nil
}

func (tc *TypeChecker) finishProvidersChecking(nodes []ast.Node) error {
	tc.computeProvidersFixedPoint()

	if err := tc.validateIntraCallSites(); err != nil {
		return err
	}

	eng := tc.providersEngine()
	for _, check := range eng.PendingWith {
		tc.checkUnusedWiringKeys(check)
	}

	if err := tc.validateSidecarExportable(nodes); err != nil {
		return err
	}

	if eng.DeferWiringRootCheck {
		return nil
	}

	return tc.validateWiringRoots(nodes)
}

func (tc *TypeChecker) validateWiringRoots(nodes []ast.Node) error {
	if len(nodes) == 0 {
		for id := range tc.Functions {
			if !tc.isWiringRootIdent(id) {
				continue
			}
			if err := tc.validateWiringRootFn(id); err != nil {
				return err
			}
		}
		return nil
	}
	for _, node := range nodes {
		fn, ok := node.(ast.FunctionNode)
		if !ok {
			continue
		}
		if !tc.isWiringRoot(fn) {
			continue
		}
		if err := tc.validateWiringRootFn(fn.Ident.ID); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TypeChecker) validateWiringRootFn(id ast.Identifier) error {
	slots := tc.FunctionProviders[id]
	if len(slots) == 0 {
		return nil
	}
	names := make([]string, len(slots))
	for i, s := range slots {
		names[i] = string(s.RootIdent)
	}
	chain := tc.buildWiringRootObligationChain(id, slots)
	var span ast.SourceSpan
	if sig, ok := tc.Functions[id]; ok {
		span = sig.Ident.Span
	}
	return diagnosticfRelated(span, "providers-unsatisfied", tc.providersWiringRootRelated(id),
		"%s requires %s; not supplied at wiring root\n  required by: %s%s",
		id, strings.Join(names, ", "), chain, providersFixItHint(names))
}

func (tc *TypeChecker) providersWiringRootRelated(rootFn ast.Identifier) []RelatedDiagnostic {
	var related []RelatedDiagnostic
	seen := make(map[ast.Identifier]struct{})
	if tc.providers == nil {
		return related
	}
	for _, edge := range tc.providers.CallEdges {
		if edge.CallerFn != rootFn || edge.ImportLocal != "" {
			continue
		}
		if len(tc.FunctionProviders[edge.CalleeFn]) == 0 {
			continue
		}
		if _, ok := seen[edge.CalleeFn]; ok {
			continue
		}
		seen[edge.CalleeFn] = struct{}{}
		if sig, ok := tc.Functions[edge.CalleeFn]; ok && sig.Ident.Span.IsSet() {
			related = append(related, RelatedDiagnostic{
				Msg:  string(edge.CalleeFn) + " declares Providers requirements",
				Span: sig.Ident.Span,
			})
		}
	}
	return related
}

// ValidateModuleProviders runs post-module-merge validation including cross-package call sites.
func ValidateModuleProviders(
	callerForstPkg string,
	tc *TypeChecker,
	importPathToForstPkg map[string]string,
	perPkgSlots map[string]map[ast.Identifier][]ProviderSlot,
) error {
	if tc == nil || tc.providers == nil {
		return nil
	}
	for _, edge := range CrossPackageCallEdges(callerForstPkg, tc, importPathToForstPkg) {
		targetSlots := perPkgSlots[edge.CalleePkg][edge.CalleeFn]
		if len(targetSlots) == 0 {
			continue
		}
		if scopeSatisfiesAllSlots(tc, edge.Scope, targetSlots) {
			continue
		}
		if !tc.isWiringRootIdent(edge.CallerFn) && callerForwardsSlots(tc, edge.CallerFn, targetSlots) {
			continue
		}
		if err := checkCrossPackageCallSatisfied(tc, edge, targetSlots); err != nil {
			return err
		}
	}
	if err := tc.validateIntraCallSites(); err != nil {
		return err
	}
	return tc.validateWiringRoots(nil)
}

func scopeSatisfiesAllSlots(tc *TypeChecker, scope map[string]ast.TypeNode, slots []ProviderSlot) bool {
	for _, slot := range slots {
		if !tc.scopeSatisfiesSlot(slot, scope) {
			return false
		}
	}
	return true
}

func callerForwardsSlots(tc *TypeChecker, caller ast.Identifier, calleeSlots []ProviderSlot) bool {
	callerSlots := tc.FunctionProviders[caller]
	callerKeys := make(map[string]struct{})
	for _, slot := range callerSlots {
		callerKeys[slot.Key] = struct{}{}
	}
	for _, slot := range calleeSlots {
		if _, ok := callerKeys[slot.Key]; !ok {
			return false
		}
	}
	return true
}

func checkCrossPackageCallSatisfied(tc *TypeChecker, edge providersgraph.CallEdge, calleeSlots []ProviderSlot) error {
	var missing []string
	for _, slot := range calleeSlots {
		if !tc.scopeSatisfiesSlot(slot, edge.Scope) {
			missing = append(missing, string(slot.RootIdent))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	calleeLabel := string(edge.CalleeFn)
	if edge.CalleePkg != "" {
		calleeLabel = edge.CalleePkg + "." + calleeLabel
	}
	parts := []string{string(edge.CallerFn), calleeLabel}
	for _, root := range missing {
		suffix := tc.walkToDirectProviderUse(edge.CalleePkg, edge.CalleeFn, root)
		for _, step := range suffix {
			if step.label == calleeLabel || step.label == string(edge.CallerFn) {
				continue
			}
			parts = append(parts, step.label)
		}
		parts = append(parts, root)
	}
	chain := strings.Join(parts, " → ")
	return diagnosticfRelated(edge.Span, "providers-unsatisfied", nil,
		"%s requires %s; not supplied\n  required by: %s%s",
		calleeLabel, strings.Join(missing, ", "), chain, providersFixItHint(missing))
}
