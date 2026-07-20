package typechecker

import (
	"testing"

	"forst/internal/nodeinterop"
)

func TestNodeRuntimeSummary_deduplicatesAndSortsModuleIDs(t *testing.T) {
	tc := &TypeChecker{}
	tc.SetNodeRuntimeInfo(NodeRuntimeInfo{
		NeedsNodeRuntime: true,
		Manifest: nodeinterop.ManifestV1{
			Exports: []nodeinterop.ExportEntry{
				{ModuleID: "legacy/b.ts", Name: "mul", Kind: nodeinterop.ExportKindFunction},
				{ModuleID: "legacy/a.ts", Name: "sub", Kind: nodeinterop.ExportKindFunction},
				{ModuleID: "legacy/b.ts", Name: "add", Kind: nodeinterop.ExportKindFunction},
			},
		},
	})

	modules, exports, ids := tc.NodeRuntimeSummary()
	if modules != 2 {
		t.Fatalf("modules = %d want 2", modules)
	}
	if exports != 3 {
		t.Fatalf("exports = %d want 3", exports)
	}
	if len(ids) != 2 || ids[0] != "legacy/a.ts" || ids[1] != "legacy/b.ts" {
		t.Fatalf("moduleIDs = %v", ids)
	}
}

func TestNodeRuntimeSummary_zeroWhenNotNeeded(t *testing.T) {
	tc := &TypeChecker{}
	modules, exports, ids := tc.NodeRuntimeSummary()
	if modules != 0 || exports != 0 || ids != nil {
		t.Fatalf("summary = (%d, %d, %v)", modules, exports, ids)
	}
}

func TestNodeRuntimeSummary_nilTypeChecker(t *testing.T) {
	var tc *TypeChecker
	modules, exports, ids := tc.NodeRuntimeSummary()
	if modules != 0 || exports != 0 || ids != nil {
		t.Fatalf("summary = (%d, %d, %v)", modules, exports, ids)
	}
}
