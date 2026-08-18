package ftconfig

import (
	"fmt"
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

// ResolveCmd resolves cmd against boundaryRoot when relative; otherwise returns cmd unchanged.
func (p GeneratePluginConfig) ResolveCmd(boundaryRoot string) string {
	cmd := strings.TrimSpace(p.Cmd)
	if cmd == "" || filepath.IsAbs(cmd) {
		return cmd
	}
	return filepath.Join(filepath.Clean(boundaryRoot), cmd)
}
