package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

type obligationStep struct {
	pkg   string
	fn    ast.Identifier
	label string // display label (may include import local for cross-package)
}

// BuildProvidersObligationChain formats a transitive chain from rootFn to providerRoot
// (e.g. TestHandle → Handle → alpha.LogExpiry → Logger).
func (tc *TypeChecker) BuildProvidersObligationChain(rootFn ast.Identifier, providerRoot string) string {
	startPkg := tc.ForstPackage()
	path := tc.walkToDirectProviderUse(startPkg, rootFn, providerRoot)
	if len(path) == 0 {
		return string(rootFn) + " → " + providerRoot
	}
	parts := make([]string, 0, len(path)+1)
	for _, step := range path {
		parts = append(parts, step.label)
	}
	if parts[len(parts)-1] != providerRoot {
		parts = append(parts, providerRoot)
	}
	return strings.Join(parts, " → ")
}

// buildCallSiteObligationChain formats caller → … → provider roots for missing slots at a call site.
func (tc *TypeChecker) buildCallSiteObligationChain(caller, callee ast.Identifier, missingRoots []string) string {
	parts := []string{string(caller), string(callee)}
	seen := map[string]struct{}{string(caller): {}, string(callee): {}}
	for _, root := range missingRoots {
		suffix := tc.walkToDirectProviderUse(tc.ForstPackage(), callee, root)
		for _, step := range suffix {
			if step.label == string(callee) {
				continue
			}
			if _, ok := seen[step.label]; ok {
				continue
			}
			seen[step.label] = struct{}{}
			parts = append(parts, step.label)
		}
		if _, ok := seen[root]; !ok {
			seen[root] = struct{}{}
			parts = append(parts, root)
		}
	}
	return strings.Join(parts, " → ")
}

// buildWiringRootObligationChain lists each missing provider root with a transitive chain.
func (tc *TypeChecker) buildWiringRootObligationChain(rootFn ast.Identifier, slots []ProviderSlot) string {
	if len(slots) == 0 {
		return string(rootFn)
	}
	if len(slots) == 1 {
		return tc.BuildProvidersObligationChain(rootFn, string(slots[0].RootIdent))
	}
	chains := make([]string, 0, len(slots))
	for _, slot := range slots {
		chains = append(chains, tc.BuildProvidersObligationChain(rootFn, string(slot.RootIdent)))
	}
	return strings.Join(chains, "; ")
}

func (tc *TypeChecker) walkToDirectProviderUse(startPkg string, startFn ast.Identifier, providerRoot string) []obligationStep {
	if tc.directUseOfRoot(startPkg, startFn, providerRoot) {
		return []obligationStep{{pkg: startPkg, fn: startFn, label: string(startFn)}}
	}

	type queueItem struct {
		pkg  string
		fn   ast.Identifier
		path []obligationStep
	}
	visited := make(map[string]struct{})
	key := func(pkg string, fn ast.Identifier) string { return pkg + "\x00" + string(fn) }
	startKey := key(startPkg, startFn)
	visited[startKey] = struct{}{}

	queue := []queueItem{{
		pkg:  startPkg,
		fn:   startFn,
		path: []obligationStep{{pkg: startPkg, fn: startFn, label: string(startFn)}},
	}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, edge := range tc.outgoingCallEdges(cur.pkg, cur.fn) {
			nextKey := key(edge.targetPkg, edge.targetFn)
			if _, ok := visited[nextKey]; ok {
				continue
			}
			visited[nextKey] = struct{}{}
			nextPath := append(append([]obligationStep(nil), cur.path...), obligationStep{
				pkg:   edge.targetPkg,
				fn:    edge.targetFn,
				label: edge.label,
			})
			if tc.directUseOfRoot(edge.targetPkg, edge.targetFn, providerRoot) {
				return nextPath
			}
			queue = append(queue, queueItem{pkg: edge.targetPkg, fn: edge.targetFn, path: nextPath})
		}
	}
	return nil
}

type outgoingEdge struct {
	targetPkg string
	targetFn  ast.Identifier
	label     string
}

func (tc *TypeChecker) outgoingCallEdges(pkg string, fn ast.Identifier) []outgoingEdge {
	tci := tc.tcForPkg(pkg)
	if tci == nil || tci.providers == nil {
		return nil
	}
	var out []outgoingEdge
	for _, edge := range tci.providers.CallEdges {
		if edge.CallerFn != fn {
			continue
		}
		if edge.ImportLocal != "" {
			importPath, ok := tci.ImportPathForLocal(edge.ImportLocal)
			targetPkg := pkg
			if tc.moduleResult != nil && ok {
				if mapped := tc.moduleResult.ImportPathToForstPkg()[importPath]; mapped != "" {
					targetPkg = mapped
				}
			}
			label := edge.ImportLocal + "." + string(edge.CalleeFn)
			out = append(out, outgoingEdge{targetPkg: targetPkg, targetFn: edge.CalleeFn, label: label})
			continue
		}
		out = append(out, outgoingEdge{targetPkg: pkg, targetFn: edge.CalleeFn, label: string(edge.CalleeFn)})
	}
	return out
}

func (tc *TypeChecker) directUseOfRoot(pkg string, fn ast.Identifier, root string) bool {
	tci := tc.tcForPkg(pkg)
	if tci == nil || tci.providers == nil {
		return false
	}
	m := tci.providers.Direct[fn]
	if m == nil {
		return false
	}
	_, ok := m[root]
	return ok
}

func (tc *TypeChecker) tcForPkg(pkg string) *TypeChecker {
	if tc.moduleResult != nil {
		if sibling := tc.moduleResult.ForstPackageTypeChecker(pkg); sibling != nil {
			return sibling
		}
	}
	if pkg == "" || pkg == tc.ForstPackage() {
		return tc
	}
	return nil
}

const providersUnsatisfiedHint = "\n  hint: add %s to the with { … } block or ciProviders bundle"

func providersFixItHint(roots []string) string {
	if len(roots) == 0 {
		return ""
	}
	return fmt.Sprintf(providersUnsatisfiedHint, strings.Join(roots, ", "))
}
