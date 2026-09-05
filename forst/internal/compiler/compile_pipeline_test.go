package compiler

import (
	"strings"
	"testing"

	"forst/internal/bridgeinterop"
	"forst/internal/typechecker"
)

func TestRequireNoBridge_allowsWhenNoBridgeRuntime(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	if err := checkRequireNoBridge(Args{RequireNoBridge: true}, tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireNoBridge_rejectsWhenNeedsBridgeRuntime(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetBridgeRuntimeInfo(typechecker.BridgeRuntimeInfo{NeedsBridgeRuntime: true})
	err := checkRequireNoBridge(Args{RequireNoBridge: true}, tc)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "require-no-bridge") {
		t.Fatalf("error = %v", err)
	}
}

func TestRequireNoBridge_ignoredWhenFlagUnset(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetBridgeRuntimeInfo(typechecker.BridgeRuntimeInfo{NeedsBridgeRuntime: true})
	if err := checkRequireNoBridge(Args{}, tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatBridgeRuntimeLogLine_notRequired(t *testing.T) {
	t.Parallel()
	if got := FormatBridgeRuntimeLogLine(typechecker.New(nil, false)); got != "bridge runtime: not required" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatBridgeRuntimeLogLine_requiredWithModules(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetBridgeRuntimeInfo(typechecker.BridgeRuntimeInfo{
		NeedsBridgeRuntime: true,
		Manifest: bridgeinterop.ManifestV1{
			Exports: []bridgeinterop.ExportEntry{
				{ModuleID: "legacy/payment.ts", Name: "create", Kind: "asyncFunction"},
				{ModuleID: "legacy/payment.ts", Name: "refund", Kind: "function"},
				{ModuleID: "legacy/events.ts", Name: "emit", Kind: "function"},
			},
		},
	})
	got := FormatBridgeRuntimeLogLine(tc)
	want := "bridge runtime: required (2 modules, 3 exports) — legacy/events.ts, legacy/payment.ts"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

type bridgeRuntimeLogSpy struct {
	info  []string
	debug []string
}

func (s *bridgeRuntimeLogSpy) Info(args ...any)  { s.info = append(s.info, args[0].(string)) }
func (s *bridgeRuntimeLogSpy) Debug(args ...any) { s.debug = append(s.debug, args[0].(string)) }

func TestLogBridgeRuntimeRequirement_notRequiredUsesDebug(t *testing.T) {
	t.Parallel()
	spy := &bridgeRuntimeLogSpy{}
	logBridgeRuntimeRequirement(spy, typechecker.New(nil, false))
	if len(spy.info) != 0 {
		t.Fatalf("info = %v, want none", spy.info)
	}
	if len(spy.debug) != 1 || spy.debug[0] != "bridge runtime: not required" {
		t.Fatalf("debug = %v", spy.debug)
	}
}

func TestLogBridgeRuntimeRequirement_requiredUsesInfo(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetBridgeRuntimeInfo(typechecker.BridgeRuntimeInfo{NeedsBridgeRuntime: true})
	spy := &bridgeRuntimeLogSpy{}
	logBridgeRuntimeRequirement(spy, tc)
	if len(spy.debug) != 0 {
		t.Fatalf("debug = %v, want none", spy.debug)
	}
	if len(spy.info) != 1 || !strings.HasPrefix(spy.info[0], "bridge runtime: required") {
		t.Fatalf("info = %v", spy.info)
	}
}
