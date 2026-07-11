package typechecker

import "forst/internal/nodeinterop"

// NodeCallTarget describes a resolved opted-in TypeScript call target.
type NodeCallTarget struct {
	ModuleID   string
	ExportName string
	Kind       string
}

// NodeCallTarget returns compile-time facts for pkgLocal.exportName when it is a node import call.
func (tc *TypeChecker) NodeCallTarget(pkgLocal, exportName string) (NodeCallTarget, bool) {
	if tc == nil {
		return NodeCallTarget{}, false
	}
	mod, ok := tc.nodeModuleForLocal(pkgLocal)
	if !ok {
		return NodeCallTarget{}, false
	}
	if tc.nodeIndexResolver == nil {
		return NodeCallTarget{}, false
	}
	_, _, kind, err := tc.nodeIndexResolver.ExportSignature(mod.ModuleID, exportName)
	if err != nil {
		return NodeCallTarget{}, false
	}
	return NodeCallTarget{
		ModuleID:   mod.ModuleID,
		ExportName: exportName,
		Kind:       kind,
	}, true
}

// NodeModuleForLocal exposes node import binding lookup for codegen.
func (tc *TypeChecker) NodeModuleForLocal(local string) (moduleID string, ok bool) {
	mod, ok := tc.nodeModuleForLocal(local)
	if !ok {
		return "", false
	}
	return mod.ModuleID, true
}

// NodeExportKindFunction is the sync export kind constant for codegen checks.
const NodeExportKindFunction = nodeinterop.ExportKindFunction

// NodeExportKindGenerator is the sync generator export kind.
const NodeExportKindGenerator = nodeinterop.ExportKindGenerator

// NodeExportKindAsyncGenerator is the async generator export kind.
const NodeExportKindAsyncGenerator = nodeinterop.ExportKindAsyncGenerator

// NodeExportKindAsyncFunction is the async export kind constant for codegen checks.
const NodeExportKindAsyncFunction = nodeinterop.ExportKindAsyncFunction
