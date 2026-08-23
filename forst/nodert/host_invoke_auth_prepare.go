package nodert

import (
	"os"

	"forst/internal/ftconfig"
)

// PrepareEmbeddedHostInvokeAuthRelay loads ftconfig from the boundary root and wires invoke auth relay.
func PrepareEmbeddedHostInvokeAuthRelay() error {
	root := ftconfig.RootFromEnv()
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return bridgeRuntimeErr("getwd: %w", err)
		}
		var discoverErr error
		root, discoverErr = ftconfig.BoundaryRootFromDir(cwd)
		if discoverErr != nil {
			return bridgeRuntimeErr("discover boundary root: %w", discoverErr)
		}
	}
	cfg, err := ftconfig.LoadFromDir(root)
	if err != nil {
		return bridgeRuntimeErr("load ftconfig: %w", err)
	}
	return EnsureEmbeddedHostInvokeAuthRelay(cfg)
}
