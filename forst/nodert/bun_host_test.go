package nodert

import (
	"testing"

	"forst/internal/ftconfig"
)

func TestBridgeHost_bunCompiledBootstrapPingAndCallSync(t *testing.T) {
	runBridgeHostBootstrapE2E(t, bridgeHostE2ESpec{
		Host:   ftconfig.JSHostBun,
		Binary: "bun",
		Format: ftconfig.LegacyModuleCompiled,
	})
}

func TestBridgeHost_bunTypeScriptBootstrapCallSync(t *testing.T) {
	runBridgeHostBootstrapE2E(t, bridgeHostE2ESpec{
		Host:        ftconfig.JSHostBun,
		Binary:      "bun",
		Format:      ftconfig.LegacyModuleTypeScript,
		ModuleID:    typeScriptLegacyPaymentModuleID,
		LegacySetup: writeTypeScriptLegacyPaymentModule,
	})
}
