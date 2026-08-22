package nodert

import "forst/internal/ftconfig"

func testBridgeNodeTypeScript() ftconfig.JSBridge {
	return ftconfig.JSBridge{
		Host:         ftconfig.JSHostNode,
		ModuleFormat: ftconfig.LegacyModuleTypeScript,
		OutDir:       ".forst/js",
	}
}

func testBridgeNodeCompiled() ftconfig.JSBridge {
	return ftconfig.JSBridge{
		Host:         ftconfig.JSHostNode,
		ModuleFormat: ftconfig.LegacyModuleCompiled,
		OutDir:       ".forst/js",
	}
}

func testBridgeBunTypeScript() ftconfig.JSBridge {
	return ftconfig.JSBridge{
		Host:         ftconfig.JSHostBun,
		ModuleFormat: ftconfig.LegacyModuleTypeScript,
		OutDir:       ".forst/js",
	}
}
