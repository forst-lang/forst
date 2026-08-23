package typechecker

import (
	"testing"

	"forst/internal/bridgeinterop"
)

func TestBridgeRuntimeSummary_deduplicatesAndSortsModuleIDs(t *testing.T) {
	tc := &TypeChecker{}
	tc.SetBridgeRuntimeInfo(BridgeRuntimeInfo{
		NeedsBridgeRuntime: true,
		Manifest: bridgeinterop.ManifestV1{
			Exports: []bridgeinterop.ExportEntry{
				{ModuleID: "legacy/b.ts", Name: "mul", Kind: bridgeinterop.ExportKindFunction},
				{ModuleID: "legacy/a.ts", Name: "sub", Kind: bridgeinterop.ExportKindFunction},
				{ModuleID: "legacy/b.ts", Name: "add", Kind: bridgeinterop.ExportKindFunction},
			},
		},
	})

	modules, exports, ids := tc.BridgeRuntimeSummary()
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

func TestBridgeRuntimeSummary_zeroWhenNotNeeded(t *testing.T) {
	tc := &TypeChecker{}
	modules, exports, ids := tc.BridgeRuntimeSummary()
	if modules != 0 || exports != 0 || ids != nil {
		t.Fatalf("summary = (%d, %d, %v)", modules, exports, ids)
	}
}

func TestBridgeRuntimeSummary_nilTypeChecker(t *testing.T) {
	var tc *TypeChecker
	modules, exports, ids := tc.BridgeRuntimeSummary()
	if modules != 0 || exports != 0 || ids != nil {
		t.Fatalf("summary = (%d, %d, %v)", modules, exports, ids)
	}
}
