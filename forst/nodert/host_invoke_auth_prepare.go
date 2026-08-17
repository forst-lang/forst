package nodert

import (
	"fmt"
	"os"
	"strings"

	"forst/internal/ftconfig"
)

// PrepareEmbeddedHostInvokeAuthRelay loads ftconfig from the boundary root and wires invoke auth relay.
func PrepareEmbeddedHostInvokeAuthRelay() error {
	root := strings.TrimSpace(os.Getenv(EnvBoundaryRoot))
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("node runtime: getwd: %w", err)
		}
		var discoverErr error
		root, discoverErr = ftconfig.BoundaryRootFromDir(cwd)
		if discoverErr != nil {
			return fmt.Errorf("node runtime: discover boundary root: %w", discoverErr)
		}
	}
	cfg, err := ftconfig.LoadFromDir(root)
	if err != nil {
		return fmt.Errorf("node runtime: load ftconfig: %w", err)
	}
	return EnsureEmbeddedHostInvokeAuthRelay(cfg)
}
