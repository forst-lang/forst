package goload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ForstSDKModuleRoot returns the absolute path to the directory containing the Forst
// compiler Go module (module path "forst"), walking upward from startDir.
// Used to inject `replace forst => <path>` in executor temp modules so imports like
// forst/gateway resolve during `go run`.
func ForstSDKModuleRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for i := 0; i < 64; i++ {
		modPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modPath); err == nil {
			s := string(data)
			if strings.Contains(s, "module forst") {
				return dir, nil
			}
		}
		nested := filepath.Join(dir, "forst", "go.mod")
		if data, err := os.ReadFile(nested); err == nil && strings.Contains(string(data), "module forst") {
			return filepath.Join(dir, "forst"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not locate Forst SDK module (module forst) starting from %s", startDir)
}
