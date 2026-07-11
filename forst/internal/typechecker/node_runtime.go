package typechecker

import (
	"sort"

	"forst/internal/nodeinterop"
)

// NodeRuntimeInfo holds compile-time facts about Node interop for the linked program.
type NodeRuntimeInfo struct {
	NeedsNodeRuntime bool
	Manifest         nodeinterop.ManifestV1
	ManifestJSON     string
}

// NodeRuntimeInfo returns node runtime facts for this typecheck (zero value when not needed).
func (tc *TypeChecker) NodeRuntimeInfo() NodeRuntimeInfo {
	if tc == nil {
		return NodeRuntimeInfo{}
	}
	return tc.nodeRuntime
}

// NeedsNodeRuntime reports whether the linked program contains opted-in TypeScript imports.
func (tc *TypeChecker) NeedsNodeRuntime() bool {
	return tc.NodeRuntimeInfo().NeedsNodeRuntime
}

// SetNodeRuntimeInfo sets node runtime facts (populated by nodeinterop analysis or tests).
func (tc *TypeChecker) SetNodeRuntimeInfo(info NodeRuntimeInfo) {
	if tc == nil {
		return
	}
	tc.nodeRuntime = info
}

// NodeRuntimeSummary returns module count, export count, and sorted module IDs for CLI output.
func (tc *TypeChecker) NodeRuntimeSummary() (modules, exports int, moduleIDs []string) {
	if tc == nil || !tc.NeedsNodeRuntime() {
		return 0, 0, nil
	}
	state := tc.NodeRuntimeState()
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
		modules = tc.NodeImportCount()
	}
	return modules, exports, moduleIDs
}
