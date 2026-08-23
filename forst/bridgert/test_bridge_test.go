package bridgert

import "forst/internal/ftconfig"

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
