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

type nodeRuntimeLogSpy struct {
	info  []string
	debug []string
}

func (s *nodeRuntimeLogSpy) Info(args ...any)  { s.info = append(s.info, args[0].(string)) }
func (s *nodeRuntimeLogSpy) Debug(args ...any) { s.debug = append(s.debug, args[0].(string)) }

func TestLogNodeRuntimeRequirement_notRequiredUsesDebug(t *testing.T) {
	t.Parallel()
	spy := &nodeRuntimeLogSpy{}
	logNodeRuntimeRequirement(spy, typechecker.New(nil, false))
	if len(spy.info) != 0 {
		t.Fatalf("info = %v, want none", spy.info)
	}
	if len(spy.debug) != 1 || spy.debug[0] != "node runtime: not required" {
		t.Fatalf("debug = %v", spy.debug)
	}
}

func TestLogNodeRuntimeRequirement_requiredUsesInfo(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetNodeRuntimeInfo(typechecker.NodeRuntimeInfo{NeedsNodeRuntime: true})
	spy := &nodeRuntimeLogSpy{}
	logNodeRuntimeRequirement(spy, tc)
	if len(spy.debug) != 0 {
		t.Fatalf("debug = %v, want none", spy.debug)
	}
	if len(spy.info) != 1 || !strings.HasPrefix(spy.info[0], "node runtime: required") {
		t.Fatalf("info = %v", spy.info)
	}
}
