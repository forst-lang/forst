package nodeinterop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const ManifestVersionV1 = 1

// Export kind constants for forst-node-manifest-v1.
const (
	ExportKindFunction       = "function"
	ExportKindAsyncFunction  = "asyncFunction"
	ExportKindGenerator      = "generator"
	ExportKindAsyncGenerator = "asyncGenerator"
)

var validExportKinds = map[string]struct{}{
	ExportKindFunction:       {},
	ExportKindAsyncFunction:  {},
	ExportKindGenerator:      {},
	ExportKindAsyncGenerator: {},
}

// ManifestV1 is the compile-time execution allowlist embedded when needsNodeRuntime is true.
type ManifestV1 struct {
	Version      int           `json:"version"`
	BoundaryRoot string        `json:"boundaryRoot"`
	Exports      []ExportEntry `json:"exports"`
}

// ExportEntry describes one callable TypeScript export the bridge may invoke.
type ExportEntry struct {
	ModuleID string `json:"moduleId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// Validate checks manifest invariants from the node interop spec.
func (m *ManifestV1) Validate() error {
	if m == nil {
		return fmt.Errorf("manifest: nil")
	}
	if m.Version != ManifestVersionV1 {
		return fmt.Errorf("manifest: version must be %d, got %d", ManifestVersionV1, m.Version)
	}
	if strings.TrimSpace(m.BoundaryRoot) == "" {
		return fmt.Errorf("manifest: boundaryRoot is required")
	}
	if !filepath.IsAbs(m.BoundaryRoot) {
		return fmt.Errorf("manifest: boundaryRoot must be absolute")
	}
	seen := make(map[string]struct{}, len(m.Exports))
	for i, exp := range m.Exports {
		if err := exp.validate(); err != nil {
			return fmt.Errorf("manifest: exports[%d]: %w", i, err)
		}
		key := exp.ModuleID + "\x00" + exp.Name
		if _, dup := seen[key]; dup {
			return fmt.Errorf("manifest: duplicate export %q in %q", exp.Name, exp.ModuleID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (e ExportEntry) validate() error {
	if err := validateModuleID(e.ModuleID); err != nil {
		return err
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if _, ok := validExportKinds[e.Kind]; !ok {
		return fmt.Errorf("invalid kind %q", e.Kind)
	}
	return nil
}

func validateModuleID(moduleID string) error {
	if strings.TrimSpace(moduleID) == "" {
		return fmt.Errorf("moduleId is required")
	}
	if filepath.IsAbs(moduleID) {
		return fmt.Errorf("moduleId must be project-relative")
	}
	if strings.Contains(moduleID, "..") {
		return fmt.Errorf("moduleId must not contain ..")
	}
	if strings.HasPrefix(moduleID, "file://") {
		return fmt.Errorf("moduleId must not be a file URL")
	}
	ext := strings.ToLower(filepath.Ext(moduleID))
	switch ext {
	case ".ts", ".tsx", ".js":
	default:
		return fmt.Errorf("moduleId extension must be .ts, .tsx, or .js")
	}
	return nil
}

// EmbeddedManifestV1 is the portable subset embedded in generated Go (no boundaryRoot).
type EmbeddedManifestV1 struct {
	Version int           `json:"version"`
	Exports []ExportEntry `json:"exports"`
}

// ValidateEmbedded checks version and exports without requiring boundaryRoot.
func (m *EmbeddedManifestV1) ValidateEmbedded() error {
	if m == nil {
		return fmt.Errorf("manifest: nil")
	}
	if m.Version != ManifestVersionV1 {
		return fmt.Errorf("manifest: version must be %d, got %d", ManifestVersionV1, m.Version)
	}
	seen := make(map[string]struct{}, len(m.Exports))
	for i, exp := range m.Exports {
		if err := exp.validate(); err != nil {
			return fmt.Errorf("manifest: exports[%d]: %w", i, err)
		}
		key := exp.ModuleID + "\x00" + exp.Name
		if _, dup := seen[key]; dup {
			return fmt.Errorf("manifest: duplicate export %q in %q", exp.Name, exp.ModuleID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// EmbeddedManifestJSON returns deterministic compact JSON for code generation.
// Exports are sorted by moduleId, name, then kind; boundaryRoot is omitted.
func (m *ManifestV1) EmbeddedManifestJSON() ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest: nil")
	}
	embedded := EmbeddedManifestV1{
		Version: m.Version,
		Exports: append([]ExportEntry(nil), m.Exports...),
	}
	if err := embedded.ValidateEmbedded(); err != nil {
		return nil, err
	}
	sortExports(embedded.Exports)
	return encodeManifestJSON(embedded)
}

// CanonicalJSON returns deterministic compact JSON for hashing and cross-language contract tests.
// Exports are sorted by moduleId, name, then kind.
func (m *ManifestV1) CanonicalJSON() ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest: nil")
	}
	canonical := *m
	canonical.Exports = append([]ExportEntry(nil), m.Exports...)
	sortExports(canonical.Exports)
	return encodeManifestJSON(canonical)
}

func sortExports(exports []ExportEntry) {
	sort.Slice(exports, func(i, j int) bool {
		a, b := exports[i], exports[j]
		if a.ModuleID != b.ModuleID {
			return a.ModuleID < b.ModuleID
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Kind < b.Kind
	})
}

func encodeManifestJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out, nil
}

// ParseManifestV1 parses and validates JSON manifest bytes.
func ParseManifestV1(data []byte) (*ManifestV1, error) {
	var m ManifestV1
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: parse: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}
