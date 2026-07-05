package typechecker

import (
	"forst/internal/ast"
	"forst/internal/providersgraph"
)

// ProvidersEngine owns all Providers inference state for one package check.
type ProvidersEngine struct {
	Direct               map[ast.Identifier]map[string]ProviderSlot
	CallEdges            []providersgraph.CallEdge
	ScopeStack           []providersgraph.ProviderScope
	PendingWith          []pendingWithCheck
	Slots                map[ast.Identifier][]ProviderSlot
	KnownRoots           map[string]ast.TypeNode
	DeferWiringRootCheck bool
	ForstPackage         string
}

func newProvidersEngine() *ProvidersEngine {
	return &ProvidersEngine{
		Direct:     make(map[ast.Identifier]map[string]ProviderSlot),
		KnownRoots: make(map[string]ast.TypeNode),
	}
}

func (tc *TypeChecker) providersEngine() *ProvidersEngine {
	if tc.providers == nil {
		tc.providers = newProvidersEngine()
	}
	return tc.providers
}

func (tc *TypeChecker) initProvidersInference() {
	var deferCheck bool
	var forstPkg string
	if tc.providers != nil {
		deferCheck = tc.providers.DeferWiringRootCheck
		forstPkg = tc.providers.ForstPackage
	}
	tc.providers = newProvidersEngine()
	tc.providers.DeferWiringRootCheck = deferCheck
	tc.providers.ForstPackage = forstPkg
	tc.Warnings = nil
}

// SetFunctionProviders writes merged slots back after module-level propagation.
func (tc *TypeChecker) SetFunctionProviders(slots map[ast.Identifier][]ProviderSlot) {
	eng := tc.providersEngine()
	if slots == nil {
		eng.Slots = make(map[ast.Identifier][]ProviderSlot)
	} else {
		eng.Slots = slots
	}
	tc.FunctionProviders = eng.Slots
}

// SetDeferProvidersWiringRootCheck defers wiring-root validation until after module merge.
func (tc *TypeChecker) SetDeferProvidersWiringRootCheck(deferCheck bool) {
	tc.providersEngine().DeferWiringRootCheck = deferCheck
}

// SetSamePackageGoImportPath sets the Go import path for mixed .go + .ft packages (e.g. lichtung/internal/graph).
func (tc *TypeChecker) SetSamePackageGoImportPath(importPath string) {
	tc.samePackageGoImportPath = importPath
}

// SetForstPackage records the Forst package name for cross-package edge resolution.
func (tc *TypeChecker) SetForstPackage(name string) {
	tc.providersEngine().ForstPackage = name
}

// ModuleResultView is the cross-package metadata typecheckers need during module check.
type ModuleResultView interface {
	ImportPathToForstPkg() map[string]string
	ForstPackageTypeChecker(pkg string) *TypeChecker
}

// SetModuleResult attaches module-level context for Forst sibling import resolution.
func (tc *TypeChecker) SetModuleResult(m ModuleResultView) {
	tc.moduleResult = moduleResultAdapter{m}
}

type moduleResultAdapter struct{ ModuleResultView }

func (a moduleResultAdapter) ForstPackageSlots(pkg string) map[ast.Identifier][]ProviderSlot {
	tc := a.ForstPackageTypeChecker(pkg)
	if tc == nil {
		return nil
	}
	return tc.FunctionProviders
}

// ForstPackage returns the Forst package name when set during module check.
func (tc *TypeChecker) ForstPackage() string {
	if tc.providers == nil {
		return ""
	}
	return tc.providers.ForstPackage
}

func (tc *TypeChecker) importPathToForstPkgMap() map[string]string {
	if tc.moduleResult == nil {
		return nil
	}
	return tc.moduleResult.ImportPathToForstPkg()
}

// resolveForstSiblingCall resolves alpha.Foo when alpha is a Forst package in the same module.
func (tc *TypeChecker) resolveForstSiblingCall(importLocal, funcName string, e ast.FunctionCallNode, _ [][]ast.TypeNode) ([]ast.TypeNode, error) {
	importMap := tc.importPathToForstPkgMap()
	if importMap == nil {
		return nil, nil
	}
	importPath, ok := tc.ImportPathForLocal(importLocal)
	if !ok {
		return nil, nil
	}
	targetPkg := importMap[importPath]
	if targetPkg == "" || targetPkg == tc.ForstPackage() {
		return nil, nil
	}
	if tc.moduleResult == nil {
		return nil, nil
	}
	siblingTC := tc.moduleResult.ForstPackageTypeChecker(targetPkg)
	if siblingTC == nil {
		return nil, nil
	}
	callee := ast.Identifier(funcName)
	sig, ok := siblingTC.Functions[callee]
	if !ok {
		return nil, nil
	}
	callSpan := e.CallSpan
	if !callSpan.IsSet() {
		callSpan = e.Function.Span
	}
	tc.recordCrossPackageCall(importLocal, callee, callSpan)
	if len(sig.ReturnTypes) == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil
	}
	return sig.ReturnTypes, nil
}

