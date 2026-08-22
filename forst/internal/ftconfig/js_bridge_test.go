package ftconfig

import (
	"testing"
)

func TestEffectiveJSBridge_defaultsCompiledNode(t *testing.T) {
	cfg := Default()
	bridge, err := EffectiveJSBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.Host != JSHostNode {
		t.Fatalf("Host: %q", bridge.Host)
	}
	if bridge.ModuleFormat != LegacyModuleCompiled {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
	if bridge.OutDir != ".forst/js" {
		t.Fatalf("OutDir: %q", bridge.OutDir)
	}
}

func TestEffectiveJSBridge_loaderTsxWithBunErrors(t *testing.T) {
	cfg := Default()
	cfg.Javascript.Host = JSHostBun
	cfg.Node.Loader = "tsx"
	_, err := EffectiveJSBridge(cfg)
	if err == nil {
		t.Fatal("expected error for node.loader tsx with bun host")
	}
}

func TestEffectiveJSBridge_loaderTsxMapsToTypeScript(t *testing.T) {
	cfg := Default()
	cfg.Node.Loader = "tsx"
	bridge, err := EffectiveJSBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleTypeScript {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestEffectiveJSBridge_loaderNoneMapsToCompiled(t *testing.T) {
	cfg := Default()
	cfg.Node.Loader = "none"
	bridge, err := EffectiveJSBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleCompiled {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestEffectiveJSBridge_deprecatedArtifactField(t *testing.T) {
	cfg := Default()
	cfg.Javascript.LegacyModules.Artifact = "source"
	bridge, err := EffectiveJSBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleTypeScript {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestEffectiveJSBridge_deprecatedPrecompiledValue(t *testing.T) {
	cfg := Default()
	cfg.Javascript.LegacyModules.Format = "precompiled"
	bridge, err := EffectiveJSBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.ModuleFormat != LegacyModuleCompiled {
		t.Fatalf("ModuleFormat: %q", bridge.ModuleFormat)
	}
}

func TestNeedTsx_compiledJSManifest(t *testing.T) {
	bridge := JSBridge{Host: JSHostNode, ModuleFormat: LegacyModuleCompiled}
	if NeedTsx(bridge, []string{".forst/js/legacy/payment.js"}) {
		t.Fatal("expected no tsx for compiled js manifest")
	}
}

func TestNeedTsx_typeScriptOrTS(t *testing.T) {
	bridge := JSBridge{Host: JSHostNode, ModuleFormat: LegacyModuleCompiled}
	if !NeedTsx(bridge, []string{"legacy/payment.ts"}) {
		t.Fatal("expected tsx when manifest has .ts")
	}
	bridge = JSBridge{Host: JSHostNode, ModuleFormat: LegacyModuleTypeScript}
	if !NeedTsx(bridge, nil) {
		t.Fatal("expected tsx for typescript format on node")
	}
	if NeedTsx(JSBridge{Host: JSHostBun, ModuleFormat: LegacyModuleTypeScript}, []string{"legacy/payment.ts"}) {
		t.Fatal("bun should never need tsx")
	}
	if NeedTsx(JSBridge{Host: JSHostNode, ModuleFormat: LegacyModuleTypeScript}, []string{".forst/js/legacy/payment.js"}) {
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
	if InferHostFromBinary("bun") != JSHostBun {
		t.Fatal("bun")
	}
	if InferHostFromBinary("/opt/homebrew/bin/bun") != JSHostBun {
		t.Fatal("bun path")
	}
	if InferHostFromBinary("node") != JSHostNode {
		t.Fatal("node")
	}
}

func TestEffectiveJSBridge_denoFailsClosed(t *testing.T) {
	cfg := Default()
	cfg.Javascript.Host = JSHostDeno
	_, err := EffectiveJSBridge(cfg)
	if err == nil {
		t.Fatal("expected deno disabled error")
	}
	SetDenoHostEnabledForTest(true)
	defer SetDenoHostEnabledForTest(false)
	bridge, err := EffectiveJSBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.Host != JSHostDeno {
		t.Fatalf("Host: %q", bridge.Host)
	}
}
