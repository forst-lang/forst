package ftconfig

import (
	"testing"
)

func TestEffectiveBridge_defaultsCompiledNode(t *testing.T) {
	cfg := Default()
	bridge, err := EffectiveBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.Host != BridgeHostNode {
		t.Fatalf("Host: %q", bridge.Host)
	}
	if bridge.ModuleFormat != LegacyModuleCompiled {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
	if bridge.OutDir != ".forst/js" {
		t.Fatalf("OutDir: %q", bridge.OutDir)
	}
}

func TestEffectiveBridge_typescriptFormatWithBun(t *testing.T) {
	cfg := Default()
	cfg.Bridge.Host = BridgeHostBun
	cfg.Bridge.LegacyModules.Format = LegacyModuleTypeScript
	bridge, err := EffectiveBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleTypeScript {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestEffectiveBridge_legacyModulesFormatTypeScript(t *testing.T) {
	cfg := Default()
	cfg.Bridge.LegacyModules.Format = LegacyModuleTypeScript
	bridge, err := EffectiveBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleTypeScript {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestEffectiveBridge_legacyModulesFormatCompiled(t *testing.T) {
	cfg := Default()
	cfg.Bridge.LegacyModules.Format = LegacyModuleCompiled
	bridge, err := EffectiveBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleCompiled {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestEffectiveBridge_precompiledAliasValue(t *testing.T) {
	cfg := Default()
	cfg.Bridge.LegacyModules.Format = "precompiled"
	bridge, err := EffectiveBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleCompiled {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestNeedTsx_compiledJSManifest(t *testing.T) {
	bridge := Bridge{Host: BridgeHostNode, ModuleFormat: LegacyModuleCompiled}
	if NeedTsx(bridge, []string{".forst/js/legacy/payment.js"}) {
		t.Fatal("expected no tsx for compiled js manifest")
	}
}

func TestNeedTsx_typeScriptOrTS(t *testing.T) {
	bridge := Bridge{Host: BridgeHostNode, ModuleFormat: LegacyModuleCompiled}
	if !NeedTsx(bridge, []string{"legacy/payment.ts"}) {
		t.Fatal("expected tsx when manifest has .ts")
	}
	bridge = Bridge{Host: BridgeHostNode, ModuleFormat: LegacyModuleTypeScript}
	if !NeedTsx(bridge, nil) {
		t.Fatal("expected tsx for typescript format on node")
	}
	if NeedTsx(Bridge{Host: BridgeHostBun, ModuleFormat: LegacyModuleTypeScript}, []string{"legacy/payment.ts"}) {
		t.Fatal("bun should never need tsx")
	}
	if NeedTsx(Bridge{Host: BridgeHostNode, ModuleFormat: LegacyModuleTypeScript}, []string{".forst/js/legacy/payment.js"}) {
		t.Fatal("typescript format with only .js manifest should not need tsx")
	}
}

func TestRuntimeModuleID_compiled(t *testing.T) {
	got := RuntimeModuleID("legacy/payment.ts", ".forst/js", LegacyModuleCompiled)
	want := ".forst/js/legacy/payment.js"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestInferHostFromBinary(t *testing.T) {
	if InferHostFromBinary("bun") != BridgeHostBun {
		t.Fatal("bun")
	}
	if InferHostFromBinary("/opt/homebrew/bin/bun") != BridgeHostBun {
		t.Fatal("bun path")
	}
	if InferHostFromBinary("node") != BridgeHostNode {
		t.Fatal("node")
	}
}

func TestEffectiveBridge_denoFailsClosed(t *testing.T) {
	cfg := Default()
	cfg.Bridge.Host = BridgeHostDeno
	_, err := EffectiveBridge(cfg)
	if err == nil {
		t.Fatal("expected deno disabled error")
	}
	SetDenoHostEnabledForTest(true)
	defer SetDenoHostEnabledForTest(false)
	bridge, err := EffectiveBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.Host != BridgeHostDeno {
		t.Fatalf("Host: %q", bridge.Host)
	}
}
