package typechecker

import (
	"sort"

	"forst/internal/bridgeinterop"
)

// BridgeRuntimeInfo holds compile-time facts about bridge interop for the linked program.
type BridgeRuntimeInfo struct {
	NeedsBridgeRuntime bool
	Manifest         bridgeinterop.ManifestV1
	ManifestJSON     string
}

// BridgeRuntimeInfo returns bridge runtime facts for this typecheck (zero value when not needed).
func (tc *TypeChecker) BridgeRuntimeInfo() BridgeRuntimeInfo {
	if tc == nil {
		return BridgeRuntimeInfo{}
	}
	return tc.bridgeRuntime
}

// NeedsBridgeRuntime reports whether the linked program contains opted-in TypeScript imports.
func (tc *TypeChecker) NeedsBridgeRuntime() bool {
	return tc.BridgeRuntimeInfo().NeedsBridgeRuntime
}

// SetBridgeRuntimeInfo sets bridge runtime facts (populated by bridgeinterop analysis or tests).
func (tc *TypeChecker) SetBridgeRuntimeInfo(info BridgeRuntimeInfo) {
	if tc == nil {
		return
	}
	tc.bridgeRuntime = info
}

// BridgeRuntimeSummary returns module count, export count, and sorted module IDs for CLI output.
func (tc *TypeChecker) BridgeRuntimeSummary() (modules, exports int, moduleIDs []string) {
	if tc == nil || !tc.NeedsBridgeRuntime() {
		return 0, 0, nil
	}
	state := tc.BridgeRuntimeState()
	exports = len(state.Manifest.Exports)
	seen := make(map[string]struct{})
	for _, exp := range state.Manifest.Exports {
		if exp.ModuleID == "" {
			continue
		}
		if _, ok := seen[exp.ModuleID]; ok {
			continue
		}
		seen[exp.ModuleID] = struct{}{}
		moduleIDs = append(moduleIDs, exp.ModuleID)
	}
	sort.Strings(moduleIDs)
	modules = len(moduleIDs)
	if modules == 0 {
		modules = tc.BridgeImportCount()
	}
	return modules, exports, moduleIDs
}
