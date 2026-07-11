package compiler

import (
	"strings"
	"testing"

	"forst/internal/nodeinterop"
	"forst/internal/typechecker"
)

func TestRequireNoNode_allowsWhenNoNodeRuntime(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	if err := checkRequireNoNode(Args{RequireNoNode: true}, tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireNoNode_rejectsWhenNeedsNodeRuntime(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetNodeRuntimeInfo(typechecker.NodeRuntimeInfo{NeedsNodeRuntime: true})
	err := checkRequireNoNode(Args{RequireNoNode: true}, tc)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "require-no-node") {
		t.Fatalf("error = %v", err)
	}
}

func TestRequireNoNode_ignoredWhenFlagUnset(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetNodeRuntimeInfo(typechecker.NodeRuntimeInfo{NeedsNodeRuntime: true})
	if err := checkRequireNoNode(Args{}, tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatNodeRuntimeLogLine_notRequired(t *testing.T) {
	t.Parallel()
	if got := FormatNodeRuntimeLogLine(typechecker.New(nil, false)); got != "node runtime: not required" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatNodeRuntimeLogLine_requiredWithModules(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetNodeRuntimeInfo(typechecker.NodeRuntimeInfo{
		NeedsNodeRuntime: true,
		Manifest: nodeinterop.ManifestV1{
			Exports: []nodeinterop.ExportEntry{
				{ModuleID: "legacy/payment.ts", Name: "create", Kind: "asyncFunction"},
				{ModuleID: "legacy/payment.ts", Name: "refund", Kind: "function"},
				{ModuleID: "legacy/events.ts", Name: "emit", Kind: "function"},
			},
		},
	})
	got := FormatNodeRuntimeLogLine(tc)
	want := "node runtime: required (2 modules, 3 exports) — legacy/events.ts, legacy/payment.ts"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
