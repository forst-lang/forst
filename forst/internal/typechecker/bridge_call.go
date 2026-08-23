package typechecker

import (
	"fmt"
	"path/filepath"
	"strings"

	"forst/internal/ast"
	"forst/internal/bridgeinterop"
	"forst/internal/ftconfig"
)

// BridgeCallTarget describes a resolved opted-in TypeScript call target.
type BridgeCallTarget struct {
	ModuleID       string // runtime module id passed to bridgert
	SourceModuleID string // source module id for the TypeScript index
	ExportName     string
	Kind           string
}

// IndexModuleID returns the module id used for TypeScript index lookups.
func (t BridgeCallTarget) IndexModuleID() string {
	if t.SourceModuleID != "" {
		return t.SourceModuleID
	}
	return t.ModuleID
}

// BridgeExportParamTypes returns Forst parameter types from the loaded TS index for an export.
func (tc *TypeChecker) BridgeExportParamTypes(moduleID, exportName string) ([]ast.TypeNode, error) {
	if tc == nil || tc.nodeIndexResolver == nil {
		return nil, fmt.Errorf("node index resolver not initialized")
	}
	params, _, _, err := tc.nodeIndexResolver.ExportSignature(moduleID, exportName)
	return params, err
}

// BridgeCallTarget returns compile-time facts for pkgLocal.exportName when it is a JS import call.
func (tc *TypeChecker) BridgeCallTarget(pkgLocal, exportName string) (BridgeCallTarget, bool) {
	if tc == nil {
		return BridgeCallTarget{}, false
	}
	mod, ok := tc.nodeModuleForLocal(pkgLocal)
	if !ok {
		return BridgeCallTarget{}, false
	}
	if tc.nodeIndexResolver == nil {
		return BridgeCallTarget{}, false
	}
	_, _, kind, err := tc.nodeIndexResolver.ExportSignature(mod.ModuleID, exportName)
	if err != nil {
		return BridgeCallTarget{}, false
	}
	return BridgeCallTarget{
		ModuleID:       tc.bridgeCallRuntimeModuleID(mod.ModuleID),
		SourceModuleID: mod.ModuleID,
		ExportName:     exportName,
		Kind:           kind,
	}, true
}

func (tc *TypeChecker) bridgeRuntimeUsesCompiledModules() bool {
	if tc == nil {
		return false
	}
	for _, exp := range tc.bridgeRuntime.Manifest.Exports {
		ext := strings.ToLower(filepath.Ext(strings.TrimSpace(exp.ModuleID)))
		if ext == ".js" {
			return true
		}
	}
	return false
}

func (tc *TypeChecker) bridgeCallRuntimeModuleID(sourceModuleID string) string {
	if tc.bridgeRuntimeUsesCompiledModules() {
		return ftconfig.CompiledModuleID(sourceModuleID)
	}
	return sourceModuleID
}

// BridgeModuleForLocal exposes JS import binding lookup for codegen.
func (tc *TypeChecker) BridgeModuleForLocal(local string) (moduleID string, ok bool) {
	mod, ok := tc.nodeModuleForLocal(local)
	if !ok {
		return "", false
	}
	return mod.ModuleID, true
}

// BridgeExportKindFunction is the sync export kind constant for codegen checks.
const BridgeExportKindFunction = bridgeinterop.ExportKindFunction

// BridgeExportKindGenerator is the sync generator export kind.
const BridgeExportKindGenerator = bridgeinterop.ExportKindGenerator

// BridgeExportKindAsyncGenerator is the async generator export kind.
const BridgeExportKindAsyncGenerator = bridgeinterop.ExportKindAsyncGenerator

// BridgeExportKindAsyncFunction is the async export kind constant for codegen checks.
const BridgeExportKindAsyncFunction = bridgeinterop.ExportKindAsyncFunction
