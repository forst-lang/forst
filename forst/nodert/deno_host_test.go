package nodert

import (
	"testing"

	"forst/internal/ftconfig"
)

func TestBridgeHost_denoCompiledBootstrapPingAndCallSync(t *testing.T) {
	runBridgeHostBootstrapE2E(t, bridgeHostE2ESpec{
		Host:   ftconfig.JSHostDeno,
		Binary: "deno",
		Format: ftconfig.LegacyModuleCompiled,
		BeforeConfigure: func(t *testing.T) {
			t.Setenv("FORST_DENO_HOST_ENABLED", "1")
		},
	})
}

func TestBridgeHost_denoTypeScriptBootstrapCallSync(t *testing.T) {
	runBridgeHostBootstrapE2E(t, bridgeHostE2ESpec{
		Host:   ftconfig.JSHostDeno,
		Binary: "deno",
		Format: ftconfig.LegacyModuleTypeScript,
		ModuleID: typeScriptLegacyPaymentModuleID,
		LegacySetup: writeTypeScriptLegacyPaymentModule,
		BeforeConfigure: func(t *testing.T) {
			t.Setenv("FORST_DENO_HOST_ENABLED", "1")
		},
	})
}
