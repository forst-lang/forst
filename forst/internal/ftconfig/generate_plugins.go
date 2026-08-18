package ftconfig

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Validate checks one generate.plugins entry.
func (p GeneratePluginConfig) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(p.Cmd) == "" {
		return fmt.Errorf("cmd is required")
	}
	if strings.TrimSpace(p.Out) == "" {
		return fmt.Errorf("out is required")
	}
	if filepath.IsAbs(p.Out) {
		return fmt.Errorf("out must be relative to the boundary root, got absolute path %q", p.Out)
	}
	cleaned := filepath.Clean(p.Out)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("out %q escapes the boundary root", p.Out)
	}
	return nil
}

// EffectiveOutDir resolves plugin out against boundaryRoot.
func (p GeneratePluginConfig) EffectiveOutDir(boundaryRoot string) string {
	return filepath.Join(filepath.Clean(boundaryRoot), filepath.Clean(p.Out))
}

// ResolveCmd resolves a plugin executable path.
// Absolute cmd is returned unchanged. Paths containing a separator (including ./bin/foo)
// are resolved against boundaryRoot. Bare names are looked up on PATH.
func (p GeneratePluginConfig) ResolveCmd(boundaryRoot string) (string, error) {
	cmd := strings.TrimSpace(p.Cmd)
	if cmd == "" {
		return "", fmt.Errorf("cmd is required")
	}
	if filepath.IsAbs(cmd) {
		return cmd, nil
	}
	if strings.Contains(cmd, string(filepath.Separator)) || strings.HasPrefix(cmd, ".") {
		return filepath.Join(filepath.Clean(boundaryRoot), cmd), nil
	}
	path, err := exec.LookPath(cmd)
	if err != nil {
		return "", fmt.Errorf("plugin cmd %q not found on PATH: %w", cmd, err)
	}
	return path, nil
}
