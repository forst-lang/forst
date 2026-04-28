package goload

import (
	"os"
	"path/filepath"
)

// GoWorkspaceForPackages returns the directory used as go/packages Config.Dir so imports like
// "forst/gateway" resolve. Prefer the embedded Forst SDK module when walking upward from start
// (file or directory); otherwise fall back to FindModuleRoot.
func GoWorkspaceForPackages(start string) string {
	startDir := start
	if fi, err := os.Stat(start); err == nil && !fi.IsDir() {
		startDir = filepath.Dir(start)
	}
	if sdk, err := ForstSDKModuleRoot(startDir); err == nil {
		return sdk
	}
	return FindModuleRoot(startDir)
}
