package nodert

import (
	"fmt"
	"path"
	"strings"
)

// ExportKind identifies callable export shapes from the compile-time index.
type ExportKind string

const (
	ExportKindFunction       ExportKind = "function"
	ExportKindAsyncFunction  ExportKind = "asyncFunction"
	ExportKindGenerator      ExportKind = "generator"
	ExportKindAsyncGenerator ExportKind = "asyncGenerator"
)

// Manifest is the compile-time allowlist embedded in binaries that need Node.
type Manifest struct {
	Version      int           `json:"version"`
	BoundaryRoot string        `json:"boundaryRoot"`
	Exports      []ExportEntry `json:"exports"`
}

// ExportEntry is one callable export allowed at runtime.
type ExportEntry struct {
	ModuleID string     `json:"moduleId"`
	Name     string     `json:"name"`
	Kind     ExportKind `json:"kind"`
}

// ValidateEmbedded checks version and exports for compile-time embedded manifests (no boundaryRoot).
func (m Manifest) ValidateEmbedded() error {
	if m.Version != ManifestVersion {
		return fmt.Errorf("unsupported manifest version %d", m.Version)
	}
	if len(m.Exports) == 0 {
		return fmt.Errorf("manifest exports must not be empty")
	}
	for i, exp := range m.Exports {
		if err := ValidateModuleID(exp.ModuleID); err != nil {
			return fmt.Errorf("exports[%d]: %w", i, err)
		}
		if strings.TrimSpace(exp.Name) == "" {
			return fmt.Errorf("exports[%d]: export name is required", i)
		}
		if !isKnownExportKind(exp.Kind) {
			return fmt.Errorf("exports[%d]: unknown export kind %q", i, exp.Kind)
		}
	}
	return nil
}

// Validate checks manifest shape before initialize or call preflight.
func (m Manifest) Validate() error {
	if err := m.ValidateEmbedded(); err != nil {
		return err
	}
	if strings.TrimSpace(m.BoundaryRoot) == "" {
		return fmt.Errorf("manifest boundaryRoot is required")
	}
	return nil
}

// AllowCall validates a call against the manifest allowlist before RPC send.
func (m Manifest) AllowCall(moduleID, exportName string, kind ExportKind) error {
	if err := ValidateModuleID(moduleID); err != nil {
		return err
	}
	for _, exp := range m.Exports {
		if exp.ModuleID == moduleID && exp.Name == exportName {
			if kind != "" && exp.Kind != kind {
				return fmt.Errorf("%w: export %q kind is %q, expected %q", ErrForbidden, exportName, exp.Kind, kind)
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %s:%s", ErrForbidden, moduleID, exportName)
}

// ValidateModuleID enforces project-relative POSIX path rules for RPC payloads.
func ValidateModuleID(moduleID string) error {
	if strings.TrimSpace(moduleID) == "" {
		return fmt.Errorf("%w: empty moduleId", ErrInvalidModuleID)
	}
	if strings.HasPrefix(moduleID, "/") || strings.HasPrefix(moduleID, "\\") {
		return fmt.Errorf("%w: absolute path %q", ErrInvalidModuleID, moduleID)
	}
	if strings.Contains(moduleID, "://") || strings.HasPrefix(moduleID, "file:") {
		return fmt.Errorf("%w: URL moduleId %q", ErrInvalidModuleID, moduleID)
	}
	clean := path.Clean(moduleID)
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("%w: path escapes boundary %q", ErrInvalidModuleID, moduleID)
	}
	ext := path.Ext(clean)
	switch ext {
	case ".ts", ".tsx", ".js":
	default:
		return fmt.Errorf("%w: unsupported extension in %q", ErrInvalidModuleID, moduleID)
	}
	if strings.Contains(clean, "node_modules/") {
		return fmt.Errorf("%w: node_modules path %q", ErrInvalidModuleID, moduleID)
	}
	return nil
}

func isKnownExportKind(kind ExportKind) bool {
	switch kind {
	case ExportKindFunction, ExportKindAsyncFunction, ExportKindGenerator, ExportKindAsyncGenerator:
		return true
	default:
		return false
	}
}
