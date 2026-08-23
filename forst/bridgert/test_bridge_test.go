package bridgert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ftconfig"
)

func testBridgeNodeTypeScript() ftconfig.Bridge {
	return ftconfig.Bridge{
		Host:         ftconfig.BridgeHostNode,
		ModuleFormat: ftconfig.LegacyModuleTypeScript,
		OutDir:       ".forst/js",
	}
}

func testBridgeNodeCompiled() ftconfig.Bridge {
	return ftconfig.Bridge{
		Host:         ftconfig.BridgeHostNode,
		ModuleFormat: ftconfig.LegacyModuleCompiled,
		OutDir:       ".forst/js",
	}
}

func testBridgeBunTypeScript() ftconfig.Bridge {
	return ftconfig.Bridge{
		Host:         ftconfig.BridgeHostBun,
		ModuleFormat: ftconfig.LegacyModuleTypeScript,
		OutDir:       ".forst/js",
	}
}

func writeBridgeTypeScriptFtconfig(t *testing.T, root string) {
	t.Helper()
	cfg := ftconfig.Default()
	cfg.Bridge.LegacyModules.Format = ftconfig.LegacyModuleTypeScript
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
