package programbuild

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// ManifestFileName is the build output manifest basename.
	ManifestFileName = "manifest.json"
	// BinDir is the relative directory for linked program binaries under -o.
	BinDir = "bin"
	// KindProgram is manifest.json kind for forst build -o output.
	KindProgram = "program"
	// SchemaVersion is the current manifest.json schemaVersion field.
	SchemaVersion = 1
)

// ContractVersion is the invoke HTTP contract recorded in manifest.json.
// Keep in sync with invokeserver.HTTPContractVersion.
const ContractVersion = "2"

// ProgramManifest describes a native program build output directory from forst build -o.
// Kind is the embedded program binary (entry + invoke + optional Node host), not a sidecar.
// Binary is the relative path to the linked executable under the -o directory (from entry stem).
// HostMode reflects ftconfig bridge.hostMode at build time for CI layout checks.
type ProgramManifest struct {
	SchemaVersion     int      `json:"schemaVersion"`
	Kind              string   `json:"kind"`
	CompilerVersion   string   `json:"compilerVersion"`
	ContractVersion   string   `json:"contractVersion"`
	Entry             string   `json:"entry"`
	BoundaryRoot      string   `json:"boundaryRoot"`
	GOOS              string   `json:"goos"`
	GOARCH            string   `json:"goarch"`
	EmbeddedInvoke    bool     `json:"embeddedInvoke"`
	HostMode          bool     `json:"hostMode"`
	SkipNodeHostDefault bool     `json:"skipNodeHostDefault"`
	Packages          []string `json:"packages"`
	Binary            string   `json:"binary"`
	BuiltAt           string   `json:"builtAt"`
}

// Validate checks required manifest fields for a program build artifact.
func (m ProgramManifest) Validate() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("manifest schemaVersion = %d want %d", m.SchemaVersion, SchemaVersion)
	}
	if m.Kind != KindProgram {
		return fmt.Errorf("manifest kind = %q want %q", m.Kind, KindProgram)
	}
	if strings.TrimSpace(m.Binary) == "" {
		return fmt.Errorf("manifest binary is empty")
	}
	return nil
}

// Load reads manifest.json from a forst build -o directory.
func Load(outputDir string) (ProgramManifest, error) {
	data, err := os.ReadFile(filepath.Join(outputDir, ManifestFileName))
	if err != nil {
		return ProgramManifest{}, fmt.Errorf("read %s: %w", ManifestFileName, err)
	}
	var manifest ProgramManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ProgramManifest{}, fmt.Errorf("parse %s: %w", ManifestFileName, err)
	}
	if err := manifest.Validate(); err != nil {
		return ProgramManifest{}, err
	}
	return manifest, nil
}

// Write marshals manifest to manifest.json under outputDir.
func Write(outputDir string, manifest ProgramManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	manifestPath := filepath.Join(outputDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", ManifestFileName, err)
	}
	return nil
}
